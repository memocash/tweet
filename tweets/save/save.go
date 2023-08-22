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
	var tweetMedia string
	if tweet.Entities != nil && len(tweet.Entities.Media) > 0 {
		tweetMedia = tweet.Entities.Media[0].MediaURL
	}
	tweetText := &Text{
		Text:     tweet.Text,
		Link:     fmt.Sprintf("https://twitter.com/%s/status/%d", tweet.User.ScreenName, tweet.ID),
		Date:     tweet.CreatedAt,
		Media:    tweetMedia,
		FlagLink: flags.Link,
		FlagDate: flags.Date,
	}
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
		message := tweetText.Gen(memo.OldMaxPostSize)
		logMsg = "post (regular)"
		parentHash, err := wallet.MakePost(wlt, html.UnescapeString(message))
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
			message := tweetText.Gen(memo.OldMaxPostSize)
			logMsg = "post (reply parent not found)"
			txHash, err = wallet.MakePost(wlt, html.UnescapeString(message))
			if err != nil {
				return jerr.Get("error making post", err)
			}
		} else {
			message := tweetText.Gen(memo.OldMaxReplySize)
			logMsg = "reply"
			txHash, err = wallet.MakeReply(wlt, parentSavedTweet.TxHash, html.UnescapeString(message))
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
