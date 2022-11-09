package util

import (
	"fmt"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/cmd/util"
	"github.com/memocash/tweet/database"
	"html"
)

func TransferTweets(client *twitter.Client, address wallet.Address, key wallet.PrivateKey,account string, archive util.TweetObject, appendLink bool, appendDate bool) (int, error) {
	var tweetList []twitter.Tweet
	//if there are at least 20 tweets not yet archived, get the oldest 20, otherwise just get all of them
	if len(archive.TweetList) - archive.Archived >= 20 {
		tweetList = archive.TweetList[len(archive.TweetList)-archive.Archived-20:len(archive.TweetList)-archive.Archived]
	} else{
		tweetList = archive.TweetList
	}
	//reverse tweetList so that tweets are posted in chronological order in memo, instead of reverse chronological
	for i := len(tweetList)/2 - 1; i >= 0; i-- {
		opp := len(tweetList) - 1 - i
		tweetList[i], tweetList[opp] = tweetList[opp], tweetList[i]
	}
	numTransferred := 0
	//post each tweet that isn't a reply to some other tweet
	for _, tweet := range tweetList {
		if tweet.InReplyToStatusID == 0 {
			tweetLink := fmt.Sprintf("\nhttps://twitter.com/twitter/status/%d\n", tweet.ID)
			tweetDate := fmt.Sprintf("\n%s\n", tweet.CreatedAt)
			tweetText := tweet.Text
			if appendLink {
				tweetText += tweetLink
			}
			if appendDate {
				tweetText += tweetDate
			}
			parentHash, err := database.MakePost(address, key, html.UnescapeString(tweetText))
			if err != nil {
				return numTransferred,jerr.Get("error making post", err)
			}
			//post all replies to the current tweet, no matter how recent they are
			err = recursiveReplies(parentHash, tweet, address, key, archive, appendLink, appendDate)
			if err != nil {
				return numTransferred,jerr.Get("error getting replies", err)
			}
		}
		numTransferred += 1
	}
	return numTransferred,nil
}
func recursiveReplies(parentHash []byte, tweet twitter.Tweet, address wallet.Address, key wallet.PrivateKey, archive util.TweetObject, appendLink bool, appendDate bool) error {
	replies := archive.TweetList
	for _, reply := range replies {
		//if the reply isn't a reply to the current tweet, skip it
		if reply.InReplyToStatusID != tweet.ID{
			continue
		}
		replyLink := fmt.Sprintf("\nhttps://twitter.com/twitter/status/%d\n",reply.ID)
		replyDate := fmt.Sprintf("\n%s\n",reply.CreatedAt)
		replyText := reply.Text
		if appendLink {
			replyText += replyLink
		}
		if appendDate {
			replyText += replyDate
		}
		parentHash,err := database.MakeReply(parentHash,address,key, html.UnescapeString(replyText))
		//recursively call this same function to post all replies to the current reply
		err = recursiveReplies(parentHash, reply, address, key, archive, appendLink, appendDate)
		if err != nil {
			return jerr.Get("error making reply", err)
		}
	}
	return nil
}
