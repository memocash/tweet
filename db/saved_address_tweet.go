package db

import (
	"fmt"
	"github.com/jchavannes/jgo/jutil"
	"github.com/syndtr/goleveldb/leveldb/util"
	"strings"
)

type SavedAddressTweet struct {
	Address    string
	ScreenName string
	TweetId    int64
	TxHash     []byte
}

func (t *SavedAddressTweet) GetPrefix() string {
	return PrefixSavedAddressTweet
}

func (t *SavedAddressTweet) GetUid() []byte {
	return []byte(fmt.Sprintf("%s-%s-%019d", t.Address, t.ScreenName, t.TweetId))
}

func (t *SavedAddressTweet) SetUid(b []byte) {
	parts := strings.Split(string(b), "-")
	if len(parts) != 3 {
		return
	}
	t.Address = parts[0]
	t.ScreenName = parts[1]
	t.TweetId = jutil.GetInt64FromString(strings.TrimLeft(parts[2], "0"))
}

func (t *SavedAddressTweet) Serialize() []byte {
	return t.TxHash
}

func (t *SavedAddressTweet) Deserialize(d []byte) {
	t.TxHash = d
}

func GetRecentSavedAddressTweet(address, screenName string) (*SavedAddressTweet, error) {
	var savedAddressTweet = new(SavedAddressTweet)
	if err := GetLastItem(savedAddressTweet, []byte(fmt.Sprintf("%s-%s", address, screenName))); err != nil {
		return nil, fmt.Errorf("error getting recent saved address item; %w", err)
	}
	return savedAddressTweet, nil
}

func GetNumSavedAddressTweet(address, screenName string) (int, error) {
	count, err := GetNum([]byte(fmt.Sprintf("%s-%s-%s-", PrefixSavedAddressTweet, address, screenName)))
	if err != nil {
		return 0, fmt.Errorf("error getting num saved address tweets; %w", err)
	}
	return count, nil
}

func GetSavedAddressTweet(address, screenName string, tweetId int64) (*SavedAddressTweet, error) {
	var savedAddressTweet = &SavedAddressTweet{
		Address:    address,
		ScreenName: screenName,
		TweetId:    tweetId,
	}
	if err := GetSpecificItem(savedAddressTweet); err != nil {
		return nil, fmt.Errorf("error getting saved address tweet; %w", err)
	}
	return savedAddressTweet, nil
}

func GetAllSavedAddressTweet() ([]*SavedAddressTweet, error) {
	db, err := GetDb()
	if err != nil {
		return nil, fmt.Errorf("error getting database handler for get all saved address tweets; %w", err)
	}
	iter := db.NewIterator(util.BytesPrefix([]byte(fmt.Sprintf("%s-", PrefixSavedAddressTweet))), nil)
	defer iter.Release()
	var savedAddressTweets []*SavedAddressTweet
	for iter.Next() {
		var savedAddressTweet = new(SavedAddressTweet)
		Set(savedAddressTweet, iter)
		savedAddressTweets = append(savedAddressTweets, savedAddressTweet)
	}
	return savedAddressTweets, nil
}
