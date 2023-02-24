package db

import (
	"fmt"
	"github.com/jchavannes/jgo/jutil"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/util"
)

const (
	PrefixAddressSeenTx = "address-seen-tx"
	PrefixCompletedTx   = "completed"
	PrefixTweetTx       = "tweets"
)

var _db *leveldb.DB

func GetDb() (*leveldb.DB, error) {
	if _db != nil {
		return _db, nil
	}
	db, err := leveldb.OpenFile("tweets.db", nil)
	if err != nil {
		return nil, fmt.Errorf("%w; error opening db", err)
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
	return jutil.CombineBytes([]byte(o.GetPrefix()), []byte{Spacer}, o.GetUid())
}

func Save(objects []ObjectI) error {
	db, err := GetDb()
	if err != nil {
		return fmt.Errorf("%w; error getting database handler for save", err)
	}
	batch := new(leveldb.Batch)
	for _, object := range objects {
		batch.Put(GetObjectCombinedUid(object), object.Serialize())
	}
	if err := db.Write(batch, nil); err != nil {
		return fmt.Errorf("%w; error saving leveldb objects", err)
	}
	return nil
}

func Delete(objects []ObjectI) error {
	db, err := GetDb()
	if err != nil {
		return fmt.Errorf("%w; error getting database handler for delete", err)
	}
	batch := new(leveldb.Batch)
	for _, object := range objects {
		batch.Delete(GetObjectCombinedUid(object))
	}
	if err := db.Write(batch, nil); err != nil {
		return fmt.Errorf("%w; error deleting leveldb objects", err)
	}
	return nil
}

func GetItem(obj ObjectI) error {
	db, err := GetDb()
	if err != nil {
		return fmt.Errorf("%w; error getting database handler for save", err)
	}
	val, err := db.Get(GetObjectCombinedUid(obj), nil)
	if err != nil {
		return fmt.Errorf("%w; error getting db item single", err)
	}
	obj.Deserialize(val)
	return nil
}

func GetFirstItem(obj ObjectI, prefix []byte) error {
	db, err := GetDb()
	if err != nil {
		return fmt.Errorf("%w; error getting database handler for save", err)
	}
	topicPrefix := jutil.CombineBytes([]byte(obj.GetPrefix()), []byte{Spacer})
	rng := util.BytesPrefix(jutil.CombineBytes(topicPrefix, prefix))
	iter := db.NewIterator(rng, nil)
	if !iter.First() {
		return leveldb.ErrNotFound
	}
	Set(obj, iter)
	return nil
}

func GetLastItem(obj ObjectI, prefix []byte) error {
	db, err := GetDb()
	if err != nil {
		return fmt.Errorf("%w; error getting database handler for save", err)
	}
	topicPrefix := jutil.CombineBytes([]byte(obj.GetPrefix()), []byte{Spacer})
	rng := util.BytesPrefix(jutil.CombineBytes(topicPrefix, prefix))
	iter := db.NewIterator(rng, nil)
	if !iter.Last() {
		return leveldb.ErrNotFound
	}
	Set(obj, iter)
	return nil
}

func GetNum(prefix []byte) (int, error) {
	db, err := GetDb()
	if err != nil {
		return 0, fmt.Errorf("%w; error getting database handler for get num db objects", err)
	}
	iter := db.NewIterator(util.BytesPrefix(prefix), nil)
	defer iter.Release()
	var count int
	for iter.Next() {
		count++
	}
	return count, nil
}

func Set(obj ObjectI, iter iterator.Iterator) {
	obj.SetUid(iter.Key()[len(obj.GetPrefix())+1:])
	obj.Deserialize(iter.Value())
}
