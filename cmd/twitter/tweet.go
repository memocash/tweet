package twitter

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"github.com/memocash/tweet/config"
	twitterscraper "github.com/n0madic/twitter-scraper"
	"github.com/spf13/cobra"
	"log"
)

var tweetCmd = &cobra.Command{
	Use:   "tweet",
	Short: "tweet <url>",
	Run: func(c *cobra.Command, args []string) {
		if len(args) < 1 {
			log.Fatal("must specify tweet url")
		}
		scraper := twitterscraper.New()
		scraper.SetSearchMode(twitterscraper.SearchLatest)
		if err := scraper.Login(config.GetTwitterCreds().GetStrings()...); err != nil {
			log.Fatalf("fatal error logging in to twitter; %v", err)
		}
		tweet, err := scraper.GetTweet(args[0])
		if err != nil {
			log.Fatalf("fatal error getting tweet; %v", err)
		}
		var b = new(bytes.Buffer)
		e := gob.NewEncoder(b)
		if err := e.Encode(tweet); err != nil {
			log.Fatalf("fatal error encoding tweet; %v", err)
		}
		log.Printf("tweet: %s\n", tweet.Text)
		log.Printf("gob: %s\n", base64.StdEncoding.EncodeToString(b.Bytes()))
	},
}

var tweetByGobCmd = &cobra.Command{
	Use:   "tweet-by-gob",
	Short: "tweet-by-gob <gob>",
	Run: func(c *cobra.Command, args []string) {
		if len(args) < 1 {
			log.Fatal("must specify tweet gob")
		}
		by, err := base64.StdEncoding.DecodeString(args[0])
		if err != nil {
			log.Fatalf("fatal error decoding base64; %v", err)
		}
		b := bytes.Buffer{}
		b.Write(by)
		d := gob.NewDecoder(&b)
		var tweet twitterscraper.Tweet
		if err = d.Decode(&tweet); err != nil {
			log.Fatalf("fatal error decoding gob; %v", err)
		}
		log.Printf("tweet: %#v\n", tweet)
		for _, url := range tweet.URLs {
			log.Printf("url: %s\n", url)
		}
		for _, photo := range tweet.Photos {
			log.Printf("url: %s\n", photo.URL)
		}
	},
}
