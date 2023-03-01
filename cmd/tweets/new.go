package tweets

import (
	"errors"
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/config"
	"github.com/memocash/tweet/db"
	"github.com/memocash/tweet/tweets"
	"github.com/memocash/tweet/tweets/obj"
	"github.com/memocash/tweet/wallet"
	"github.com/spf13/cobra"
	"github.com/syndtr/goleveldb/leveldb"
)

var getNewCmd = &cobra.Command{
	Use:   "get-new",
	Short: "Listens for new tweets on an account",
	Long:  "Prints out each new tweet as it comes in. ",
	Run: func(c *cobra.Command, args []string) {
		link, _ := c.Flags().GetBool(FlagLink)
		date, _ := c.Flags().GetBool(FlagDate)
		streamConfigs := config.GetConfig().Streams
		//before starting the stream, ge the latest tweets newer than the last tweet in the db
		for _, streamConfig := range streamConfigs {
			accountKey := obj.GetAccountKeyFromArgs([]string{streamConfig.Key, streamConfig.Name})
			//check if there are any transferred tweets with the prefix containing this address and this screenName
			savedAddressTweet, err := db.GetRecentSavedAddressTweet(accountKey.Address.GetEncoded(), accountKey.Account)
			if err != nil && !errors.Is(err, leveldb.ErrNotFound) {
				jerr.Get("error getting recent saved address tweet", err).Fatal()
			}
			if savedAddressTweet != nil {
				wlt := wallet.NewWallet(accountKey.Address, accountKey.Key)
				err := tweets.GetSkippedTweets(accountKey, &wlt, tweets.Connect(), db.Flags{Link: link, Date: date}, 100, true)
				if err != nil {
					jerr.Get("error getting skipped tweets for get new tweets", err).Fatal()
				}
			}
		}
		stream, err := tweets.NewStream()
		if err != nil {
			jerr.Get("error getting new tweet stream", err).Fatal()
		}
		if err := stream.ListenForNewTweets(streamConfigs); err != nil {
			jerr.Get("error twitter initiate stream get new tweets", err).Fatal()
		}
	},
}
