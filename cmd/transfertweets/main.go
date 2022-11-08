package transfertweets

import (
	"encoding/json"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/cmd/util"
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
	Long: "The first time this command is run, it will populate tweetArchive.json with all of the tweets" +
		" from the twitter account and make the oldest 20 into memo posts. After it will transfer 20 each time" +
		"it is run. Deleting the tweetArchive.json file will cause it to restart from the beginning.",
	Args: cobra.ExactArgs(2),
	RunE: func(c *cobra.Command, args []string) error {
		key,address, account := util.Setup(args)
		client := tweets.Connect()
		//Structure of tweetArchive.json
		type tweetObject struct {
			TweetList []twitter.Tweet
			//number of tweets already archived
			Archived  int
		}
		var archive tweetObject
		var tweetList []twitter.Tweet
		content, err := ioutil.ReadFile("./tweetArchive.json")
		if err == nil{
			err = json.Unmarshal(content, &archive)
		}
		//if tweetArchive.json doesn't exist, initialize it
		if err != nil{
			tweetList = tweets.GetAllTweets(account,client)
			archive.TweetList = tweetList
			archive.Archived = 0
		}
		//len - Archived - 20 to len - Archived (oldest 20 tweets not already archived)
		tweetList = archive.TweetList[len(archive.TweetList)-archive.Archived-20:len(archive.TweetList)-archive.Archived]

		//reverse tweetList so they are posted in chronological order in memo, instead of reverse chronological
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
	//if link and date are supplied, the tweet will be linked and the date will be added to the memo post
	transferCmd.PersistentFlags().BoolVarP(&link, "link", "l", false, "link to tweet")
	transferCmd.PersistentFlags().BoolVarP(&date, "date", "d", false, "add date to post")
	return transferCmd
	}

