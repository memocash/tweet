package getnewtweets

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/config"
	"github.com/memocash/tweet/database"
	"github.com/memocash/tweet/tweets"
	"github.com/memocash/tweet/tweets/obj"
	"github.com/spf13/cobra"
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
			//create an AccountKey object from the streamconfig
			accountKey := obj.GetAccountKeyFromArgs([]string{streamConfig.Key, streamConfig.Name})
			txList, err := tweets.GetNewTweets(streamConfig.Name,tweets.Connect(),db)
			if err != nil {
				jerr.Get("error getting tweets since the bot was last run", err).Fatal()
			}
			numLeft := len(txList)
			for numLeft > 0 {
				if _, err = tweets.Transfer(accountKey, db, link, date); err != nil {
					jerr.Get("fatal error transferring tweets", err).Fatal()
				}
				numLeft -= 20
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
