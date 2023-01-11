package tweets

import (
	"encoding/json"
	"fmt"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/fallenstedt/twitter-stream"
	"github.com/fallenstedt/twitter-stream/rules"
	"github.com/fallenstedt/twitter-stream/token_generator"
	"github.com/jchavannes/jgo/jerr"
	"github.com/jchavannes/jgo/jlog"
	"github.com/memocash/tweet/config"
	"github.com/memocash/tweet/database"
	"github.com/memocash/tweet/tweets/obj"
	"github.com/syndtr/goleveldb/leveldb"
	"regexp"
	"strconv"
)

type Stream struct {
	Api   *twitterstream.TwitterApi
	Db    *leveldb.DB
	Token *token_generator.RequestBearerTokenResponse
}

func NewStream(db *leveldb.DB) (*Stream, error) {
	conf := config.GetTwitterAPIConfig()
	if !conf.IsSet() {
		return nil, jerr.New("Application Access Token required")
	}
	token, err := twitterstream.NewTokenGenerator().SetApiKeyAndSecret(conf.ConsumerKey, conf.ConsumerSecret).RequestBearerToken()
	if err != nil {
		return nil, jerr.Get("error getting twitter API token", err)
	}
	return &Stream{
		Db:    db,
		Token: token,
	}, nil
}

func (s *Stream) CloseApi() {
	if s.Api != nil {
		jlog.Log("Stopping Twitter API stream")
		s.Api.Stream.StopStream()
	}
	s.Api = nil
}

func (s *Stream) SetFreshApi() {
	s.CloseApi()
	s.Api = twitterstream.NewTwitterStream(s.Token.AccessToken)
}

func (s *Stream) FilterAccount(streamConfigs []config.Stream) error {
	/*s.SetFreshApi()
	defer s.CloseApi()*/
	var res *rules.TwitterRuleResponse
	var err error
	//remove duplicate names from stream configs
	uniqueNames := make(map[string]bool)
	for _, streamConfig := range streamConfigs {
		uniqueNames[streamConfig.Name] = true
	}
	for name := range uniqueNames {
		streamRules := twitterstream.NewRuleBuilder().AddRule("from:"+name, "get tweets from this account").Build()
		res, err = s.Api.Rules.Create(streamRules, false) // dryRun is set to false.
		if err != nil {
			return jerr.Get("error creating twitter API rules", err)
		} else if res.Errors != nil && len(res.Errors) > 0 {
			//https://developer.twitter.com/en/support/twitter-api/error-troubleshooting
			return jerr.Newf("Received an error from twitter: %v", res.Errors)
		}
	}
	return nil
}

func (s *Stream) ResetRules() error {
	/*s.SetFreshApi()
	defer s.CloseApi()*/
	res, err := s.Api.Rules.Get()
	if err != nil {
		return jerr.Get("error getting twitter API rules", err)
	}
	for _, rule := range res.Data {
		ID, _ := strconv.Atoi(rule.Id)
		res, err := s.Api.Rules.Delete(rules.NewDeleteRulesRequest(ID), false)
		if err != nil {
			return jerr.Get("error deleting twitter API rules", err)
		}
		if res.Errors != nil && len(res.Errors) > 0 {
			//https://developer.twitter.com/en/support/twitter-api/error-troubleshooting
			return jerr.Newf("Received an error from twitter: %v", res.Errors)
		}
	}
	return nil
}

