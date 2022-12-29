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
	"time"
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
func getNewTweets(accountKey obj.AccountKey, client *twitter.Client, db *leveldb.DB) ([]obj.TweetTx, error) {
	var userTimelineParams *twitter.UserTimelineParams
	excludeReplies := false
	//check if there are any tweetTx objects with the prefix containing this address and this screenName
	prefix := fmt.Sprintf("tweets-%s", accountKey.Account)
	iter := db.NewIterator(util2.BytesPrefix([]byte(prefix)), nil)
	tweetsFound := iter.First()
	iter.Release()
	var maxID int64 = 0
	if tweetsFound {
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
	}
	// Query to Twitter API for all tweets after IdInfo.id
	userTimelineParams = &twitter.UserTimelineParams{ScreenName: accountKey.Account, ExcludeReplies: &excludeReplies, SinceID: maxID, Count: 100}
	tweets, _, err := client.Timelines.UserTimeline(userTimelineParams)
	if err != nil {
		return nil, jerr.Get("error getting old tweets from user timeline", err)
	}
	var tweetTxs []obj.TweetTx
	for i, tweet := range tweets {
		prefix := fmt.Sprintf("tweets-%s-%019d", accountKey.Account, tweet.ID)
		tweetTx, _ := json.Marshal(obj.TweetTx{Tweet: &tweets[i], TxHash: nil})
		if err := db.Put([]byte(prefix), tweetTx, nil); err != nil {
			return nil, jerr.Get("error saving tweet posted since last time the stream was opened", err)
		}
		tweetTxs = append(tweetTxs, obj.TweetTx{Tweet: &tweets[i], TxHash: nil})
	}
	return tweetTxs, nil
}
func getNumTweets(screenName string, db *leveldb.DB) int {
	numTweets := 0
	iter := db.NewIterator(util2.BytesPrefix([]byte(fmt.Sprintf("tweets-%s", screenName))), nil)
	for iter.Next() {
		numTweets++
	}
	iter.Release()
	return numTweets
}
func getNumSavedTweets(accountKey obj.AccountKey, db *leveldb.DB) int {
	numTweets := 0
	iter := db.NewIterator(util2.BytesPrefix([]byte(fmt.Sprintf("saved-%s-%s", accountKey.Address, accountKey.Account))), nil)
	for iter.Next() {
		numTweets++
	}
	iter.Release()
	return numTweets
}
func GetSkippedTweets(accountKey obj.AccountKey, client *twitter.Client, db *leveldb.DB, link bool, date bool) error {
	println("getting skipped tweets")
	//wait 1 second
	time.Sleep(time.Second)
	txList, err := getNewTweets(accountKey, client, db)
	if err != nil {
		return jerr.Get("error getting tweets since the bot was last run", err)
	}
	//get the ID of the newest tweet in txList
	tweetID := int64(0)
	for _, tweetTx := range txList {
		if tweetTx.Tweet.ID > tweetID {
			tweetID = tweetTx.Tweet.ID
		}
	}
	println("saving skipped tweets")
	_, err = Transfer(accountKey, db, link, date)
	if err != nil {
		return jerr.Get("fatal error transferring tweets", err)
	}
	////call transfer until the tweet with the ID of the newest tweet in txList is found, or when we've saved 100 tweets
	//totalSaved := 0
	//for {
	//	if totalSaved >= 20 {
	//		break
	//	}
	//	//check if the tweet with the ID of the newest tweet in txList is in the database
	//	prefix := fmt.Sprintf("saved-%s-%s-%019d", accountKey.Address, accountKey.Account, tweetID)
	//	_,err := db.Get([]byte(prefix), nil)
	//	if err == nil {
	//		break
	//	}
	//	if err != leveldb.ErrNotFound {
	//		return jerr.Get("error getting tweet from database", err)
	//	}
	//	numSaved, err := Transfer(accountKey, db, link, date)
	//	if err != nil {
	//		return jerr.Get("fatal error transferring tweets", err)
	//	}
	//	totalSaved += numSaved
	//}
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
