package tweetstream

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/coreos/pkg/flagutil"
	"github.com/dghubble/go-twitter/twitter"
	twitterstream "github.com/fallenstedt/twitter-stream"
	"github.com/fallenstedt/twitter-stream/rules"
	"github.com/fallenstedt/twitter-stream/stream"
	"github.com/fallenstedt/twitter-stream/token_generator"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/cmd/util"
	util2 "github.com/memocash/tweet/database/util"
	"log"
	"strconv"
)
func GetStreamingToken() (*token_generator.RequestBearerTokenResponse, error){
	flags := struct {
		consumerKey    string
		consumerSecret string
	}{}
	flag.StringVar(&flags.consumerKey, "consumer-key", "", "Twitter Consumer Key")
	flag.StringVar(&flags.consumerSecret, "consumer-secret", "", "Twitter Consumer Secret")
	flag.Parse()
	flagutil.SetFlagsFromEnv(flag.CommandLine, "TWITTER")

	if flags.consumerKey == "" || flags.consumerSecret == "" {
		log.Fatal("Application Access Token required")
	}
	return twitterstream.NewTokenGenerator().SetApiKeyAndSecret(flags.consumerKey, flags.consumerSecret).RequestBearerToken()

}
func FilterAccount(tok *token_generator.RequestBearerTokenResponse, screenName string) {
	api := twitterstream.NewTwitterStream(tok.AccessToken)
	rules := twitterstream.NewRuleBuilder().AddRule("from:" + screenName, "only get tweets from one account").Build()

	res, err := api.Rules.Create(rules, false) // dryRun is set to false.

	if err != nil {
		panic(err)
	}

	if res.Errors != nil && len(res.Errors) > 0 {
		//https://developer.twitter.com/en/support/twitter-api/error-troubleshooting
		panic(fmt.Sprintf("Received an error from twitter: %v", res.Errors))
	}
}

func ResetRules(tok *token_generator.RequestBearerTokenResponse){
	api := twitterstream.NewTwitterStream(tok.AccessToken)
	res,err := api.Rules.Get()
	if err != nil {
		panic(err)
	}
	for _, rule := range res.Data {
		ID,_ := strconv.Atoi(rule.Id)
		res, err := api.Rules.Delete(rules.NewDeleteRulesRequest(ID),false)
		if err != nil {
			panic(err)
		}
		if res.Errors != nil && len(res.Errors) > 0 {
			//https://developer.twitter.com/en/support/twitter-api/error-troubleshooting
			panic(fmt.Sprintf("Received an error from twitter: %v", res.Errors))
		}
	}
}

func InitiateStream(tok *token_generator.RequestBearerTokenResponse, address wallet.Address, key wallet.PrivateKey){
	api := fetchTweets(tok.AccessToken)

	defer InitiateStream(tok,address,key)
	tweetObject := twitter.Tweet{}
	for tweet := range api.GetMessages() {

		// Handle disconnections from twitter
		// https://developer.twitter.com/en/docs/twitter-api/tweets/volume-streams/integrate/handling-disconnections
		if tweet.Err != nil {
			fmt.Printf("got error from twitter: %v", tweet.Err)
			// Notice we "StopStream" and then "continue" the loop instead of breaking.
			// StopStream will close the long running GET request to Twitter's v2 Streaming endpoint by
			// closing the `GetMessages` channel. Once it's closed, it's safe to perform a new network request
			// with `StartStream`
			api.StopStream()
			continue
		}
		result := tweet.Data.(util.TweetStreamData)

		// Here I am printing out the text.
		// You can send this off to a queue for processing.
		// Or do your processing here in the loop
		tweetID,_ := strconv.ParseInt(result.Data.ID,10,64)
		userID,_ := strconv.ParseInt(result.Includes.Users[0].ID,10,64)
		var InReplyToStatusID int64
		if len(result.Data.ReferencedTweets) > 0 && result.Data.ReferencedTweets[0].Type == "replied_to"{
			InReplyToStatusID,_ = strconv.ParseInt(result.Data.ReferencedTweets[0].ID,10,64)
		} else{
			InReplyToStatusID = 0
		}
		//build a twitter.Tweets object from the stream data
		tweetObject = twitter.Tweet{
			ID: tweetID,
			CreatedAt: result.Data.CreatedAt.Format("200601021504"),
			Text: result.Data.Text,
			User: &twitter.User{
				ID: userID,
				Name: result.Includes.Users[0].Name,
				ScreenName: result.Includes.Users[0].Username,
			},
			InReplyToStatusID: InReplyToStatusID,
			}
		//fmt.Println(tweetObject.Text)
		TweetTx := util.TweetTx{
			Tweet: &tweetObject,
			TxHash: nil,
		}
		archive := util.Archive{
			TweetList: []util.TweetTx{TweetTx},
			Archived: 0,
		}
		//call transfertweets
		util2.TransferTweets(address, key,archive,true, true)
	}

	fmt.Println("Stopped Stream")

}
func fetchTweets(token string) stream.IStream {
	api := twitterstream.NewTwitterStream(token).Stream
	api.SetUnmarshalHook(func(bytes []byte) (interface{}, error) {
		fmt.Println(string(bytes))
		data := util.TweetStreamData{}
		if err := json.Unmarshal(bytes, &data); err != nil {
			fmt.Printf("failed to unmarshal bytes: %v", err)
		}
		return data, nil
	})
	streamExpansions := twitterstream.NewStreamQueryParamsBuilder().
		AddExpansion("author_id").
		AddTweetField("created_at").
		AddTweetField("referenced_tweets").
		Build()
	err := api.StartStream(streamExpansions)
	if err != nil {
		panic(err)
	}
	return api
}
