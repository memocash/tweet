package db

import (
	"fmt"
	"github.com/jchavannes/jgo/jutil"
	"github.com/syndtr/goleveldb/leveldb/util"
	"strings"
)

type TweetTx struct {
	ScreenName string
	TweetId    int64
	Tx         []byte
}

func (t *TweetTx) GetPrefix() string {
	return PrefixTweetTx
}

func (t *TweetTx) GetUid() []byte {
	return []byte(fmt.Sprintf("%s-%019d", t.ScreenName, t.TweetId))
}

func (t *TweetTx) SetUid(b []byte) {
	parts := strings.Split(string(b), "-")
	if len(parts) != 2 {
		return
	}
	t.ScreenName = parts[0]
	t.TweetId = jutil.GetInt64FromString(strings.TrimLeft(parts[1], "0"))
}

func (t *TweetTx) Serialize() []byte {
	return t.Tx
}

func (t *TweetTx) Deserialize(d []byte) {
	t.Tx = d
}

func GetTweetTxs(screenName string, startTweetId int64) ([]*TweetTx, error) {
	db, err := GetDb()
	if err != nil {
		return nil, fmt.Errorf("error getting database handler for get tweet txs; %w", err)
	}
	iter := db.NewIterator(util.BytesPrefix([]byte(fmt.Sprintf("%s-%s-", PrefixTweetTx, screenName))), nil)
	defer iter.Release()
	startUid := []byte(fmt.Sprintf("%s-%s-%019d", PrefixTweetTx, screenName, startTweetId))
	var tweetTxs []*TweetTx
	for firstAndOk := iter.Seek(startUid); firstAndOk || iter.Next(); firstAndOk = false {
		var tweetTx = new(TweetTx)
		Set(tweetTx, iter)
		tweetTxs = append(tweetTxs, tweetTx)
		if len(tweetTxs) >= 20 {
			break
		}
	}
	return tweetTxs, nil
}

func GetRecentTweetTx(screenName string) (*TweetTx, error) {
	var tweetTx = new(TweetTx)
	if err := GetLastItem(tweetTx, []byte(screenName)); err != nil {
		return nil, fmt.Errorf("error getting recent tweet tx item; %w", err)
	}
	return tweetTx, nil
}

func GetOldestTweetTx(screenName string) (*TweetTx, error) {
	var tweetTx = new(TweetTx)
	if err := GetFirstItem(tweetTx, []byte(screenName)); err != nil {
		return nil, fmt.Errorf("error getting oldest tweet tx item; %w", err)
	}
	return tweetTx, nil
}

func GetAllTweetTx() ([]*TweetTx, error) {
	db, err := GetDb()
	if err != nil {
		return nil, fmt.Errorf("error getting database handler for get all tweet txs; %w", err)
	}
	iter := db.NewIterator(util.BytesPrefix([]byte(fmt.Sprintf("%s-", PrefixTweetTx))), nil)
	defer iter.Release()
	var tweetTxs []*TweetTx
	for iter.Next() {
		var tweetTx = new(TweetTx)
		Set(tweetTx, iter)
		tweetTxs = append(tweetTxs, tweetTx)
	}
	return tweetTxs, nil
}
