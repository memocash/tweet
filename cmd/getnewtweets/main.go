package getnewtweets

import (
	"fmt"
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/cmd/util"
	util2 "github.com/memocash/tweet/database/util"
	"github.com/memocash/tweet/tweets"
	"github.com/spf13/cobra"
)

var link bool = false
var date bool = false

var transferCmd = &cobra.Command{
	Use:   "getnewtweets",
	Short: "Transfer tweets to memo posts",
	Long: "Similar to transfer tweets, but gets the 20 newest tweets. ",
	Args: cobra.ExactArgs(2),
	RunE: func(c *cobra.Command, args []string) error {
		key,address, account := util.Setup(args)
		client := tweets.Connect()
		_,_,_,userID := tweets.GetProfile(account,client)
		fileHeader := fmt.Sprintf("%s_%s", address, userID)
		tweetList := tweets.GetNewTweets(account,client,fileHeader)
		archive := util.Archive{
			TweetList: tweetList,
			Archived: 0,
		}
		_,err := util2.TransferTweets(address, key, archive, link, date)
		if err != nil {
			return jerr.Get("error", err)
		}
		return nil
	},
}


func GetCommand() *cobra.Command {
	//if link and date are supplied, the tweet will be linked and the date will be added to the memo post
	transferCmd.PersistentFlags().BoolVarP(&link, "link", "l", false, "link to tweet")
	transferCmd.PersistentFlags().BoolVarP(&date, "date", "d", false, "add date to post")
	return transferCmd
}

