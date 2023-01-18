package database

import (
	"encoding/json"
	"fmt"
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/tweets/obj"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"html"
	"strconv"
)

func SaveTweet(wlt Wallet, accountKey obj.AccountKey, tweet obj.TweetTx, db *leveldb.DB, appendLink bool, appendDate bool) error {
	if tweet.Tweet == nil {
		return jerr.New("tweet is nil")
	}
	tweetLink := fmt.Sprintf("\nhttps://twitter.com/twitter/status/%d\n", tweet.Tweet.ID)
	tweetDate := fmt.Sprintf("\n%s\n", tweet.Tweet.CreatedAt)
	tweetText := tweet.Tweet.Text
	if appendLink {
		tweetText += tweetLink
	}
	if appendDate {
		tweetText += tweetDate
	}
	//add the tweet to the twitter-twittername-tweetID prefix
	prefix := fmt.Sprintf("tweets-%s-%019d", tweet.Tweet.User.ScreenName, tweet.Tweet.ID)
	tweetTx, _ := json.Marshal(tweet)
	db.Put([]byte(prefix), tweetTx, nil)
	//if the tweet was a regular post, post it normally
	if tweet.Tweet.InReplyToStatusID == 0 {
		parentHash, err := MakePost(wlt, html.UnescapeString(tweetText))
		//find this tweet in archive and set its hash to the hash of the post that was just made
		tweet.TxHash = parentHash
		if err != nil {
			return jerr.Get("error making post", err)
		}
	} else {
		var parentHash []byte = nil
		//search the saved-address-twittername-tweetID prefix for the tweet that this tweet is a reply to
		prefix := fmt.Sprintf("saved-%s-%s", accountKey.Address, tweet.Tweet.User.ScreenName)
		iter := db.NewIterator(util.BytesPrefix([]byte(prefix)), nil)
		for iter.Next() {
			key := iter.Key()
			tweetID, _ := strconv.ParseInt(string(key[len(prefix)+1:]), 10, 64)
			if tweetID == tweet.Tweet.InReplyToStatusID {
				parentHash = iter.Value()
				break
			}
		}
		//if it turns out this tweet was actually a reply to another person's tweet, post it as a regular post
		if parentHash == nil {
			parentHash, err := MakePost(wlt, html.UnescapeString(tweetText))
			//find this tweet in archive and set its hash to the hash of the post that was just made
			tweet.TxHash = parentHash
			if err != nil {
				return jerr.Get("error making post", err)
			}
			//otherwise, it's part of a thread, so post it as a reply to the parent tweet
		} else {
			replyHash, err := MakeReply(wlt, parentHash, html.UnescapeString(tweetText))
			//find this tweet in archive and set its hash to the hash of the post that was just made
			tweet.TxHash = replyHash
			if err != nil {
				return jerr.Get("error making reply", err)
			}
		}
	}
	//save the txHash to the saved-address-twittername-tweetID prefix
	prefix = fmt.Sprintf("saved-%s-%s", accountKey.Address, tweet.Tweet.User.ScreenName)
	dbKey := fmt.Sprintf("%s-%019d", prefix, tweet.Tweet.ID)
	err := db.Put([]byte(dbKey), tweet.TxHash, nil)
	if err != nil {
		return jerr.Get("error saving tweetTx", err)
	}
	return nil
}
