package maint

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/config"
	"github.com/memocash/tweet/db"
	"github.com/memocash/tweet/tweets"
	twitterscraper "github.com/n0madic/twitter-scraper"
	"github.com/spf13/cobra"
	"github.com/syndtr/goleveldb/leveldb"
	"log"
	"net/http"
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
		dbCookies, err := db.GetCookies()
		if err != nil && !errors.Is(err, leveldb.ErrNotFound) {
			log.Fatalf("error getting cookies from db; %v", err)
		}
		if dbCookies != nil {
			var cookieList []*http.Cookie
			err := json.Unmarshal(dbCookies.CookieData, &cookieList)
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
