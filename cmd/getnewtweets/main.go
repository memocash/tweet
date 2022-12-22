package getnewtweets

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/config"
	"github.com/memocash/tweet/database"
	"github.com/memocash/tweet/tweets"
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
