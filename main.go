package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/coreos/pkg/flagutil"
	"github.com/dghubble/go-twitter/twitter"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

func main() {
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

	// user show
	userShowParams := &twitter.UserShowParams{ScreenName: "MemoCashAbdel"}
	user, _, _ := client.Users.Show(userShowParams)
	fmt.Printf("USERS SHOW:\n%+v\n", user)

	// user timeline
	userTimelineParams := &twitter.UserTimelineParams{ScreenName: "MemoCashAbdel", Count: 2}
	tweets, _, _ := client.Timelines.UserTimeline(userTimelineParams)
	fmt.Printf("USER TIMELINE:\n%+v\n", tweets)

}