package transfertweets

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/cmd/util"
	util2 "github.com/memocash/tweet/database/util"
	"github.com/memocash/tweet/tweets"
	"github.com/spf13/cobra"
	"github.com/syndtr/goleveldb/leveldb"
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
	RunE: func(c *cobra.Command, args []string) error {
		key,address,account := util.Setup(args)
		client := tweets.Connect()
		fileName := "tweets.db"
		db, err := leveldb.OpenFile(fileName, nil)
		if err != nil{
			return jerr.Get("error opening db", err)
		}
		//if tweetArchive.json doesn't exist, initialize it
		tweets.GetAllTweets(account,client,db)
		//get the ID of the newest tweet that's already been archived
		_,_ = util2.TransferTweets(address, key, account, db, link, date)
		db.Close()
		return nil
	},
}


func GetCommand() *cobra.Command {
	//if link and date are supplied, the tweet will be linked and the date will be added to the memo post
	transferCmd.PersistentFlags().BoolVarP(&link, "link", "l", false, "link to tweet")
	transferCmd.PersistentFlags().BoolVarP(&date, "date", "d", false, "add date to post")
	return transferCmd
	}

