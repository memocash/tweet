package tweets

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/fallenstedt/twitter-stream"
	"github.com/fallenstedt/twitter-stream/rules"
	"github.com/fallenstedt/twitter-stream/token_generator"
	"github.com/jchavannes/jgo/jerr"
	"github.com/jchavannes/jgo/jlog"
	"github.com/memocash/tweet/config"
	"github.com/memocash/tweet/db"
	"github.com/memocash/tweet/tweets/obj"
	"github.com/memocash/tweet/tweets/save"
	"github.com/syndtr/goleveldb/leveldb"
	"regexp"
	"strconv"
	"sync"
)

type Stream struct {
	Api   *twitterstream.TwitterApi
	Db    *leveldb.DB
	Token *token_generator.RequestBearerTokenResponse
	Mutex sync.Mutex
}

func NewStream() (*Stream, error) {
	conf := config.GetTwitterAPIConfig()
	if !conf.IsSet() {
		return nil, jerr.New("Application Access Token required")
	}
	token, err := twitterstream.NewTokenGenerator().SetApiKeyAndSecret(conf.ConsumerKey, conf.ConsumerSecret).RequestBearerToken()
	if err != nil {
		return nil, jerr.Get("error getting twitter API token", err)
	}
	return &Stream{
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
	uniqueIds := make(map[string]bool)
	for _, streamConfig := range streamConfigs {
		uniqueIds[strconv.FormatInt(streamConfig.UserID, 10)] = true
	}
	for userId := range uniqueIds {
		streamRules := twitterstream.NewRuleBuilder().AddRule("from:"+userId, "get tweets from this account").Build()
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

func (s *Stream) ListenForNewTweets(streamConfigs []config.Stream) error {
	if s == nil {
		return jerr.New("error stream is nil for initiate stream")
	}
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	s.SetFreshApi()
	if err := s.ResetRules(); err != nil {
		return jerr.Get("error twitter stream reset rules", err)
	}
	for _, streamConfig := range streamConfigs {
		jlog.Logf("Adding stream config: %s %s\n", streamConfig.Sender, streamConfig.Name)
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
	jlog.Log("Starting Twitter API stream")
	if err := s.Api.Stream.StartStream(streamExpansions); err != nil {
		return jerr.Get("error starting twitter stream", err)
	}
	tweetObject := twitter.Tweet{}
	for tweet := range s.Api.Stream.GetMessages() {
		if tweet.Err != nil {
			if jerr.HasErrorPart(tweet.Err, "response body closed") {
				break
			}
			return jerr.Get("error twitter api stream get messages", tweet.Err)
		}
		result := tweet.Data.(obj.TweetStreamData)
		if len(result.Errors) > 0 {
			return jerr.Get("error twitter api stream get messages object", obj.CombineTweetStreamErrors(result.Errors))
		}
		tweetID, err := strconv.ParseInt(result.Data.ID, 10, 64)
		if err != nil {
			return jerr.Get("error parsing tweet id for api stream", err)
		}
		userID, err := strconv.ParseInt(result.Includes.Users[0].ID, 10, 64)
		if err != nil {
			return jerr.Get("error parsing user id api stream", err)
		}
		var InReplyToStatusID int64
		if len(result.Data.ReferencedTweets) > 0 && result.Data.ReferencedTweets[0].Type == "replied_to" {
			InReplyToStatusID, _ = strconv.ParseInt(result.Data.ReferencedTweets[0].ID, 10, 64)
		} else {
			InReplyToStatusID = 0
		}
		var tweetText = result.Data.Text
		b, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return jerr.Get("error pretty printing result object", err)
		}
		jlog.Logf("stream result object: %s\n", b)
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
		jlog.Logf("tweetText: %s\n", tweetText)
		tweetTx := obj.TweetTx{
			Tweet:  &tweetObject,
			TxHash: nil,
		}
		//call streamtweet
		//based on the stream config, get the right address to send the tweet to
		for _, conf := range streamConfigs {
			if conf.UserID == tweetObject.User.ID {
				jlog.Logf("sending tweet to key: %s\n", conf.Key)
				twitterAccountWallet := obj.GetAccountKeyFromArgs([]string{conf.Key, strconv.FormatInt(conf.UserID, 10)})
				flag, err := db.GetFlag(conf.Sender, strconv.FormatInt(conf.UserID, 10))
				if err != nil && !errors.Is(err, leveldb.ErrNotFound) {
					return jerr.Get("error getting flags from db", err)
				} else if errors.Is(err, leveldb.ErrNotFound) {
					continue
				}
				err = save.Tweet(conf.Wallet, twitterAccountWallet.GetAddress(), tweetTx, flag.Flags)
				if err != nil {
					return jerr.Get("error streaming tweet in stream", err)
				}
			}
		}
	}
	jlog.Log("Stopped Stream")
	return nil
}
