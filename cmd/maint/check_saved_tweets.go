package maint

import (
	"encoding/json"
	"github.com/jchavannes/jgo/jerr"
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
			jerr.Get("fatal error getting all saved address tweets", err).Fatal()
		}
		log.Printf("count all tweet txs: %d\n", len(allTweetTxs))
		for _, dbTweetTx := range allTweetTxs {
			tweetTx := obj.TweetTx{}
			err := json.Unmarshal(dbTweetTx.Tx, &tweetTx)
			if err != nil {
				jerr.Get("fatal error unmarshalling tweet tx", err).Fatal()
			}
			if verbose {
				log.Printf("screen name: %s, tweetId: %d\n", dbTweetTx.ScreenName, tweetTx.Tweet.ID)
			}
		}
		savedTweets, err := db.GetAllSavedAddressTweet()
		if err != nil {
			jerr.Get("fatal error getting all saved address tweets", err).Fatal()
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
