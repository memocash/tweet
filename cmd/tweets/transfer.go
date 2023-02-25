package tweets

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/db"
	"github.com/memocash/tweet/tweets"
	"github.com/memocash/tweet/tweets/obj"
	"github.com/memocash/tweet/wallet"
	"github.com/spf13/cobra"
)

var transferCmd = &cobra.Command{
	Use:   "transfer",
	Short: "Transfer tweets to memo posts",
	Long: "The first time this command is run, it will populate tweetArchive.json with all of the tweets" +
		" from the twitter account and make the oldest 20 into memo posts. After it will transfer 20 each time" +
		"it is run. Deleting the tweetArchive.json file will cause it to restart from the beginning.",
	Args: cobra.ExactArgs(2),
	Run: func(c *cobra.Command, args []string) {
		link, _ := c.Flags().GetBool(FlagLink)
		date, _ := c.Flags().GetBool(FlagDate)
		accountKey := obj.GetAccountKeyFromArgs(args)
		client := tweets.Connect()
		db, err := db.GetDb()
		if err != nil {
			jerr.Get("error opening db", err).Fatal()
		}
		defer db.Close()
		if _, err := tweets.GetAllTweets(accountKey.Account, client); err != nil {
			jerr.Get("error getting all tweets", err).Fatal()
		}
		wlt := wallet.NewWallet(accountKey.Address, accountKey.Key, db)
		if _, err = tweets.Transfer(accountKey, db, link, date, wlt); err != nil {
			jerr.Get("fatal error transferring tweets", err).Fatal()
		}
	},
}
