package tweets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	twitterscraper "github.com/AbdelSallam/twitter-scraper"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/db"
	"github.com/memocash/tweet/tweets/obj"
	"github.com/memocash/tweet/wallet"
	"github.com/syndtr/goleveldb/leveldb"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

const (
	COOKJAR_FILE = "cookies.json"
)

func getNewTweets(accountKey obj.AccountKey, numTweets int, newBot bool, scraper *twitterscraper.Scraper) ([]*db.TweetTx, error) {
	profile, err := GetProfile(accountKey.UserID, scraper)
	if err != nil {
		return nil, jerr.Get("error getting profile", err)
	}
	var userTimelineParams = &twitter.UserTimelineParams{
		UserID:     accountKey.UserID,
		Count:      numTweets,
		ScreenName: profile.Name,
	}
	recentTweetTx, err := db.GetRecentTweetTx(accountKey.UserID)
	if err != nil && !errors.Is(err, leveldb.ErrNotFound) {
		return nil, jerr.Get("error getting recent tweet tx", err)
	}
	if errors.Is(err, leveldb.ErrNotFound) && !newBot {
		log.Printf("no recent tweet tx for user %d\n", accountKey.UserID)
		return nil, nil
	}
	if recentTweetTx != nil {
		userTimelineParams.SinceID = recentTweetTx.TweetId
	}
	_, err = GetAndSaveTwitterTweets(userTimelineParams, scraper)
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

func GetAndSaveTwitterTweets(params *twitter.UserTimelineParams, scraper *twitterscraper.Scraper) ([]obj.TweetTx, error) {
	if params.UserID == 0 {
		return nil, jerr.New("userID is required")
	}
	query := fmt.Sprintf("from:%s", params.ScreenName)
	if params.SinceID != 0 {
		query += fmt.Sprintf(" since_id:%d", params.SinceID)
	}
	if params.MaxID != 0 {
		query += fmt.Sprintf(" max_id:%d", params.MaxID)
	}
	var tweets []twitter.Tweet
	for scrapedTweet := range scraper.SearchTweets(context.Background(), query, params.Count) {
		if scrapedTweet.Error != nil {
			return nil, jerr.Get("error getting tweets", scrapedTweet.Error)
		}
		tweetID, err := strconv.ParseInt(scrapedTweet.ID, 10, 64)
		var inReplyToStatusID int64
		if scrapedTweet.InReplyToStatus != nil {
			inReplyToStatusID, err = strconv.ParseInt(scrapedTweet.InReplyToStatus.ID, 10, 64)
			if err != nil {
				return nil, jerr.Get("error parsing in reply to status id", err)
			}
		} else {
			inReplyToStatusID = 0
		}
		var entities = twitter.Entities{}
		var extendedEntity = twitter.ExtendedEntity{}
		if scrapedTweet.URLs != nil {
			for _, media := range scrapedTweet.URLs {
				entities.Media = append(entities.Media, twitter.MediaEntity{
					MediaURL: media,
				})
				extendedEntity.Media = append(extendedEntity.Media, twitter.MediaEntity{
					MediaURL: media,
				})
			}
		}
		tweet := twitter.Tweet{
			ID:                tweetID,
			InReplyToStatusID: inReplyToStatusID,
			Text:              scrapedTweet.Text,
			CreatedAt:         scrapedTweet.TimeParsed.Format(time.RubyDate),
			User: &twitter.User{
				ID:         params.UserID,
				ScreenName: params.ScreenName,
			},
			Entities:         &entities,
			ExtendedEntities: &extendedEntity,
		}
		tweets = append(tweets, tweet)
	}
	var tweetTxs = make([]obj.TweetTx, len(tweets))
	var dbTweetTxs = make([]db.ObjectI, len(tweets))
	for i := range tweets {
		tweetTxJson, err := json.Marshal(obj.TweetTx{Tweet: &tweets[i], TxHash: nil})
		if err != nil {
			return nil, jerr.Get("error marshaling tweet tx for saving twitter tweets", err)
		}
		dbTweetTxs[i] = &db.TweetTx{
			UserID:  params.UserID,
			TweetId: tweets[i].ID,
			Tx:      tweetTxJson,
		}
	}
	if err := db.Save(dbTweetTxs); err != nil {
		return nil, jerr.Get("error saving db tweet from twitter tweet", err)
	}
	return tweetTxs, nil
}

func GetSkippedTweets(accountKey obj.AccountKey, wlt *wallet.Wallet, scraper *twitterscraper.Scraper, flags db.Flags, numTweets int, newBot bool) error {
	txList, err := getNewTweets(accountKey, numTweets, newBot, scraper)
	if err != nil {
		return jerr.Get("error getting tweets since the bot was last run", err)
	}
	if len(txList) == 0 {
		log.Printf("no new tweets for user %d\n", accountKey.UserID)
		return nil
	}
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

func SaveCookies(cookies []*http.Cookie) error {
	log.Println("saving cookies")
	marshaledCookies, err := json.Marshal(cookies)
	if err != nil {
		return jerr.Get("error marshalling cookies", err)
	}
	err = os.WriteFile(COOKJAR_FILE, marshaledCookies, 0644)
	if err != nil {
		return jerr.Get("error writing cookies", err)
	}
	return nil
}
