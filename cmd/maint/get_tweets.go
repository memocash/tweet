package maint

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/config"
	"github.com/memocash/tweet/tweets"
	twitterscraper "github.com/n0madic/twitter-scraper"
	"github.com/spf13/cobra"
	"net/http"
	"os"
)

const (
	MAX_TWEETS = 10
)

var getTweetsCmd = &cobra.Command{
	Use:   "get-tweets",
	Short: "Testing getting tweets from the scraper",
	Run: func(c *cobra.Command, args []string) {
		if len(args) != 1 {
			jerr.Get("error: must specify screen name", nil).Fatal()
		}
		scraper := twitterscraper.New()
		//look for and unmarshal the cookie jar
		cookies, err := os.ReadFile(tweets.COOKJAR_FILE)
		if err != nil && !os.IsNotExist(err) {
			jerr.Get("error reading cookies", err).Fatal()
		}
		if !os.IsNotExist(err) {
			var cookieList []*http.Cookie
			err := json.Unmarshal(cookies, &cookieList)
			if err != nil {
				jerr.Get("error unmarshalling cookies", err).Fatal()
			}
			scraper.SetCookies(cookieList)
		}
		scraper.SetSearchMode(twitterscraper.SearchLatest)
		if err = scraper.Login(config.GetTwitterCreds().GetStrings()...); err != nil {
			err2 := tweets.SaveCookies(scraper.GetCookies())
			if err2 != nil {
				jerr.Get("error saving cookies", err2).Fatal()
			}
			jerr.Get("error logging in", err).Fatal()
		}
		query := fmt.Sprintf("from:%s", args[0])
		for scrapedTweet := range scraper.SearchTweets(context.Background(), query, MAX_TWEETS) {
			if scrapedTweet.Error != nil {
				err2 := tweets.SaveCookies(scraper.GetCookies())
				if err2 != nil {
					jerr.Get("error saving cookies", err2).Fatal()
				}
				jerr.Get("error getting tweets", scrapedTweet.Error).Fatal()
			}
		}
		//save the cookie jar
		err = tweets.SaveCookies(scraper.GetCookies())
		if err != nil {
			jerr.Get("error saving cookies", err).Fatal()
		}
	},
}
