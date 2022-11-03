package transfertweets

import (
	"encoding/json"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/index/ref/bitcoin/util/testing/test_tx"
	"github.com/memocash/tweet/database"
	"github.com/memocash/tweet/tweets"
	"github.com/spf13/cobra"
	"io/ioutil"
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
		client := tweets.Connect()
		type tweetObject struct {
			TweetList []twitter.Tweet
			Archived  int
		}
		var archive tweetObject
		var tweetList []twitter.Tweet
		content, err := ioutil.ReadFile("./tweetArchive.json")
		if err == nil{
			err = json.Unmarshal(content, &archive)
		}
		if err != nil{
			tweetList = tweets.GetAllTweets(account,client)
			archive.TweetList = tweetList
			archive.Archived = 0
		}
		//len - Archived - 20 to len - Archived
		tweetList = archive.TweetList[len(archive.TweetList)-archive.Archived-20:len(archive.TweetList)-archive.Archived]

		//reverse tweetList
		for i := len(tweetList)/2 - 1; i >= 0; i-- {
			opp := len(tweetList) - 1 - i
			tweetList[i], tweetList[opp] = tweetList[opp], tweetList[i]
		}
		err = database.TransferTweets(address, key, tweetList, link, date)
		archive.Archived += 20
		if err != nil {
			return jerr.Get("error", err)
		}
		file,_ := json.MarshalIndent(archive, "", " ")
		_ = ioutil.WriteFile("tweetArchive.json", file, 0644)
		return nil
	},
}


func GetCommand() *cobra.Command {
	transferCmd.PersistentFlags().BoolVarP(&link, "link", "l", false, "link to tweet")
	transferCmd.PersistentFlags().BoolVarP(&date, "date", "d", false, "add date to post")
	return transferCmd
	}

