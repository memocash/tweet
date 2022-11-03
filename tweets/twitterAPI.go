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

func GetAllTweets(screenName string, client *twitter.Client) []twitter.Tweet{
	var tweetList []twitter.Tweet
	for{
		tweets := GetTweets(screenName,client)
		tweetList = append(tweetList, tweets...)
		if len(tweets) == 1{
			tweetList = tweetList[:len(tweetList)-1]
			break
		}
	}
	return tweetList
}
func GetTweets(screenName string,client *twitter.Client) []twitter.Tweet{
	//Struct and function call to get ID of most recent tweet, or 0 if maxID.json doesn't exist
	type tweetID struct {
		ID int64
	}
	maxID := tweetID{
		ID: 0,
	}
	content, err := ioutil.ReadFile("./maxID.json")
	if err == nil{
		err = json.Unmarshal(content, &maxID)
	}
	// Query to Twitter API for all tweets after maxID.id
	var userTimelineParams *twitter.UserTimelineParams
	if maxID.ID != 0{
		userTimelineParams = &twitter.UserTimelineParams{ScreenName: screenName, MaxID: maxID.ID, Count: 100}
	}
	//just get the 100 most recent if maxID.json doesn't exist
	if maxID.ID == 0{
		userTimelineParams = &twitter.UserTimelineParams{ScreenName: screenName, Count: 100}

	}
	tweets, _, _ := client.Timelines.UserTimeline(userTimelineParams)

	for _, tweet := range tweets {
		// send tweet.Text through a graphQL query
		// save the highest tweet.ID to a config file
		println(tweet.Text)
		println(tweet.CreatedAt)
		if tweet.ID < maxID.ID || maxID.ID == 0{
			maxID.ID = tweet.ID
		}
	}
	//Save ID of latest tweet to a local file
	file,_ := json.MarshalIndent(maxID, "", " ")
	_ = ioutil.WriteFile("maxID.json", file, 0644)
	return tweets
}
func GetProfile(screenName string)(string,string,string){
	client := Connect()
	// Query to Twitter API for profile info
	// user show
	userShowParams := &twitter.UserShowParams{ScreenName: screenName}
	user, _, _ := client.Users.Show(userShowParams)
	desc := user.Description
	name := user.Name
	profilePic := user.ProfileImageURL
	fmt.Printf("USERS SHOW:\n%+v\n%+v\n%+v\n",name, desc, profilePic)
	return name, desc, profilePic
}

func Connect() *twitter.Client{
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
	return client
}

