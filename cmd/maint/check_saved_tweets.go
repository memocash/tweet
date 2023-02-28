package maint

import (
	"encoding/json"
	"fmt"
	"github.com/memocash/tweet/db"
	"github.com/memocash/tweet/tweets/obj"
	"github.com/spf13/cobra"
	"log"
)

var checkSavedTweetsCmd = &cobra.Command{
	Use:   "check-saved-tweets",
	Short: "check-saved-tweets",
	Run: func(c *cobra.Command, args []string) {
		verbose, _ := c.Flags().GetBool(FlagVerbose)
		allTweetTxs, err := db.GetAllTweetTx()
		if err != nil {
			log.Fatalf("fatal error getting all saved address tweets; %v", err)
		}
		log.Printf("count all tweet txs: %d\n", len(allTweetTxs))
		for _, dbTweetTx := range allTweetTxs {
			tweetTx := obj.TweetTx{}
			err := json.Unmarshal(dbTweetTx.Tx, &tweetTx)
			if err != nil {
				log.Fatalf("fatal error unmarshalling tweet tx; %v", err)
			}
			if verbose {
				log.Printf("screen name: %s, tweetId: %d\n", dbTweetTx.ScreenName, tweetTx.Tweet.ID)
			}
		}
		savedTweets, err := db.GetAllSavedAddressTweet(nil)
		if err != nil {
			log.Fatalf("fatal error getting all saved address tweets; %v", err)
		}
		log.Printf("count all saved address tweets: %d\n", len(savedTweets))
		if verbose {
			for _, savedTweet := range savedTweets {
				log.Printf("address: %s, screen name: %s, tweetId: %d\n",
					savedTweet.Address, savedTweet.ScreenName, savedTweet.TweetId)
			}
		}
	},
}

var checkSavedTweetCmd = &cobra.Command{
	Use:   "check-saved-tweet",
	Short: "check-saved-tweet",
	Run: func(c *cobra.Command, args []string) {
		if len(args) < 2 {
			log.Fatalf("need address and screen name")
		}
		verbose, _ := c.Flags().GetBool(FlagVerbose)
		address := args[0]
		screenName := args[1]
		screenNameTweetTxs, err := db.GetTweetTxs(screenName, 0, 0)
		if err != nil {
			log.Fatalf("fatal error getting address screen name saved address tweets; %v", err)
		}
		log.Printf("count screen name tweet txs: %d\n", len(screenNameTweetTxs))
		for _, dbTweetTx := range screenNameTweetTxs {
			tweetTx := obj.TweetTx{}
			err := json.Unmarshal(dbTweetTx.Tx, &tweetTx)
			if err != nil {
				log.Fatalf("fatal error unmarshalling tweet tx; %v", err)
			}
			if verbose {
				log.Printf("screen name: %s, tweetId: %d\n", dbTweetTx.ScreenName, tweetTx.Tweet.ID)
			}
		}
		savedTweets, err := db.GetAllSavedAddressTweet([]byte(fmt.Sprintf("%s-%s", address, screenName)))
		if err != nil {
			log.Fatalf("fatal error getting address screen name saved address tweets; %v", err)
		}
		log.Printf("count address screen name saved address tweets: %d\n", len(savedTweets))
		if verbose {
			for _, savedTweet := range savedTweets {
				log.Printf("address: %s, screen name: %s, tweetId: %d\n",
					savedTweet.Address, savedTweet.ScreenName, savedTweet.TweetId)
			}
		}
	},
}

var fixSavedTweetCmd = &cobra.Command{
	Use:   "fix-saved-tweet",
	Short: "fix-saved-tweet",
	Run: func(c *cobra.Command, args []string) {
		if len(args) < 2 {
			log.Fatalf("need address and screen name")
		}
		noDryRun, _ := c.Flags().GetBool(FlagNoDryRun)
		address := args[0]
		screenName := args[1]
		savedTweets, err := db.GetAllSavedAddressTweet(nil)
		if err != nil {
			log.Fatalf("fatal error getting all saved address tweets; %v", err)
		}
		for _, savedTweet := range savedTweets {
			if len(savedTweet.Address) == len(address) || savedTweet.ScreenName != screenName {
				continue
			}
			log.Printf("found bad saved tweet address: %s, screen name: %s, tweetId: %d\n",
				savedTweet.Address, savedTweet.ScreenName, savedTweet.TweetId)
			if noDryRun {
				if err := db.Delete([]db.ObjectI{savedTweet}); err != nil {
					log.Fatalf("fatal error deleting bad saved tweet; %v", err)
				}
				savedTweet.Address = address
				if err := db.Save([]db.ObjectI{savedTweet}); err != nil {
					log.Fatalf("fatal error saving fixed saved tweet; %v", err)
				}
			} else {
				log.Printf("dry run, skipping fix\n")
			}
		}
	},
}
