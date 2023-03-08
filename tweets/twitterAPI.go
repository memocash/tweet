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
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"log"
	"os"
	"strconv"
)

func GetAllTweets(userId int64, client *twitter.Client) (int, error) {
	var numTweets = 0
	for {
		tweets, err := getOldTweets(userId, client)
		if err != nil {
			return numTweets, jerr.Get("error getting old tweets", err)
		}
		if len(tweets) == 1 {
			return numTweets, nil
		}
		numTweets += len(tweets)
	}
}

func getOldTweets(userId int64, client *twitter.Client) ([]obj.TweetTx, error) {
	excludeReplies := false
	var userTimelineParams = &twitter.UserTimelineParams{
		UserID:         userId,
		ExcludeReplies: &excludeReplies,
		Count:          100,
	}
	recentTweetTx, err := db.GetOldestTweetTx(userId)
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

func getNewTweets(accountKey obj.AccountKey, client *twitter.Client, numTweets int, newBot bool) ([]*db.TweetTx, error) {
	excludeReplies := false
	println("getting new tweets", accountKey.UserID)
	var userTimelineParams = &twitter.UserTimelineParams{
		UserID:         accountKey.UserID,
		ExcludeReplies: &excludeReplies,
		Count:          numTweets,
	}
	recentTweetTx, err := db.GetRecentTweetTx(accountKey.UserID)
	if err != nil && !errors.Is(err, leveldb.ErrNotFound) {
		return nil, jerr.Get("error getting recent tweet tx", err)
	}
	if errors.Is(err, leveldb.ErrNotFound) && !newBot {
		return nil, nil
	}
	if recentTweetTx != nil {
		userTimelineParams.SinceID = recentTweetTx.TweetId
		println("recent tweet tx", recentTweetTx.TweetId)
	}
	_, err = GetAndSaveTwitterTweets(client, userTimelineParams)
	if err != nil {
		return nil, jerr.Get("error getting new tweets from twitter", err)
	}
	recentSavedTweetTx, err := db.GetRecentSavedAddressTweet(accountKey.Address.GetEncoded(), accountKey.UserID)
	if err != nil && !errors.Is(err, leveldb.ErrNotFound) {
		return nil, jerr.Get("error getting recent saved address tweet", err)
	}
	var recentTweetId int64
	if recentSavedTweetTx == nil {
		recentTweetId = 0
	} else {
		recentTweetId = recentSavedTweetTx.TweetId
	}
	tweetTxs, err := db.GetTweetTxs(accountKey.UserID, recentTweetId, numTweets)
	return tweetTxs, nil
}

func GetAndSaveTwitterTweets(client *twitter.Client, params *twitter.UserTimelineParams) ([]obj.TweetTx, error) {
	if params.UserID == 0 {
		return nil, jerr.New("userID is required")
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
			UserID:  strconv.FormatInt(params.UserID, 10),
			TweetId: tweets[i].ID,
			Tx:      tweetTxJson,
		}
	}
	if err := db.Save(dbTweetTxs); err != nil {
		return nil, jerr.Get("error saving db tweet from twitter tweet", err)
	}
	return tweetTxs, nil
}

func getNewTweetsLocal(accountKey obj.AccountKey, numTweets int) ([]obj.TweetTx, error) {
	file := fmt.Sprintf("tweets-%s.json", strconv.FormatInt(accountKey.UserID, 10))
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
			UserID:  strconv.FormatInt(accountKey.UserID, 10),
			TweetId: tweetTxs[i].Tweet.ID,
			Tx:      tweetTxJson,
		}
	}
	if err := db.Save(dbTweetTxs); err != nil {
		return nil, jerr.Get("error saving db tweet from twitter tweet local", err)
	}
	return tweetTxs, nil
}

func GetSkippedTweets(accountKey obj.AccountKey, wlt *wallet.Wallet, client *twitter.Client, flags db.Flags, numTweets int, newBot bool) error {
	txList, err := getNewTweets(accountKey, client, numTweets, newBot)
	//txList, err := getNewTweetsLocal(accountKey, db, numTweets)
	if err != nil {
		return jerr.Get("error getting tweets since the bot was last run", err)
	}
	if len(txList) == 0 {
		println("no new tweets")
		return nil
	}
	//get the ID of the newest tweet in txList
	tweetID := int64(0)
	for _, tweetTx := range txList {
		if tweetTx.TweetId > tweetID {
			tweetID = tweetTx.TweetId
		}
	}
	totalSaved := 0
	for {
		if totalSaved >= numTweets {
			break
		}
		savedAddressTweet, err := db.GetSavedAddressTweet(accountKey.Address.GetEncoded(), accountKey.UserID, tweetID)
		if err != nil && !errors.Is(err, leveldb.ErrNotFound) {
			return jerr.Get("error getting saved address tweet for get skipped", err)
		}
		if savedAddressTweet != nil {
			break
		}
		numSaved, err := CreateMemoPostsFromDb(accountKey, flags, *wlt)
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
