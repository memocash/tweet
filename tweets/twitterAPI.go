package tweets

import (
	"encoding/json"
	"fmt"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/jchavannes/jgo/jerr"
	config2 "github.com/memocash/tweet/config"
	"github.com/memocash/tweet/tweets/obj"
	"github.com/syndtr/goleveldb/leveldb"
	util2 "github.com/syndtr/goleveldb/leveldb/util"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"log"
	"strconv"
)

func GetAllTweets(screenName string, client *twitter.Client, db *leveldb.DB) (int, error) {
	var numTweets = 0
	for {
		tweets, err := getOldTweets(screenName, client, db)
		if err != nil {
			return numTweets, jerr.Get("error getting old tweets", err)
		}
		if len(tweets) == 1 {
			return numTweets, nil
		}
		numTweets += len(tweets)
	}
}

func getOldTweets(screenName string, client *twitter.Client, db *leveldb.DB) ([]obj.TweetTx, error) {
	var userTimelineParams *twitter.UserTimelineParams
	excludeReplies := false
	//check if there are any tweetTx objects with the prefix containing this address and this screenName
	prefix := fmt.Sprintf("tweets-%s", screenName)
	iter := db.NewIterator(util2.BytesPrefix([]byte(prefix)), nil)
	tweetsFound := iter.First()
	iter.Release()
	var maxID int64
	if tweetsFound {
		//get the newest tweet in the saved_address_tweetID
		iter := db.NewIterator(util2.BytesPrefix([]byte(prefix)), nil)
		maxID = 0
		for iter.Next() {
			key := iter.Key()
			tweetID, _ := strconv.ParseInt(string(key[len(prefix)+1:]), 10, 64)
			if tweetID < maxID || maxID == 0 {
				maxID = tweetID
			}
		}
		iter.Release()
		userTimelineParams = &twitter.UserTimelineParams{ScreenName: screenName, ExcludeReplies: &excludeReplies, MaxID: maxID, Count: 100}
	} else {
		userTimelineParams = &twitter.UserTimelineParams{ScreenName: screenName, ExcludeReplies: &excludeReplies, Count: 100}
	}
	// Query to Twitter API for all tweets after IdInfo.id
	tweets, _, err := client.Timelines.UserTimeline(userTimelineParams)
	if err != nil {
		return nil, jerr.Get("error getting old tweets from user timeline", err)
	}
	var tweetTxs []obj.TweetTx
	for i, tweet := range tweets {
		prefix := fmt.Sprintf("tweets-%s-%019d", screenName, tweet.ID)
		tweetTx, _ := json.Marshal(obj.TweetTx{Tweet: &tweets[i], TxHash: nil})
		if err := db.Put([]byte(prefix), tweetTx, nil); err != nil {
			return nil, jerr.Get("error saving old tweet", err)
		}
		tweetTxs = append(tweetTxs, obj.TweetTx{Tweet: &tweets[i], TxHash: nil})
	}
	return tweetTxs, nil
}
func getNewTweets(screenName string, client *twitter.Client, db *leveldb.DB) ([]obj.TweetTx, error) {
	var userTimelineParams *twitter.UserTimelineParams
	excludeReplies := false
	//check if there are any tweetTx objects with the prefix containing this address and this screenName
	prefix := fmt.Sprintf("tweets-%s", screenName)
	iter := db.NewIterator(util2.BytesPrefix([]byte(prefix)), nil)
	tweetsFound := iter.First()
	iter.Release()
	var maxID int64
	if tweetsFound {
		//get the newest tweet in the saved_address_tweetID
		iter := db.NewIterator(util2.BytesPrefix([]byte(prefix)), nil)
		maxID = 0
		for iter.Next() {
			key := iter.Key()
			tweetID, _ := strconv.ParseInt(string(key[len(prefix)+1:]), 10, 64)
			if tweetID > maxID{
				maxID = tweetID
			}
		}
		iter.Release()
		userTimelineParams = &twitter.UserTimelineParams{ScreenName: screenName, ExcludeReplies: &excludeReplies, SinceID: maxID, Count: 100}
	}
	// Query to Twitter API for all tweets after IdInfo.id
	tweets, _, err := client.Timelines.UserTimeline(userTimelineParams)
	if err != nil {
		return nil, jerr.Get("error getting old tweets from user timeline", err)
	}
	var tweetTxs []obj.TweetTx
	for i, tweet := range tweets {
		prefix := fmt.Sprintf("tweets-%s-%019d", screenName, tweet.ID)
		tweetTx, _ := json.Marshal(obj.TweetTx{Tweet: &tweets[i], TxHash: nil})
		if err := db.Put([]byte(prefix), tweetTx, nil); err != nil {
			return nil, jerr.Get("error saving tweet posted since last time the stream was opened", err)
		}
		tweetTxs = append(tweetTxs, obj.TweetTx{Tweet: &tweets[i], TxHash: nil})
	}
	return tweetTxs, nil
}

func GetSkippedTweets(accountKey obj.AccountKey, client *twitter.Client, db *leveldb.DB, link bool, date bool) error {
	//check if there are any transferred tweets with the prefix containing this address and this screenName
	savedPrefix := fmt.Sprintf("saved-%s-%s", accountKey.Address, accountKey.Account)
	iter := db.NewIterator(util2.BytesPrefix([]byte(savedPrefix)), nil)
	tweetsFound := iter.First()
	iter.Release()
	if !tweetsFound {
		return nil
	}
	txList, err := getNewTweets(accountKey.Account, client, db)
	if err != nil {
		return jerr.Get("error getting tweets since the bot was last run", err)
	}
	numLeft := len(txList)
	for numLeft > 0 {
		if _, err = Transfer(accountKey, db, link, date); err != nil {
			return jerr.Get("fatal error transferring tweets", err)
		}
		numLeft -= 20
	}
	return nil
}
func Connect() *twitter.Client {
	conf := config2.GetTwitterAPIConfig()
	if !conf.IsSet() {
		log.Fatal("Application Access Token required")
	}
	// oauth2 configures a client that uses app credentials to keep a fresh token
	config := &clientcredentials.Config{
		ClientID:     conf.ConsumerKey,
		ClientSecret: conf.ConsumerSecret,
		TokenURL:     "https://api.twitter.com/oauth2/token",
	}
	// http.Client will automatically authorize Requests
	httpClient := config.Client(oauth2.NoContext)

	// Twitter client
	client := twitter.NewClient(httpClient)
	return client
}
