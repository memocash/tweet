package tweets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/jchavannes/jgo/jerr"
	config2 "github.com/memocash/tweet/config"
	"github.com/memocash/tweet/db"
	"github.com/memocash/tweet/tweets/obj"
	"github.com/memocash/tweet/wallet"
	"github.com/michimani/gotwi"
	"github.com/michimani/gotwi/resources"
	"github.com/michimani/gotwi/tweet/timeline"
	"github.com/michimani/gotwi/tweet/timeline/types"
	"github.com/syndtr/goleveldb/leveldb"
	"log"
	"os"
	"strconv"
	"time"
)

func GetAllTweets(userId int64, client *gotwi.Client) (int, error) {
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

func getOldTweets(userId int64, client *gotwi.Client) ([]obj.TweetTx, error) {
	var inputParams = &types.ListTweetsInput{
		ID:         strconv.FormatInt(userId, 10),
		MaxResults: types.ListMaxResults(100),
	}
	recentTweetTx, err := db.GetOldestTweetTx(userId)
	if err != nil && !errors.Is(err, leveldb.ErrNotFound) {
		return nil, jerr.Get("error getting oldest tweet tx", err)
	}
	if recentTweetTx != nil {
		inputParams.UntilID = strconv.FormatInt(recentTweetTx.TweetId, 10)
	}
	tweetTxs, err := GetAndSaveTwitterTweets(client, inputParams)
	if err != nil {
		return nil, jerr.Get("error getting new tweets from twitter", err)
	}
	return tweetTxs, nil
}

func getNewTweets(accountKey obj.AccountKey, client *gotwi.Client, numTweets int, newBot bool) ([]*db.TweetTx, error) {
	var inputParams = &types.ListTweetsInput{
		ID:         strconv.FormatInt(accountKey.UserID, 10),
		MaxResults: types.ListMaxResults(numTweets),
	}
	recentTweetTx, err := db.GetRecentTweetTx(accountKey.UserID)
	if err != nil && !errors.Is(err, leveldb.ErrNotFound) {
		return nil, jerr.Get("error getting recent tweet tx", err)
	}
	if errors.Is(err, leveldb.ErrNotFound) && !newBot {
		return nil, nil
	}
	if recentTweetTx != nil {
		inputParams.SinceID = strconv.FormatInt(recentTweetTx.TweetId, 10)
	}
	_, err = GetAndSaveTwitterTweets(client, inputParams)
	if err != nil {
		return nil, jerr.Get("error getting new tweets from twitter", err)
	}
	recentSavedTweetTx, err := db.GetRecentSavedAddressTweet(accountKey.Address.GetAddr(), accountKey.UserID)
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
	if err != nil {
		return nil, jerr.Get("error getting tweet txs", err)
	}
	return tweetTxs, nil
}

func GetAndSaveTwitterTweets(client *gotwi.Client, params *types.ListTweetsInput) ([]obj.TweetTx, error) {
	if params.ID == "" {
		return nil, jerr.New("userID is required")
	}
	tweets, err := timeline.ListTweets(context.Background(), client, params)
	if err != nil {
		return nil, jerr.Get("error getting old tweets from user timeline", err)
	}
	var tweetTxs = make([]obj.TweetTx, len(tweets.Data))
	var dbTweetTxs = make([]db.ObjectI, len(tweets.Data))
	for i := range tweets.Data {
		convertedTweet := convertToV1Tweet(tweets.Data[i])
		tweetTxJson, err := json.Marshal(obj.TweetTx{Tweet: convertedTweet, TxHash: nil})
		if err != nil {
			return nil, jerr.Get("error marshaling tweet tx for saving twitter tweets", err)
		}
		dbTweetTxs[i] = &db.TweetTx{
			UserID:  convertedTweet.User.ID,
			TweetId: convertedTweet.ID,
			Tx:      tweetTxJson,
		}
	}
	if err := db.Save(dbTweetTxs); err != nil {
		return nil, jerr.Get("error saving db tweet from twitter tweet", err)
	}
	return tweetTxs, nil
}
func convertToV1Tweet(tweet resources.Tweet) *twitter.Tweet {
	tweetID, err := strconv.ParseInt(*tweet.ID, 10, 64)
	if err != nil {
		log.Fatalf("error converting tweet id to int: %s", err)
	}
	userID, err := strconv.ParseInt(*tweet.AuthorID, 10, 64)
	if err != nil {
		log.Fatalf("error converting user id to int: %s", err)
	}
	inReplyToStatusID, err := strconv.ParseInt(*tweet.ConversationID, 10, 64)
	if err != nil {
		log.Fatalf("error converting in reply to status id to int: %s", err)
	}
	var entities *twitter.Entities
	var extendedEntity *twitter.ExtendedEntity
	if tweet.Entities != nil {
		//for each url in tweet.Entities.Urls, convert to twitter.MediaEntity
		for _, url := range tweet.Entities.URLs {
			mediaEntity := twitter.MediaEntity{
				//might have to use different URL
				MediaURL: *url.URL,
			}
			entities.Media = append(entities.Media, mediaEntity)
			extendedEntity.Media = append(extendedEntity.Media, mediaEntity)
		}
	}
	v1Tweet := &twitter.Tweet{
		ID:                tweetID,
		InReplyToStatusID: inReplyToStatusID,
		Text:              *tweet.Text,
		CreatedAt:         tweet.CreatedAt.Format(time.RubyDate),
		User: &twitter.User{
			ID: userID,
		},
		Entities:         entities,
		ExtendedEntities: extendedEntity,
	}
	return v1Tweet
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
			UserID:  accountKey.UserID,
			TweetId: tweetTxs[i].Tweet.ID,
			Tx:      tweetTxJson,
		}
	}
	if err := db.Save(dbTweetTxs); err != nil {
		return nil, jerr.Get("error saving db tweet from twitter tweet local", err)
	}
	return tweetTxs, nil
}

func GetSkippedTweets(accountKey obj.AccountKey, wlt *wallet.Wallet, client *gotwi.Client, flags db.Flags, numTweets int, newBot bool) error {
	txList, err := getNewTweets(accountKey, client, numTweets, newBot)
	//txList, err := getNewTweetsLocal(accountKey, db, numTweets)
	if err != nil {
		return jerr.Get("error getting tweets since the bot was last run", err)
	}
	if len(txList) == 0 {
		return nil
	}
	//get the ID of the newest tweet in txList
	tweetID := int64(0)
	for _, tweetTx := range txList {
		if tweetTx.TweetId > tweetID {
			tweetID = tweetTx.TweetId
		}
	}
	for {
		savedAddressTweet, err := db.GetSavedAddressTweet(accountKey.Address.GetAddr(), accountKey.UserID, tweetID)
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
	}
	return nil
}
func Connect() *gotwi.Client {
	conf := config2.GetTwitterAPIConfig()
	if !conf.IsSet() {
		log.Fatal("Application Access Token required")
	}
	client, err := gotwi.NewClient(&gotwi.NewClientInput{
		AuthenticationMethod: gotwi.AuthenMethodOAuth2BearerToken,
		OAuthToken:           conf.ConsumerKey,
		OAuthTokenSecret:     conf.ConsumerSecret,
	})
	if err != nil {
		jerr.Get("error creating twitter client", err).Fatal()
	}
	return client
}
