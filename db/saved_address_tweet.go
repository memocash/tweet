package db

import (
	"fmt"
	"github.com/jchavannes/jgo/jutil"
	"github.com/syndtr/goleveldb/leveldb/util"
	"strconv"
)

type SavedAddressTweet struct {
	Address [25]byte
	UserID  int64
	TweetId int64
	TxHash  []byte
}

func (t *SavedAddressTweet) GetPrefix() string {
	return PrefixSavedAddressTweet
}

func (t *SavedAddressTweet) GetUid() []byte {
	return []byte(fmt.Sprintf("%s-%d-%019d", t.Address, t.UserID, t.TweetId))
}

func (t *SavedAddressTweet) SetUid(b []byte) {
	if len(b) != 41 {
		println("\n\n\ninvalid uid for saved address tweet\n\n\n")
		return
	}
	copy(t.Address[:], b[:25])
	t.UserID = jutil.GetInt64Big(b[25:33])
	t.TweetId = jutil.GetInt64Big(b[33:])

}

func (t *SavedAddressTweet) Serialize() []byte {
	return t.TxHash
}

func (t *SavedAddressTweet) Deserialize(d []byte) {
	t.TxHash = d
}

func GetRecentSavedAddressTweet(address string, userId int64) (*SavedAddressTweet, error) {
	var savedAddressTweet = new(SavedAddressTweet)
	if err := GetLastItem(savedAddressTweet, []byte(fmt.Sprintf("%s-%s", address, strconv.FormatInt(userId, 10)))); err != nil {
		return nil, fmt.Errorf("error getting recent saved address item; %w", err)
	}
	return savedAddressTweet, nil
}

func GetNumSavedAddressTweet(address string, userId int64) (int, error) {
	count, err := GetNum([]byte(fmt.Sprintf("%s-%s-%s-", PrefixSavedAddressTweet, address, strconv.FormatInt(userId, 10))))
	if err != nil {
		return 0, fmt.Errorf("error getting num saved address tweets; %w", err)
	}
	return count, nil
}

func GetSavedAddressTweet(address [25]byte, userId int64, tweetId int64) (*SavedAddressTweet, error) {
	var savedAddressTweet = &SavedAddressTweet{
		Address: address,
		UserID:  userId,
		TweetId: tweetId,
	}
	if err := GetSpecificItem(savedAddressTweet); err != nil {
		return nil, fmt.Errorf("error getting saved address tweet; %w", err)
	}
	return savedAddressTweet, nil
}

func GetAllSavedAddressTweet(prefix []byte) ([]*SavedAddressTweet, error) {
	db, err := GetDb()
	if err != nil {
		return nil, fmt.Errorf("error getting database handler for get all saved address tweets; %w", err)
	}
	iterPrefix := []byte(fmt.Sprintf("%s-", PrefixSavedAddressTweet))
	if len(prefix) > 0 {
		iterPrefix = append(iterPrefix, prefix...)
	}
	iter := db.NewIterator(util.BytesPrefix(iterPrefix), nil)
	defer iter.Release()
	var savedAddressTweets []*SavedAddressTweet
	for iter.Next() {
		var savedAddressTweet = new(SavedAddressTweet)
		Set(savedAddressTweet, iter)
		savedAddressTweets = append(savedAddressTweets, savedAddressTweet)
	}
	if err := iter.Error(); err != nil {
		return nil, fmt.Errorf("error iterating over all saved address tweets; %w", err)
	}
	return savedAddressTweets, nil
}
