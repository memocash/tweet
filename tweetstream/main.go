package tweetstream

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/coreos/pkg/flagutil"
	twitterstream "github.com/fallenstedt/twitter-stream"
	"github.com/fallenstedt/twitter-stream/rules"
	"github.com/fallenstedt/twitter-stream/stream"
	"github.com/fallenstedt/twitter-stream/token_generator"
	"github.com/memocash/tweet/cmd/util"
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

func InitiateStream(tok *token_generator.RequestBearerTokenResponse){
	api := fetchTweets(tok.AccessToken)

	defer InitiateStream(tok)
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
		result := tweet.Data.(util.StreamDataExample)

		// Here I am printing out the text.
		// You can send this off to a queue for processing.
		// Or do your processing here in the loop
		fmt.Println(result.Data.Text)
	}

	fmt.Println("Stopped Stream")

}
func fetchTweets(token string) stream.IStream {
	api := twitterstream.NewTwitterStream(token).Stream
	api.SetUnmarshalHook(func(bytes []byte) (interface{}, error) {
		data := util.StreamDataExample{}
		if err := json.Unmarshal(bytes, &data); err != nil {
			fmt.Printf("failed to unmarshal bytes: %v", err)
		}
		return data, nil
	})
	streamExpansions := twitterstream.NewStreamQueryParamsBuilder().
		AddExpansion("author_id").
		AddTweetField("created_at").
		Build()
	err := api.StartStream(streamExpansions)
	if err != nil {
		panic(err)
	}
	return api
}
