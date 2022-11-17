package getnewtweets

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/cmd/util"
	"github.com/memocash/tweet/tweetstream"
	"github.com/spf13/cobra"
)

var link bool = false
var date bool = false

var transferCmd = &cobra.Command{
	Use:   "getnewtweets",
	Short: "Listens for new tweets on an account",
	Long: "Prints out each new tweet as it comes in. ",
	Args: cobra.ExactArgs(2),
	RunE: func(c *cobra.Command, args []string) error {
		_,_, account := util.Setup(args)
		//client := tweets.Connect()
		//_,_,_,userID := tweets.GetProfile(account,client)
		//fileHeader := fmt.Sprintf("%s_%s", address, userID)
		streamToken, err:= tweetstream.GetStreamingToken()
		tweetstream.ResetRules(streamToken)
		tweetstream.FilterAccount(streamToken, account)
		tweetstream.InitiateStream(streamToken)
		tweetstream.ResetRules(streamToken)
		if err != nil{
			return jerr.Get("error getting stream token", err)
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

