package util

import (
	"fmt"
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/cmd/util"
	"github.com/memocash/tweet/database"
	"html"
)

func TransferTweets(address wallet.Address, key wallet.PrivateKey,account string, archive util.Archive, appendLink bool, appendDate bool) (int, error) {
	var tweetList []util.TweetTx
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
		tweetLink := fmt.Sprintf("\nhttps://twitter.com/twitter/status/%d\n", tweet.Tweet.ID)
		tweetDate := fmt.Sprintf("\n%s\n", tweet.Tweet.CreatedAt)
		tweetText := tweet.Tweet.Text
		if appendLink {
			tweetText += tweetLink
		}
		if appendDate {
			tweetText += tweetDate
		}
		if tweet.Tweet.InReplyToStatusID == 0{
			parentHash, err := database.MakePost(address, key, html.UnescapeString(tweetText))
			//find this tweet in archive and set its parent hash to the hash of the post that was just made
			for i := range archive.TweetList {
				if archive.TweetList[i].Tweet.ID == tweet.Tweet.ID {
					archive.TweetList[i].TxHash = parentHash
					break
				}
			}
			tweet.TxHash = parentHash
			if err != nil {
				return numTransferred,jerr.Get("error making post", err)
			}
		} else{
			//find the parent tweet
			var parentHash []byte
			for _, parentTweet := range tweetList {
				if parentTweet.Tweet.ID == tweet.Tweet.InReplyToStatusID{
					parentHash = parentTweet.TxHash
				}
			}
			if parentHash == nil{
				parentHash, err := database.MakePost(address, key, html.UnescapeString(tweetText))
				//find this tweet in archive and set its parent hash to the hash of the post that was just made
				for i := range archive.TweetList {
					if archive.TweetList[i].Tweet.ID == tweet.Tweet.ID {
						archive.TweetList[i].TxHash = parentHash
						break
					}
				}
				tweet.TxHash = parentHash
				if err != nil {
					return numTransferred,jerr.Get("error making post", err)
				}
			}
			replyHash, err := database.MakeReply(parentHash,address, key, html.UnescapeString(tweetText))
			//find this tweet in archive and set its parent hash to the hash of the post that was just made
			for i := range archive.TweetList {
				if archive.TweetList[i].Tweet.ID == tweet.Tweet.ID {
					archive.TweetList[i].TxHash = replyHash
					break
				}
			}
			tweet.TxHash = replyHash
			if err != nil {
				return numTransferred,jerr.Get("error making reply", err)
			}
		}
		numTransferred += 1
	}
	return numTransferred,nil
}