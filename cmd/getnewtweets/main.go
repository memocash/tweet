package getnewtweets

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/cmd/util"
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
		var errChan = make(chan error)
		for _, streamConfig := range streamConfigs {
			go func(config config.Stream) {
				key, address, account := util.Setup([]string{config.Key, config.Name})
				tweetstream.ResetRules(streamToken)
				tweetstream.FilterAccount(streamToken, account)
				tweetstream.InitiateStream(streamToken, address,key, db)
				tweetstream.ResetRules(streamToken)
				if err != nil {
					errChan <- jerr.Get("error getting stream token", err)
				}
			}(streamConfig)
		}
		return <- errChan
	},
}

func GetCommand() *cobra.Command {
	//if link and date are supplied, the tweet will be linked and the date will be added to the memo post
	transferCmd.PersistentFlags().BoolVarP(&link, "link", "l", false, "link to tweet")
	transferCmd.PersistentFlags().BoolVarP(&date, "date", "d", false, "add date to post")
	return transferCmd
}
