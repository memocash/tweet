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

func GetAllTweets(screenName string, client *twitter.Client, db *leveldb.DB) error {
	for {
		tweets, err := GetOldTweets(screenName, client, db)
		if err != nil {
			return jerr.Get("error getting old tweets", err)
		}
		if len(tweets) == 1 {
			return nil
		}
	}
}

func GetOldTweets(screenName string, client *twitter.Client, db *leveldb.DB) ([]obj.TweetTx, error) {
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
func GetNewTweets(screenName string, client *twitter.Client, db *leveldb.DB) ([]obj.TweetTx, error) {
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
