package maint

import (
	"context"
	"encoding/json"
	"fmt"
	twitterscraper "github.com/AbdelSallam/twitter-scraper"
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/config"
	"github.com/spf13/cobra"
	"net/http"
	"os"
)

const (
	MAX_TWEETS   = 10
	COOKJAR_FILE = "cookies.json"
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
		cookies, err := os.ReadFile(COOKJAR_FILE)
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
		err = scraper.Login(config.GetTwitterAPIConfig().UserName, config.GetTwitterAPIConfig().Password)
		if err != nil {
			err2 := SaveCookies(scraper.GetCookies())
			if err2 != nil {
				jerr.Get("error saving cookies", err2).Fatal()
			}
			jerr.Get("error logging in", err).Fatal()
		}
		query := fmt.Sprintf("from:%s", args[0])
		println("Searching for", query)
		for scrapedTweet := range scraper.SearchTweets(context.Background(), query, MAX_TWEETS) {
			if scrapedTweet.Error != nil {
				err2 := SaveCookies(scraper.GetCookies())
				if err2 != nil {
					jerr.Get("error saving cookies", err2).Fatal()
				}
				jerr.Get("error getting tweets", scrapedTweet.Error).Fatal()
			}
			println("Got tweet", scrapedTweet.ID, scrapedTweet.Text)
		}
		//save the cookie jar
		err = SaveCookies(scraper.GetCookies())
		if err != nil {
			jerr.Get("error saving cookies", err).Fatal()
		}
	},
}

func SaveCookies(cookies []*http.Cookie) error {
	marshaledCookies, err := json.Marshal(cookies)
	if err != nil {
		return jerr.Get("error marshalling cookies", err)
	}
	err = os.WriteFile(COOKJAR_FILE, marshaledCookies, 0644)
	if err != nil {
		return jerr.Get("error writing cookies", err)
	}
	return nil
}
