package tweets

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/coreos/pkg/flagutil"
	"github.com/dghubble/go-twitter/twitter"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"io/ioutil"
	"log"
)

func Load() {
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

	// oauth2 configures a client that uses app credentials to keep a fresh token
	config := &clientcredentials.Config{
		ClientID:     flags.consumerKey,
		ClientSecret: flags.consumerSecret,
		TokenURL:     "https://api.twitter.com/oauth2/token",
	}
	// http.Client will automatically authorize Requests
	httpClient := config.Client(oauth2.NoContext)

	// Twitter client
	client := twitter.NewClient(httpClient)

	//Struct and function call to get ID of most recent tweet, or 0 if sinceID.json doesn't exist
	type tweetID struct {
		ID int64
	}
	sinceID := tweetID{
		ID: 0,
	}
	content, err := ioutil.ReadFile("./sinceID.json")
	if err == nil{
		err = json.Unmarshal(content, &sinceID)
	}

	// Query to Twitter API for all tweets after sinceID.id
	userTimelineParams := &twitter.UserTimelineParams{ScreenName: "MemoCashAbdel", SinceID: sinceID.ID}
	tweets, _, _ := client.Timelines.UserTimeline(userTimelineParams)

	for _, tweet := range tweets {
		// send tweet.Text through a graphQL query
		// save the highest tweet.ID to a config file
		if tweet.ID > sinceID.ID{
			sinceID.ID = tweet.ID
		}
	}
	//Save ID of latest tweet to a local file
	file,_ := json.MarshalIndent(sinceID, "", " ")
	_ = ioutil.WriteFile("sinceID.json", file, 0644)
	fmt.Printf("USER TIMELINE:\n%+v\n", tweets)
}
