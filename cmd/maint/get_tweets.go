package maint

import (
	"context"
	"fmt"
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/config"
	twitterscraper "github.com/n0madic/twitter-scraper"
	"github.com/spf13/cobra"
)

const (
	SCREEN_NAME = "MemoCashAbdel"
	MAX_TWEETS  = 10
)

var getTweetsCmd = &cobra.Command{
	Use:   "get-tweets",
	Short: "Testing getting tweets from the scraper",
	Run: func(c *cobra.Command, args []string) {
		scraper := twitterscraper.New()
		scraper.SetSearchMode(twitterscraper.SearchLatest)
		err := scraper.Login(config.GetTwitterAPIConfig().UserName, config.GetTwitterAPIConfig().Password)
		if err != nil {
			jerr.Get("error logging in", err).Fatal()
		}
		query := fmt.Sprintf("from:%s", SCREEN_NAME)

		for scrapedTweet := range scraper.SearchTweets(context.Background(), query, MAX_TWEETS) {
			if scrapedTweet.Error != nil {
				jerr.Get("error getting tweets", scrapedTweet.Error).Fatal()
			}
			println("Got tweet", scrapedTweet.ID, scrapedTweet.Text)
		}
	},
}
