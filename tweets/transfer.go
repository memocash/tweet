package tweets

import (
	"encoding/json"
	"fmt"
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/db"
	"github.com/memocash/tweet/tweets/obj"
	"github.com/memocash/tweet/tweets/save"
	"github.com/memocash/tweet/wallet"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"regexp"
	"strconv"
)

func Transfer(accountKey obj.AccountKey, levelDb *leveldb.DB, appendLink bool, appendDate bool, wlt wallet.Wallet) (int, error) {
	var tweetList []obj.TweetTx
	//find the greatest ID of all the already saved tweets
	prefix := fmt.Sprintf("saved-%s-%s", accountKey.Address, accountKey.Account)
	iter := levelDb.NewIterator(util.BytesPrefix([]byte(prefix)), nil)
	var startID int64 = 0
	for iter.Next() {
		key := iter.Key()
		tweetID, _ := strconv.ParseInt(string(key[len(prefix)+1:]), 10, 64)
		if tweetID > startID || startID == 0 {
			startID = tweetID
		}
	}
	iter.Release()
	//get up to 20 tweets from the tweets-twittername-tweetID prefix with the smallest IDs greater than the startID
	tweetTxs, err := db.GetTweetTxs(accountKey.Account, startID)
	if err != nil {
		return 0, jerr.Get("error getting tweet txs from db", err)
	}
	for _, dbTweetTx := range tweetTxs {
		var tweetTx obj.TweetTx
		if err := json.Unmarshal(dbTweetTx.Tx, &tweetTx); err != nil {
			return 0, jerr.Get("error unmarshalling tweetTx", err)
		}
		tweetList = append(tweetList, tweetTx)
	}
	numTransferred := 0
	for _, tweet := range tweetList {
		match, _ := regexp.MatchString("https://t.co/[a-zA-Z0-9]*$", tweet.Tweet.Text)
		if match {
			//remove the https://t.co from the tweet text
			tweet.Tweet.Text = regexp.MustCompile("https://t.co/[a-zA-Z0-9]*$").ReplaceAllString(tweet.Tweet.Text, "")
		}
		if tweet.Tweet.Entities != nil && tweet.Tweet.Entities.Media != nil && len(tweet.Tweet.Entities.Media) > 0 {
			//append the url to the tweet text on a new line
			for _, media := range tweet.Tweet.ExtendedEntities.Media {
				tweet.Tweet.Text += fmt.Sprintf("\n%s", media.MediaURL)
			}
		}
		if err := save.Tweet(wlt, accountKey, tweet, levelDb, appendLink, appendDate); err != nil {
			return numTransferred, jerr.Get("error streaming tweets for transfer", err)
		}
		numTransferred++
	}
	return numTransferred, nil
}