func (s *Stream) InitiateStream(streamConfigs []config.Stream) error {
	s.SetFreshApi()
	if err := s.ResetRules(); err != nil {
		return jerr.Get("error twitter stream reset rules", err)
	}
	if err := s.FilterAccount(streamConfigs); err != nil {
		return jerr.Get("error twitter stream filter account", err)
	}
	s.Api.Stream.SetUnmarshalHook(func(bytes []byte) (interface{}, error) {
		fmt.Println(string(bytes))
		data := obj.TweetStreamData{}
		if err := json.Unmarshal(bytes, &data); err != nil {
			fmt.Printf("failed to unmarshal bytes: %v", err)
		}
		return data, nil
	})
	streamExpansions := twitterstream.NewStreamQueryParamsBuilder().
		AddExpansion("author_id").
		AddTweetField("created_at").
		AddTweetField("referenced_tweets").
		AddExpansion("attachments.media_keys").
		AddMediaField("url").
		Build()
	if err := s.Api.Stream.StartStream(streamExpansions); err != nil {
		return jerr.Get("error starting twitter stream", err)
	}
	tweetObject := twitter.Tweet{}
	for tweet := range s.Api.Stream.GetMessages() {
		if tweet.Err != nil {
			if jerr.HasErrorPart(tweet.Err, "response body closed") {
				break
			}
			return jerr.Get("got error from twitter", tweet.Err)
		}
		result := tweet.Data.(obj.TweetStreamData)
		tweetID, _ := strconv.ParseInt(result.Data.ID, 10, 64)
		userID, _ := strconv.ParseInt(result.Includes.Users[0].ID, 10, 64)
		var InReplyToStatusID int64
		if len(result.Data.ReferencedTweets) > 0 && result.Data.ReferencedTweets[0].Type == "replied_to" {
			InReplyToStatusID, _ = strconv.ParseInt(result.Data.ReferencedTweets[0].ID, 10, 64)
		} else {
			InReplyToStatusID = 0
		}
		var tweetText = result.Data.Text
		//pretty print result object
		b, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(b))

		if len(result.Data.Attachments.MediaKeys) > 0 {
			//use regex library to find the string https://t.co in the tweet text
			match, _ := regexp.MatchString("https://t.co/[a-zA-Z0-9]*$", tweetText)
			if match {
				//remove the https://t.co from the tweet text
				tweetText = regexp.MustCompile("https://t.co/[a-zA-Z0-9]*$").ReplaceAllString(tweetText, "")
			}
			for _, media := range result.Includes.Media {
				//append the url into to tweet text
				tweetText += "\n" + media.URL
			}
		}
		tweetObject = twitter.Tweet{
			ID:        tweetID,
			CreatedAt: result.Data.CreatedAt,
			Text:      tweetText,
			User: &twitter.User{
				ID:         userID,
				Name:       result.Includes.Users[0].Name,
				ScreenName: result.Includes.Users[0].Username,
			},
			InReplyToStatusID: InReplyToStatusID,
		}
		println(tweetText)
		println("\n\n\n")
		//fmt.Println(tweetObject.Text)
		tweetTx := obj.TweetTx{
			Tweet:  &tweetObject,
			TxHash: nil,
		}
		//call streamtweet
		//based on the stream config, get the right address to send the tweet to
		for _, conf := range streamConfigs {
			if conf.Name == tweetObject.User.ScreenName {
				println("sending tweet to key: ", conf.Key)
				twitterAccountWallet := obj.GetAccountKeyFromArgs([]string{conf.Key, conf.Name})
				var link = true
				var date = false
				flags,err := s.Db.Get([]byte("flags-" + conf.Sender +"-"+conf.Name),nil)
				if err != nil {
					if err == leveldb.ErrNotFound {
						continue
					} else {
						return jerr.Get("error getting flags from db", err)
					}
				} else{
					type Flags struct {
						Link bool `json:"link"`
						Date bool `json:"date"`
					}
					var flagsStruct Flags
					if err := json.Unmarshal(flags, &flagsStruct); err != nil {
						return jerr.Get("error unmarshalling flags", err)
					}
					link = flagsStruct.Link
					date = flagsStruct.Date
				}
				if err := database.SaveTweet(twitterAccountWallet, tweetTx, s.Db, link, date); err != nil {
					return jerr.Get("error streaming tweet in stream", err)
				}
			}
		}
	}
	fmt.Println("Stopped Stream")
	return nil
}
