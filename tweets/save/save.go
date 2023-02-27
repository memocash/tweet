package save

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jchavannes/jgo/jerr"
	"github.com/jchavannes/jgo/jlog"
	"github.com/memocash/index/ref/bitcoin/memo"
	"github.com/memocash/tweet/db"
	"github.com/memocash/tweet/tweets/obj"
	"github.com/memocash/tweet/wallet"
	"github.com/syndtr/goleveldb/leveldb"
	"html"
)

func Tweet(wlt wallet.Wallet, address string, tweet obj.TweetTx, flags db.Flags) error {
	if tweet.Tweet == nil {
		return jerr.New("tweet is nil")
	}
	existingSavedAddressTweet, err := db.GetSavedAddressTweet(address, tweet.Tweet.User.ScreenName, tweet.Tweet.ID)
	if err != nil && !errors.Is(err, leveldb.ErrNotFound) {
		return jerr.Get("error getting existing saved address tweet for save", err)
	}
	if existingSavedAddressTweet != nil {
		return nil
	}
	tweetLink := fmt.Sprintf("\nhttps://twitter.com/twitter/status/%d\n", tweet.Tweet.ID)
	tweetDate := fmt.Sprintf("\n%s\n", tweet.Tweet.CreatedAt)
	tweetText := tweet.Tweet.Text
	tweetJson, err := json.Marshal(tweet)
	if err != nil {
		return jerr.Get("error marshaling tweetTx", err)
	}
	if err := db.Save([]db.ObjectI{&db.TweetTx{
		ScreenName: tweet.Tweet.User.ScreenName,
		TweetId:    tweet.Tweet.ID,
		Tx:         tweetJson,
	}}); err != nil {
		return jerr.Get("error saving tweetTx db object", err)
	}
	if tweet.Tweet.InReplyToStatusID == 0 {
		tweetText = trimTweet(tweetText, tweetLink, tweetDate, flags.Link, flags.Date, memo.OldMaxPostSize)
		jlog.Log("making post (twitter post was not a reply)")
		parentHash, err := wallet.MakePost(wlt, html.UnescapeString(tweetText))
		if err != nil {
			return jerr.Get("error making post", err)
		}
		tweet.TxHash = parentHash
	} else {
		parentSavedTweet, err := db.GetSavedAddressTweet(
			address, tweet.Tweet.User.ScreenName, tweet.Tweet.InReplyToStatusID)
		if err != nil && !errors.Is(err, leveldb.ErrNotFound) {
			return jerr.Get("error getting saved address tweet for tweet parent reply", err)
		}
		//if it turns out this tweet was actually a reply to another person's tweet, post it as a regular post
		if parentSavedTweet == nil {
			tweetText = trimTweet(tweetText, tweetLink, tweetDate, flags.Link, flags.Date, memo.OldMaxPostSize)
			jlog.Log("making post (reply parent not found)")
			parentHash, err := wallet.MakePost(wlt, html.UnescapeString(tweetText))
			//find this tweet in archive and set its hash to the hash of the post that was just made
			tweet.TxHash = parentHash
			if err != nil {
				return jerr.Get("error making post", err)
			}
			//otherwise, it's part of a thread, so post it as a reply to the parent tweet
		} else {
			tweetText = trimTweet(tweetText, tweetLink, tweetDate, flags.Link, flags.Date, memo.OldMaxReplySize)
			jlog.Log("making reply")
			replyHash, err := wallet.MakeReply(wlt, parentSavedTweet.TxHash, html.UnescapeString(tweetText))
			//find this tweet in archive and set its hash to the hash of the post that was just made
			tweet.TxHash = replyHash
			if err != nil {
				return jerr.Get("error making reply", err)
			}
		}
	}
	if err := db.Save([]db.ObjectI{&db.SavedAddressTweet{
		Address:    address,
		ScreenName: tweet.Tweet.User.ScreenName,
		TweetId:    tweet.Tweet.ID,
		TxHash:     tweet.TxHash,
	}}); err != nil {
		return jerr.Get("error saving saved address tweet", err)
	}
	return nil
}

func trimTweet(tweetText string, tweetLink string, tweetDate string, appendLink bool, appendDate bool, size int) string {
	if appendLink {
		if len([]byte(tweetText))+len([]byte(tweetLink)) > size {
			//if the tweet is too long, trim it
			tweetText = string([]byte(tweetText)[:size-len([]byte(tweetLink))])
		}
		tweetText += tweetLink
	}
	if appendDate {
		if len([]byte(tweetText))+len([]byte(tweetDate)) > size {
			//if the tweet is too long, trim it
			tweetText = string([]byte(tweetText)[:size-len([]byte(tweetDate))])
		}
		tweetText += tweetDate
	}
	return tweetText
}
