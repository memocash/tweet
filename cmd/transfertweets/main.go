package transfertweets

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/index/ref/bitcoin/util/testing/test_tx"
	"github.com/memocash/tweet/database"
	"github.com/memocash/tweet/tweets"
	"github.com/spf13/cobra"
)

var link bool = false
var date bool = false

var transferCmd = &cobra.Command{
	Use:   "transfertweets",
	Short: "Transfer tweets to memo posts",
	Args: cobra.ExactArgs(2),
	RunE: func(c *cobra.Command, args []string) error {
		key := test_tx.GetPrivateKey(args[0])
		address := key.GetAddress()
		account := args[1]
		tweetList := tweets.GetTweets(account)
		err := database.TransferTweets(address, key, tweetList, link, date)
		if err != nil {
			return jerr.Get("error", err)
		}
		return nil
	},
}


func GetCommand() *cobra.Command {
	transferCmd.PersistentFlags().BoolVarP(&link, "link", "l", false, "link to tweet")
	transferCmd.PersistentFlags().BoolVarP(&date, "date", "d", false, "add date to post")
	return transferCmd
	}

