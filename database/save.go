package database

import (
	"encoding/json"
	"fmt"
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/index/ref/bitcoin/memo"
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
	//get the length of tweetText in bytes
	//add the tweet to the twitter-twittername-tweetID prefix
	prefix := fmt.Sprintf("tweets-%s-%019d", tweet.Tweet.User.ScreenName, tweet.Tweet.ID)
	tweetTx, _ := json.Marshal(tweet)
	db.Put([]byte(prefix), tweetTx, nil)
	prefix = fmt.Sprintf("saved-%s-%s", accountKey.Address, tweet.Tweet.User.ScreenName)
	dbKey := fmt.Sprintf("%s-%019d", prefix, tweet.Tweet.ID)
	found, err := db.Has([]byte(dbKey), nil)
	if err != nil{
		return jerr.Get("error checking if tweet is already saved", err)
	}
	if found {
		//database has already saved this tweet
		return nil
	}
	//if the tweet was a regular post, post it normally
	if tweet.Tweet.InReplyToStatusID == 0 {
		tweetText = trimTweet(tweetText, tweetLink, tweetDate, appendLink, appendDate, memo.OldMaxPostSize)
		println("making post (twitter post was not a reply)")
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
			tweetText = trimTweet(tweetText, tweetLink, tweetDate, appendLink, appendDate, memo.OldMaxPostSize)
			println("making post (reply parent not found)")
			parentHash, err := MakePost(wlt, html.UnescapeString(tweetText))
			//find this tweet in archive and set its hash to the hash of the post that was just made
			tweet.TxHash = parentHash
			if err != nil {
				return jerr.Get("error making post", err)
			}
			//otherwise, it's part of a thread, so post it as a reply to the parent tweet
		} else {
			tweetText = trimTweet(tweetText, tweetLink, tweetDate, appendLink, appendDate, memo.OldMaxReplySize)
			println("making reply")
			replyHash, err := MakeReply(wlt, parentHash, html.UnescapeString(tweetText))
			//find this tweet in archive and set its hash to the hash of the post that was just made
			tweet.TxHash = replyHash
			if err != nil {
				return jerr.Get("error making reply", err)
			}
		}
	}
	//save the txHash to the saved-address-twittername-tweetID prefix
	err = db.Put([]byte(dbKey), tweet.TxHash, nil)
	if err != nil {
		return jerr.Get("error saving tweetTx", err)
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
