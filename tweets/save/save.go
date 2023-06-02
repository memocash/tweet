package save

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/jchavannes/btcd/chaincfg/chainhash"
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/index/ref/bitcoin/memo"
	wallet2 "github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/db"
	"github.com/memocash/tweet/tweets/obj"
	"github.com/memocash/tweet/wallet"
	"github.com/syndtr/goleveldb/leveldb"
	"html"
	"log"
)

func Tweet(wlt wallet.Wallet, address string, tweet *twitter.Tweet, flags db.Flags) error {
	if tweet == nil {
		return jerr.New("tweet is nil")
	}
	existingSavedAddressTweet, err := db.GetSavedAddressTweet(wallet2.GetAddressFromString(address).GetAddr(),
		tweet.User.ID, tweet.ID)
	if err != nil && !errors.Is(err, leveldb.ErrNotFound) {
		return jerr.Get("error getting existing saved address tweet for save", err)
	}
	if existingSavedAddressTweet != nil {
		return nil
	}
	tweetLink := fmt.Sprintf("\nhttps://twitter.com/%s/status/%d\n", tweet.User.ScreenName, tweet.ID)
	tweetDate := fmt.Sprintf("\n%s\n", tweet.CreatedAt)
	tweetText := tweet.Text
	tweetJson, err := json.Marshal(obj.TweetTx{Tweet: tweet})
	if err != nil {
		return jerr.Get("error marshaling tweetTx", err)
	}
	if err := db.Save([]db.ObjectI{&db.TweetTx{
		UserID:  tweet.User.ID,
		TweetId: tweet.ID,
		Tx:      tweetJson,
	}}); err != nil {
		return jerr.Get("error saving tweetTx db object", err)
	}
	var logMsg string
	var txHash chainhash.Hash
	if tweet.InReplyToStatusID == 0 {
		tweetText = trimTweet(tweetText, tweetLink, tweetDate, flags.Link, flags.Date, memo.OldMaxPostSize)
		logMsg = "post (regular)"
		parentHash, err := wallet.MakePost(wlt, html.UnescapeString(tweetText))
		if err != nil {
			return jerr.Get("error making post", err)
		}
		txHash = parentHash
	} else {
		parentSavedTweet, err := db.GetSavedAddressTweet(
			wallet2.GetAddressFromString(address).GetAddr(), tweet.User.ID, tweet.InReplyToStatusID)
		if err != nil && !errors.Is(err, leveldb.ErrNotFound) {
			return jerr.Get("error getting saved address tweet for tweet parent reply", err)
		}
		if parentSavedTweet == nil {
			tweetText = trimTweet(tweetText, tweetLink, tweetDate, flags.Link, flags.Date, memo.OldMaxPostSize)
			logMsg = "post (reply parent not found)"
			txHash, err = wallet.MakePost(wlt, html.UnescapeString(tweetText))
			if err != nil {
				return jerr.Get("error making post", err)
			}
		} else {
			tweetText = trimTweet(tweetText, tweetLink, tweetDate, flags.Link, flags.Date, memo.OldMaxReplySize)
			logMsg = "reply"
			txHash, err = wallet.MakeReply(wlt, parentSavedTweet.TxHash, html.UnescapeString(tweetText))
			if err != nil {
				return jerr.Get("error making reply", err)
			}
		}
	}
	log.Printf("broadcasted tweet tx: %s (%s) - %s\n", txHash, tweet.User.ScreenName, logMsg)
	if err := db.Save([]db.ObjectI{&db.SavedAddressTweet{
		Address: wallet2.GetAddressFromString(address).GetAddr(),
		UserID:  tweet.User.ID,
		TweetId: tweet.ID,
		TxHash:  txHash[:],
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
