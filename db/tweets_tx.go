package db

import (
	"fmt"
	"github.com/jchavannes/jgo/jutil"
	"github.com/syndtr/goleveldb/leveldb/util"
	"strconv"
)

type TweetTx struct {
	UserID  int64
	TweetId int64
	Tx      []byte
}

func (t *TweetTx) GetPrefix() string {
	return PrefixTweetTx
}

func (t *TweetTx) GetUid() []byte {
	return []byte(fmt.Sprintf("%d-%019d", t.UserID, t.TweetId))
}

func (t *TweetTx) SetUid(b []byte) {
	if len(b) != 16 {
		fmt.Printf("\n\n\ninvalid uid for tweet tx: %s\n\n\n", string(b))
		return
	}
	t.UserID = jutil.GetInt64Big(b[:8])
	t.TweetId = jutil.GetInt64Big(b[8:])
}

func (t *TweetTx) Serialize() []byte {
	return t.Tx
}

func (t *TweetTx) Deserialize(d []byte) {
	t.Tx = d
}

func GetTweetTxs(userId int64, startTweetId int64, max int) ([]*TweetTx, error) {
	db, err := GetDb()
	if err != nil {
		return nil, fmt.Errorf("error getting database handler for get tweet txs; %w", err)
	}
	iter := db.NewIterator(util.BytesPrefix([]byte(fmt.Sprintf("%s-%s-", PrefixTweetTx, strconv.FormatInt(userId, 10)))), nil)
	defer iter.Release()
	startUid := []byte(fmt.Sprintf("%s-%s-%019d", PrefixTweetTx, strconv.FormatInt(userId, 10), startTweetId))
	var tweetTxs []*TweetTx
	for firstAndOk := iter.Seek(startUid); firstAndOk || iter.Next(); firstAndOk = false {
		var tweetTx = new(TweetTx)
		Set(tweetTx, iter)
		tweetTxs = append(tweetTxs, tweetTx)
		if max > 0 && len(tweetTxs) >= max {
			break
		}
	}
	if err := iter.Error(); err != nil {
		return nil, fmt.Errorf("error iterating over get tweet txs; %w", err)
	}
	return tweetTxs, nil
}

func GetRecentTweetTx(userId int64) (*TweetTx, error) {
	var tweetTx = new(TweetTx)
	if err := GetLastItem(tweetTx, []byte(strconv.FormatInt(userId, 10))); err != nil {
		return nil, fmt.Errorf("error getting recent tweet tx item; %w", err)
	}
	return tweetTx, nil
}

func GetOldestTweetTx(userId int64) (*TweetTx, error) {
	var tweetTx = new(TweetTx)
	if err := GetFirstItem(tweetTx, []byte(strconv.FormatInt(userId, 10))); err != nil {
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
	if err := iter.Error(); err != nil {
		return nil, fmt.Errorf("error iterating over all tweet txs; %w", err)
	}
	return tweetTxs, nil
}
