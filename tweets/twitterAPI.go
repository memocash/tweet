package tweets

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/jchavannes/jgo/jerr"
	config2 "github.com/memocash/tweet/config"
	"github.com/memocash/tweet/db"
	"github.com/memocash/tweet/tweets/obj"
	"github.com/memocash/tweet/wallet"
	"github.com/syndtr/goleveldb/leveldb"
	util2 "github.com/syndtr/goleveldb/leveldb/util"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"log"
	"os"
)

func GetAllTweets(screenName string, client *twitter.Client) (int, error) {
	var numTweets = 0
	for {
		tweets, err := getOldTweets(screenName, client)
		if err != nil {
			return numTweets, jerr.Get("error getting old tweets", err)
		}
		if len(tweets) == 1 {
			return numTweets, nil
		}
		numTweets += len(tweets)
	}
}

func getOldTweets(screenName string, client *twitter.Client) ([]obj.TweetTx, error) {
	excludeReplies := false
	var userTimelineParams = &twitter.UserTimelineParams{
		ScreenName:     screenName,
		ExcludeReplies: &excludeReplies,
		Count:          100,
	}
	recentTweetTx, err := db.GetOldestTweetTx(screenName)
	if err != nil && !errors.Is(err, leveldb.ErrNotFound) {
		return nil, jerr.Get("error getting oldest tweet tx", err)
	}
	if recentTweetTx != nil {
		userTimelineParams.MaxID = recentTweetTx.TweetId
	}
	tweetTxs, err := GetAndSaveTwitterTweets(client, userTimelineParams)
	if err != nil {
		return nil, jerr.Get("error getting new tweets from twitter", err)
	}
	return tweetTxs, nil
}

func getNewTweets(accountKey obj.AccountKey, client *twitter.Client, numTweets int) ([]obj.TweetTx, error) {
	excludeReplies := false
	var userTimelineParams = &twitter.UserTimelineParams{
		ScreenName:     accountKey.Account,
		ExcludeReplies: &excludeReplies,
		Count:          numTweets,
	}
	recentTweetTx, err := db.GetRecentTweetTx(accountKey.Account)
	if err != nil && !errors.Is(err, leveldb.ErrNotFound) {
		return nil, jerr.Get("error getting recent tweet tx", err)
	}
	if recentTweetTx != nil {
		userTimelineParams.SinceID = recentTweetTx.TweetId
	}
	tweetTxs, err := GetAndSaveTwitterTweets(client, userTimelineParams)
	if err != nil {
		return nil, jerr.Get("error getting new tweets from twitter", err)
	}
	return tweetTxs, nil
}

func GetAndSaveTwitterTweets(client *twitter.Client, params *twitter.UserTimelineParams) ([]obj.TweetTx, error) {
	if params.ScreenName == "" {
		return nil, jerr.New("screen name is required")
	}
	tweets, _, err := client.Timelines.UserTimeline(params)
	if err != nil {
		return nil, jerr.Get("error getting old tweets from user timeline", err)
	}
	var tweetTxs = make([]obj.TweetTx, len(tweets))
	var dbTweetTxs = make([]db.ObjectI, len(tweets))
	for i := range tweets {
		tweetTxJson, err := json.Marshal(obj.TweetTx{Tweet: &tweets[i], TxHash: nil})
		if err != nil {
			return nil, jerr.Get("error marshaling tweet tx for saving twitter tweets", err)
		}
		dbTweetTxs[i] = &db.TweetTx{
			ScreenName: params.ScreenName,
			TweetId:    tweets[i].ID,
			Tx:         tweetTxJson,
		}
		tweetTxs[i] = obj.TweetTx{Tweet: &tweets[i]}
	}
	if err := db.Save(dbTweetTxs); err != nil {
		return nil, jerr.Get("error saving db tweet from twitter tweet", err)
	}
	return tweetTxs, nil
}

func getNewTweetsLocal(accountKey obj.AccountKey, numTweets int) ([]obj.TweetTx, error) {
	file := fmt.Sprintf("tweets-%s.json", accountKey.Account)
	f, err := os.Open(file)
	if err != nil {
		return nil, jerr.Get("error opening file for local storage of tweets", err)
	}
	defer f.Close()
	var tweetTxs []obj.TweetTx
	if err := json.NewDecoder(f).Decode(&tweetTxs); err != nil {
		return nil, jerr.Get("error decoding tweets for local storage", err)
	}
	var dbTweetTxs = make([]db.ObjectI, len(tweetTxs))
	for i, tweetTx := range tweetTxs {
		if i >= numTweets {
			break
		}
		tweetTxJson, err := json.Marshal(tweetTx)
		if err != nil {
			return nil, jerr.Get("error marshaling tweet tx for saving twitter tweets local", err)
		}
		dbTweetTxs[i] = &db.TweetTx{
			ScreenName: accountKey.Account,
			TweetId:    tweetTxs[i].Tweet.ID,
			Tx:         tweetTxJson,
		}
	}
	if err := db.Save(dbTweetTxs); err != nil {
		return nil, jerr.Get("error saving db tweet from twitter tweet local", err)
	}
	return tweetTxs, nil
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

func GetSkippedTweets(accountKey obj.AccountKey, wlt *wallet.Wallet, client *twitter.Client, db *leveldb.DB, link bool, date bool, numTweets int) error {
	println("getting skipped tweets")
	txList, err := getNewTweets(accountKey, client, numTweets)
	//txList, err := getNewTweetsLocal(accountKey, db, numTweets)
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
	totalSaved := 0
	for {
		if totalSaved >= numTweets {
			break
		}
		//check if the tweet with the ID of the newest tweet in txList is in the database
		prefix := fmt.Sprintf("saved-%s-%s-%019d", accountKey.Address, accountKey.Account, tweetID)
		_, err := db.Get([]byte(prefix), nil)
		if err == nil {
			break
		}
		if err != leveldb.ErrNotFound {
			return jerr.Get("error getting tweet from database", err)
		}
		numSaved, err := Transfer(accountKey, db, link, date, *wlt)
		if err != nil {
			return jerr.Get("fatal error transferring tweets", err)
		}
		if numSaved == 0 {
			break
		}
		totalSaved += numSaved
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
