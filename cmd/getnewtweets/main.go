package getnewtweets

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/config"
	"github.com/memocash/tweet/tweetstream"
	"github.com/spf13/cobra"
	"github.com/syndtr/goleveldb/leveldb"
)

var link bool = false
var date bool = false

var transferCmd = &cobra.Command{
	Use:   "getnewtweets",
	Short: "Listens for new tweets on an account",
	Long:  "Prints out each new tweet as it comes in. ",
	RunE: func(c *cobra.Command, args []string) error {
		streamToken, err := tweetstream.GetStreamingToken()
		fileName := "tweets.db"
		db, err := leveldb.OpenFile(fileName, nil)
		if err != nil {
			return jerr.Get("error opening db", err)
		}
		streamConfigs := config.GetConfig().Streams
		tweetstream.ResetRules(streamToken)
		tweetstream.FilterAccount(streamToken, streamConfigs)
		tweetstream.InitiateStream(streamToken, streamConfigs, db)
		tweetstream.ResetRules(streamToken)
		return nil
	},
}

func GetCommand() *cobra.Command {
	//if link and date are supplied, the tweet will be linked and the date will be added to the memo post
	transferCmd.PersistentFlags().BoolVarP(&link, "link", "l", false, "link to tweet")
	transferCmd.PersistentFlags().BoolVarP(&date, "date", "d", false, "add date to post")
	return transferCmd
}
