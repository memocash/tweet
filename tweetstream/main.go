package tweetstream

import (
	"encoding/json"
	"fmt"
	"github.com/dghubble/go-twitter/twitter"
	twitterstream "github.com/fallenstedt/twitter-stream"
	"github.com/fallenstedt/twitter-stream/rules"
	"github.com/fallenstedt/twitter-stream/token_generator"
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/config"
	"github.com/memocash/tweet/database"
	"github.com/memocash/tweet/tweets"
	"github.com/syndtr/goleveldb/leveldb"
	"regexp"
	"strconv"
)

func GetStreamingToken() (*token_generator.RequestBearerTokenResponse, error) {
	conf := config.GetTwitterAPIConfig()
	if !conf.IsSet() {
		return nil, jerr.New("Application Access Token required")
	}
	return twitterstream.NewTokenGenerator().SetApiKeyAndSecret(conf.ConsumerKey, conf.ConsumerSecret).RequestBearerToken()
}

func FilterAccount(api *twitterstream.TwitterApi, streamConfigs []config.Stream) {
	var res *rules.TwitterRuleResponse
	var err error
	//remove duplicate names from stream configs
	uniqueNames := make(map[string]bool)
	for _, streamConfig := range streamConfigs {
		uniqueNames[streamConfig.Name] = true
	}
	for name := range uniqueNames {
		rules := twitterstream.NewRuleBuilder().AddRule("from:"+name, "get tweets from this account").Build()
		res, err = api.Rules.Create(rules, false) // dryRun is set to false.
		if err != nil {
			panic(err)
		}
	}
	if res.Errors != nil && len(res.Errors) > 0 {
		//https://developer.twitter.com/en/support/twitter-api/error-troubleshooting
		panic(fmt.Sprintf("Received an error from twitter: %v", res.Errors))
	}
}

func ResetRules(api *twitterstream.TwitterApi) {
	res, err := api.Rules.Get()
	if err != nil {
		panic(err)
	}
	for _, rule := range res.Data {
		ID, _ := strconv.Atoi(rule.Id)
		res, err := api.Rules.Delete(rules.NewDeleteRulesRequest(ID), false)
		if err != nil {
			panic(err)
		}
		if res.Errors != nil && len(res.Errors) > 0 {
			//https://developer.twitter.com/en/support/twitter-api/error-troubleshooting
			panic(fmt.Sprintf("Received an error from twitter: %v", res.Errors))
		}
	}
}

func InitiateStream(api *twitterstream.TwitterApi, streamConfigs []config.Stream, db *leveldb.DB) {
	defer InitiateStream(api, streamConfigs, db)
	tweetObject := twitter.Tweet{}
	for tweet := range api.Stream.GetMessages() {

		// Handle disconnections from twitter
		// https://developer.twitter.com/en/docs/twitter-api/tweets/volume-streams/integrate/handling-disconnections
		if tweet.Err != nil {
			fmt.Printf("got error from twitter: %v", tweet.Err)
			// Notice we "StopStream" and then "continue" the loop instead of breaking.
			// StopStream will close the long running GET request to Twitter's v2 Streaming endpoint by
			// closing the `GetMessages` channel. Once it's closed, it's safe to perform a new network request
			// with `StartStream`
			api.Stream.StopStream()
			continue
		}
		result := tweet.Data.(tweets.TweetStreamData)
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
		TweetTx := tweets.TweetTx{
			Tweet:  &tweetObject,
			TxHash: nil,
		}
		//call streamtweet
		//based on the stream config, get the right address to send the tweet to
		for _, config := range streamConfigs {
			if config.Name == tweetObject.User.ScreenName {
				println("sending tweet to key: ", config.Key)
				twitterAccountWallet := tweets.GetAccountKeyFromArgs([]string{config.Key, config.Name})
				database.StreamTweet(twitterAccountWallet, TweetTx, db, true, false)
			}
		}
	}
	fmt.Println("Stopped Stream")
}

func FetchTweets(token string) *twitterstream.TwitterApi {
	api := twitterstream.NewTwitterStream(token)
	api.Stream.SetUnmarshalHook(func(bytes []byte) (interface{}, error) {
		fmt.Println(string(bytes))
		data := tweets.TweetStreamData{}
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
	err := api.Stream.StartStream(streamExpansions)
	if err != nil {
		panic(err)
	}
	return api
}
