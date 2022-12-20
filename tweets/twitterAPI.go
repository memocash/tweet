package tweets

import (
	"encoding/json"
	"fmt"
	"github.com/dghubble/go-twitter/twitter"
	config2 "github.com/memocash/tweet/config"
	"github.com/syndtr/goleveldb/leveldb"
	util2 "github.com/syndtr/goleveldb/leveldb/util"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"io/ioutil"
	"log"
	"strconv"
)

func GetAllTweets(screenName string, client *twitter.Client, db *leveldb.DB) {
	for {
		tweets := getOldTweets(screenName, client, db)
		if len(tweets) == 1 {
			break
		}
	}
}

func GetNewTweets(screenName string, client *twitter.Client, fileHeader string) []TweetTx {
	IdInfo := IdInfo{
		ArchivedID: 0,
		NewestID:   0,
	}
	fileName := fmt.Sprintf("%s_IdInfo.json", fileHeader)
	content, err := ioutil.ReadFile(fileName)
	if err == nil {
		err = json.Unmarshal(content, &IdInfo)
	}
	var userTimelineParams *twitter.UserTimelineParams
	excludeReplies := false
	if IdInfo.NewestID != 0 {
		userTimelineParams = &twitter.UserTimelineParams{ScreenName: screenName, ExcludeReplies: &excludeReplies, SinceID: IdInfo.NewestID, Count: 20}
	}
	if IdInfo.NewestID == 0 {
		userTimelineParams = &twitter.UserTimelineParams{ScreenName: screenName, ExcludeReplies: &excludeReplies, Count: 20}
	}
	tweets, _, _ := client.Timelines.UserTimeline(userTimelineParams)
	var tweetTxs []TweetTx
	for i, tweet := range tweets {
		tweetTxs = append(tweetTxs, TweetTx{Tweet: &tweets[i], TxHash: nil})
		println(tweet.Text)
		println(tweet.CreatedAt)
		if tweet.ID > IdInfo.NewestID || IdInfo.NewestID == 0 {
			IdInfo.NewestID = tweet.ID
		}
	}
	file, _ := json.MarshalIndent(IdInfo, "", " ")
	_ = ioutil.WriteFile(fileName, file, 0644)
	return tweetTxs
}
func getOldTweets(screenName string, client *twitter.Client, db *leveldb.DB) []TweetTx {
	var userTimelineParams *twitter.UserTimelineParams
	excludeReplies := false
	//check if there are any tweetTx objects with the prefix containing this address and this screenName
	prefix := fmt.Sprintf("tweets-%s", screenName)
	iter := db.NewIterator(util2.BytesPrefix([]byte(prefix)), nil)
	tweetsFound := iter.Next()
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
	tweets, _, _ := client.Timelines.UserTimeline(userTimelineParams)
	var tweetTxs []TweetTx
	for i, tweet := range tweets {
		prefix := fmt.Sprintf("tweets-%s-%019d", screenName, tweet.ID)
		tweetTx, _ := json.Marshal(TweetTx{Tweet: &tweets[i], TxHash: nil})
		db.Put([]byte(prefix), tweetTx, nil)
		//println(tweet.Text)
		//println(tweet.CreatedAt)
		tweetTxs = append(tweetTxs, TweetTx{Tweet: &tweets[i], TxHash: nil})
	}
	return tweetTxs
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
