package tweets

import (
	"encoding/json"
	"fmt"
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/database"
	"github.com/memocash/tweet/tweets/obj"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"regexp"
	"strconv"
)

func Transfer(accountKey obj.AccountKey, db *leveldb.DB, appendLink bool, appendDate bool) (int, error) {
	var tweetList []obj.TweetTx
	//find the greatest ID of all the already saved tweets
	prefix := fmt.Sprintf("saved-%s-%s", accountKey.Address, accountKey.Account)
	iter := db.NewIterator(util.BytesPrefix([]byte(prefix)), nil)
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
	prefix = fmt.Sprintf("tweets-%s", accountKey.Account)
	println(prefix)
	iter = db.NewIterator(util.BytesPrefix([]byte(prefix)), nil)
	for iter.Next() {
		key := iter.Key()
		tweetID, _ := strconv.ParseInt(string(key[len(prefix)+1:]), 10, 64)
		println("%d", tweetID)
		if tweetID > startID {
			var tweetTx obj.TweetTx
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
	iter.Release()
	numTransferred := 0
	wlt := database.NewWallet(accountKey.Address, accountKey.Key, db)
	for _, tweet := range tweetList {
		match, _ := regexp.MatchString("https://t.co/[a-zA-Z0-9]*$", tweet.Tweet.Text)
		if match {
			//remove the https://t.co from the tweet text
			tweet.Tweet.Text = regexp.MustCompile("https://t.co/[a-zA-Z0-9]*$").ReplaceAllString(tweet.Tweet.Text, "")
		}
		//marshal the tweet.Tweet object into a json and print it
		if len(tweet.Tweet.Entities.Media) > 0 {
			//append the url to the tweet text on a new line
			for _, media := range tweet.Tweet.ExtendedEntities.Media {
				tweet.Tweet.Text += fmt.Sprintf("\n%s", media.MediaURL)
			}
		}
		println("saving tweet")
		if err := database.SaveTweet(wlt, accountKey, tweet, db, appendLink, appendDate); err != nil {
			return numTransferred, jerr.Get("error streaming tweets for transfer", err)
		}
		numTransferred++
	}
	return numTransferred, nil
}
