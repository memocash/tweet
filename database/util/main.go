package util

import (
	"encoding/json"
	"fmt"
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/cmd/util"
	"github.com/memocash/tweet/database"
	"github.com/syndtr/goleveldb/leveldb"
	util3 "github.com/syndtr/goleveldb/leveldb/util"
	"html"
	"strconv"
)

func TransferTweets(address wallet.Address, key wallet.PrivateKey, screenName string, db *leveldb.DB, appendLink bool, appendDate bool) (int, error) {
	var tweetList []util.TweetTx
	//find the greatest ID of all the already saved tweets
	prefix := fmt.Sprintf("saved-%s-%s",address, screenName)
	iter := db.NewIterator(util3.BytesPrefix([]byte(prefix)), nil)
	var startID int64 = 0
	for iter.Next() {
		key := iter.Key()
		tweetID,_ := strconv.ParseInt(string(key[len(prefix)+1:]), 10, 64)
		if tweetID > startID || startID == 0 {
			startID = tweetID
		}
	}
	iter.Release()
	//get up to 20 tweets from the tweets-twittername-tweetID prefix with the smallest IDs greater than the startID
	prefix = fmt.Sprintf("tweets-%s", screenName)
	println(prefix)
	iter = db.NewIterator(util3.BytesPrefix([]byte(prefix)), nil)
	println("\n\n\n\nstartID: %d",startID)
	for iter.Next() {
		key := iter.Key()
		tweetID,_ := strconv.ParseInt(string(key[len(prefix)+1:]), 10, 64)
		println("%d",tweetID)
		if tweetID > startID {
			var tweetTx util.TweetTx
			err := json.Unmarshal(iter.Value(), &tweetTx)
			if err != nil {
				return 0, jerr.Get("error unmarshaling tweetTx", err)
			}
			tweetList = append(tweetList, tweetTx)
			println(tweetTx.Tweet.Text)
			if len(tweetList) == 20 {
				break
			}
		}
	}
	//reverse tweetList so that tweets are posted in chronological order in memo, instead of reverse chronological
	for i := len(tweetList)/2 - 1; i >= 0; i-- {
		opp := len(tweetList) - 1 - i
		tweetList[i], tweetList[opp] = tweetList[opp], tweetList[i]
	}
	numTransferred := 0
	wlt := database.NewWallet(address, key)
	for _, tweet := range tweetList {
		if tweet.Tweet == nil {
			return numTransferred, jerr.Newf("tweet is nil, verify using correct archive: %#v", tweet)
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
		//if the tweet was a regular post, post it normally
		if tweet.Tweet.InReplyToStatusID == 0 {
			parentHash, err := database.MakePost(wlt, html.UnescapeString(tweetText))
			//find this tweet in archive and set its hash to the hash of the post that was just made
			tweet.TxHash = parentHash
			if err != nil {
				return numTransferred, jerr.Get("error making post", err)
			}
		} else {
			var parentHash []byte = nil
			//search the saved-address-twittername-tweetID prefix for the tweet that this tweet is a reply to
			prefix = fmt.Sprintf("saved-%s-%s", address, screenName)
			iter = db.NewIterator(util3.BytesPrefix([]byte(prefix)), nil)
			for iter.Next() {
				key := iter.Key()
				tweetID,_ := strconv.ParseInt(string(key[len(prefix)+1:]), 10, 64)
				if tweetID == tweet.Tweet.InReplyToStatusID {
					var tweetTx util.TweetTx
					err := json.Unmarshal(iter.Value(), &tweetTx)
					if err != nil {
						return numTransferred, jerr.Get("error unmarshaling tweetTx", err)
					}
					parentHash = tweetTx.TxHash
					break
				}
			}
			//if it turns out this tweet was actually a reply to another person's tweet, post it as a regular post
			if parentHash == nil {
				parentHash, err := database.MakePost(wlt, html.UnescapeString(tweetText))
				//find this tweet in archive and set its hash to the hash of the post that was just made
				tweet.TxHash = parentHash
				if err != nil {
					return numTransferred, jerr.Get("error making post", err)
				}
				//otherwise, it's part of a thread, so post it as a reply to the parent tweet
			} else {
				replyHash, err := database.MakeReply(wlt, parentHash, html.UnescapeString(tweetText))
				//find this tweet in archive and set its hash to the hash of the post that was just made
				tweet.TxHash = replyHash
				if err != nil {
					return numTransferred, jerr.Get("error making reply", err)
				}
			}
		}
		//save the tweet to the saved-address-twittername-tweetID prefix
		prefix = fmt.Sprintf("saved-%s-%s", address, screenName)
		key := fmt.Sprintf("%s-%d", prefix, tweet.Tweet.ID)
		value, err := json.Marshal(tweet)
		if err != nil {
			return numTransferred, jerr.Get("error marshaling tweetTx", err)
		}
		err = db.Put([]byte(key), value, nil)
		if err != nil {
			return numTransferred, jerr.Get("error saving tweetTx", err)
		}
		numTransferred++
	}
	return numTransferred, nil
}
