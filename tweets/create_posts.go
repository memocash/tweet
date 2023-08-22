package tweets

import (
	"encoding/json"
	"errors"
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/db"
	"github.com/memocash/tweet/tweets/obj"
	"github.com/memocash/tweet/tweets/save"
	"github.com/memocash/tweet/wallet"
	"github.com/syndtr/goleveldb/leveldb"
	"regexp"
)

func CreateMemoPostsFromDb(accountKey obj.AccountKey, flags db.Flags, wlt wallet.Wallet) (int, error) {
	savedAddressTweet, err := db.GetRecentSavedAddressTweet(accountKey.Address.GetAddr(), accountKey.UserID)
	if err != nil && !errors.Is(err, leveldb.ErrNotFound) {
		jerr.Get("error getting recent saved address tweet", err).Fatal()
	}
	var startID int64 = 0
	if savedAddressTweet != nil {
		startID = savedAddressTweet.TweetId
	}
	tweetTxs, err := db.GetTweetTxs(accountKey.UserID, startID, 20)
	if err != nil {
		return 0, jerr.Get("error getting tweet txs from db", err)
	}
	numTransferred := 0
	for _, tweetTx := range tweetTxs {
		var tweet obj.TweetTx
		if err := json.Unmarshal(tweetTx.Tx, &tweet); err != nil {
			return 0, jerr.Get("error unmarshalling tweetTx for transfer", err)
		}
		match, _ := regexp.MatchString("https://t.co/[a-zA-Z0-9]*$", tweet.Tweet.Text)
		if match {
			//remove the https://t.co from the tweet text
			tweet.Tweet.Text = regexp.MustCompile("https://t.co/[a-zA-Z0-9]*$").ReplaceAllString(tweet.Tweet.Text, "")
		}
		if err := save.Tweet(wlt, accountKey.GetAddress(), tweet.Tweet, flags); err != nil {
			return numTransferred, jerr.Get("error streaming tweets for transfer", err)
		}
		numTransferred++
	}
	return numTransferred, nil
}
