package getnewtweets

import (
	"fmt"
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/config"
	"github.com/memocash/tweet/database"
	"github.com/memocash/tweet/tweets"
	"github.com/memocash/tweet/tweets/obj"
	"github.com/spf13/cobra"
	util2 "github.com/syndtr/goleveldb/leveldb/util"
)

var link bool = false
var date bool = false

var transferCmd = &cobra.Command{
	Use:   "getnewtweets",
	Short: "Listens for new tweets on an account",
	Long:  "Prints out each new tweet as it comes in. ",
	Run: func(c *cobra.Command, args []string) {
		db, err := database.GetDb()
		if err != nil {
			jerr.Get("fatal error getting db", err).Fatal()
		}
		streamConfigs := config.GetConfig().Streams
		//before starting the stream, ge the latest tweets newer than the last tweet in the db
		for _, streamConfig := range streamConfigs {
			accountKey := obj.GetAccountKeyFromArgs([]string{streamConfig.Key, streamConfig.Name})
			//check if there are any transferred tweets with the prefix containing this address and this screenName
			savedPrefix := fmt.Sprintf("saved-%s-%s", accountKey.Address, accountKey.Account)
			iter := db.NewIterator(util2.BytesPrefix([]byte(savedPrefix)), nil)
			tweetsFound := iter.First()
			iter.Release()
			if tweetsFound {
				err := tweets.GetSkippedTweets(accountKey,tweets.Connect(), db, link, date, 100)
				if err != nil {
					jerr.Get("error getting skipped tweets", err).Print()
				}
			}
		}
		stream, err := tweets.NewStream(db)
		if err != nil {
			jerr.Get("error getting new tweet stream", err).Fatal()
		}
		if err := stream.InitiateStream(streamConfigs); err != nil {
			jerr.Get("error twitter initiate stream get new tweets", err).Fatal()
		}
	},
}

func GetCommand() *cobra.Command {
	//if link and date are supplied, the tweet will be linked and the date will be added to the memo post
	transferCmd.PersistentFlags().BoolVarP(&link, "link", "l", false, "link to tweet")
	transferCmd.PersistentFlags().BoolVarP(&date, "date", "d", false, "add date to post")
	return transferCmd
}
