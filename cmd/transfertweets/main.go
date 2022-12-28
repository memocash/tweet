package transfertweets

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/database"
	"github.com/memocash/tweet/tweets"
	"github.com/memocash/tweet/tweets/obj"
	"github.com/spf13/cobra"
)

var link bool = false
var date bool = false

var transferCmd = &cobra.Command{
	Use:   "transfertweets",
	Short: "Transfer tweets to memo posts",
	Long: "The first time this command is run, it will populate tweetArchive.json with all of the tweets" +
		" from the twitter account and make the oldest 20 into memo posts. After it will transfer 20 each time" +
		"it is run. Deleting the tweetArchive.json file will cause it to restart from the beginning.",
	Args: cobra.ExactArgs(2),
	Run: func(c *cobra.Command, args []string) {
		accountKey := obj.GetAccountKeyFromArgs(args)
		client := tweets.Connect()
		db, err := database.GetDb()
		if err != nil {
			jerr.Get("error opening db", err).Fatal()
		}
		defer db.Close()
		if _,err := tweets.GetAllTweets(accountKey.Account, client, db); err != nil {
			jerr.Get("error getting all tweets", err).Fatal()
		}
		if _, err = tweets.Transfer(accountKey, db, link, date); err != nil {
			jerr.Get("fatal error transferring tweets", err).Fatal()
		}
	},
}

func GetCommand() *cobra.Command {
	//if link and date are supplied, the tweet will be linked and the date will be added to the memo post
	transferCmd.PersistentFlags().BoolVarP(&link, "link", "l", false, "link to tweet")
	transferCmd.PersistentFlags().BoolVarP(&date, "date", "d", false, "add date to post")
	return transferCmd
}
