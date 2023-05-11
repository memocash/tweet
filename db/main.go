package db

import (
	"fmt"
	"github.com/jchavannes/jgo/jutil"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/util"
)

const (
	PrefixAddressKey        = "linked"
	PrefixAddressSeenTx     = "address-seen-tx"
	PrefixAddressTime       = "addresstime"
	PrefixBlock             = "block"
	PrefixBotRunningCount   = "memobot-running-count"
	PrefixBotStreamsCount   = "memobot-num-streams"
	PrefixCompletedTx       = "completed"
	PrefixFlag              = "flags"
	PrefixProfile           = "profile"
	PrefixSavedAddressTweet = "saved"
	PrefixTweetTx           = "tweets"
	PrefixTxBlock           = "txblock"
	PrefixTxInput           = "input"
	PrefixTxOutput          = "output"

	PrefixSubBotCommand = "sub-bot-command"
)

var _db *leveldb.DB

func GetDb() (*leveldb.DB, error) {
	if _db != nil {
		return _db, nil
	}
	db, err := leveldb.OpenFile("tweets.db", nil)
	if err != nil {
		return nil, fmt.Errorf("error opening db; %w", err)
	}
	_db = db
	return db, nil
}

type ObjectI interface {
	GetPrefix() string
	GetUid() []byte
	SetUid([]byte)
	Serialize() []byte
	Deserialize([]byte)
}

const Spacer = '-'

func GetObjectCombinedUid(o ObjectI) []byte {
	uid := o.GetUid()
	if len(uid) == 0 {
		return []byte(o.GetPrefix())
	}
	return jutil.CombineBytes([]byte(o.GetPrefix()), []byte{Spacer}, uid)
}

func Save(objects []ObjectI) error {
	db, err := GetDb()
	if err != nil {
		return fmt.Errorf("error getting database handler for save; %w", err)
	}
	batch := new(leveldb.Batch)
	for _, object := range objects {
		batch.Put(GetObjectCombinedUid(object), object.Serialize())
	}
	if err := db.Write(batch, nil); err != nil {
		return fmt.Errorf("error saving leveldb objects; %w", err)
	}
	return nil
}

func Delete(objects []ObjectI) error {
	db, err := GetDb()
	if err != nil {
		return fmt.Errorf("error getting database handler for delete; %w", err)
	}
	batch := new(leveldb.Batch)
	for _, object := range objects {
		batch.Delete(GetObjectCombinedUid(object))
	}
	if err := db.Write(batch, nil); err != nil {
		return fmt.Errorf("error deleting leveldb objects; %w", err)
	}
	return nil
}

func GetItem(obj ObjectI) error {
	db, err := GetDb()
	if err != nil {
		return fmt.Errorf("error getting database handler for save; %w", err)
	}
	val, err := db.Get(GetObjectCombinedUid(obj), nil)
	if err != nil {
		return fmt.Errorf("error getting db item single; %w", err)
	}
	obj.Deserialize(val)
	return nil
}

func GetSpecificItem(obj ObjectI) error {
	db, err := GetDb()
	if err != nil {
		return fmt.Errorf("error getting database handler for specific item; %w", err)
	}
	fullUid := GetObjectCombinedUid(obj)
	val, err := db.Get(fullUid, nil)
	if err != nil {
		return fmt.Errorf("error getting db item specific; %w", err)
	}
	obj.Deserialize(val)
	return nil
}

func GetFirstItem(obj ObjectI, prefix []byte) error {
	db, err := GetDb()
	if err != nil {
		return fmt.Errorf("error getting database handler for save; %w", err)
	}
	topicPrefix := jutil.CombineBytes([]byte(obj.GetPrefix()), []byte{Spacer})
	rng := util.BytesPrefix(jutil.CombineBytes(topicPrefix, prefix))
	iter := db.NewIterator(rng, nil)
	if !iter.First() {
		return leveldb.ErrNotFound
	}
	if err := iter.Error(); err != nil {
		return fmt.Errorf("error iterating first item; %w", err)
	}
	Set(obj, iter)
	return nil
}

func GetLastItem(obj ObjectI, prefix []byte) error {
	db, err := GetDb()
	if err != nil {
		return fmt.Errorf("error getting database handler for save; %w", err)
	}
	topicPrefix := jutil.CombineBytes([]byte(obj.GetPrefix()), []byte{Spacer})
	rng := util.BytesPrefix(jutil.CombineBytes(topicPrefix, prefix))
	iter := db.NewIterator(rng, nil)
	if !iter.Last() {
		return leveldb.ErrNotFound
	}
	if err := iter.Error(); err != nil {
		return fmt.Errorf("error iterating last item; %w", err)
	}
	Set(obj, iter)
	return nil
}

func GetNum(prefix []byte) (int, error) {
	db, err := GetDb()
	if err != nil {
		return 0, fmt.Errorf("error getting database handler for get num db objects; %w", err)
	}
	iter := db.NewIterator(util.BytesPrefix(prefix), nil)
	defer iter.Release()
	var count int
	for iter.Next() {
		count++
	}
	if err := iter.Error(); err != nil {
		return 0, fmt.Errorf("error iterating count items; %w", err)
	}
	return count, nil
}

func Set(obj ObjectI, iter iterator.Iterator) {
	key, val := GetKeyVal(iter)
	obj.SetUid(key[len(obj.GetPrefix())+1:])
	obj.Deserialize(val)
}

func GetKeyVal(iter iterator.Iterator) ([]byte, []byte) {
	var key = make([]byte, len(iter.Key()))
	copy(key, iter.Key())
	var val = make([]byte, len(iter.Value()))
	copy(val, iter.Value())
	return key, val
}
