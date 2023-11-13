package twitter

import (
	"github.com/dghubble/go-twitter/twitter"
	"github.com/memocash/tweet/config"
	"github.com/memocash/tweet/tweets"
	twitterscraper "github.com/n0madic/twitter-scraper"
	"github.com/spf13/cobra"
	"log"
	"strconv"
)

var newestCmd = &cobra.Command{
	Use:   "newest",
	Short: "newest <username>",
	Run: func(c *cobra.Command, args []string) {
		if len(args) < 1 {
			log.Fatal("must specify user name")
		}
		scraper := twitterscraper.New()
		scraper.SetSearchMode(twitterscraper.SearchLatest)
		if err := scraper.Login(config.GetTwitterCreds().GetStrings()...); err != nil {
			log.Fatalf("fatal error logging in to twitter; %v", err)
		}
		profile, err := scraper.GetProfile(args[0])
		if err != nil {
			log.Fatalf("fatal error getting profile; %v", err)
		}
		userId, err := strconv.ParseInt(profile.UserID, 10, 64)
		if err != nil {
			log.Fatalf("fatal error converting user id to int; %v", err)
		}
		log.Printf("User: %s, id: %d\n", profile.Username, userId)
		var userTimelineParams = &twitter.UserTimelineParams{
			Count:      5,
			UserID:     userId,
			ScreenName: profile.Username,
		}
		tweets, err := tweets.GetTwitterTweets(userTimelineParams, scraper)
		if err != nil {
			log.Fatalf("fatal error getting tweet; %v", err)
		}
		for _, tweet := range tweets {
			log.Printf("tweet: %s (%s)\n", tweet.Text, tweet.CreatedAt)
		}
	},
}
