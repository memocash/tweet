package db

import (
	"fmt"
	"github.com/jchavannes/jgo/jutil"
	"github.com/memocash/tweet/database"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

const (
	PrefixAddressSeenTx = "address-seen-tx"
	PrefixCompletedTx   = "completed"
)

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
	db, err := database.GetDb()
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
	db, err := database.GetDb()
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
	db, err := database.GetDb()
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

func GetLastItem(obj ObjectI, prefix []byte) error {
	db, err := database.GetDb()
	if err != nil {
		return fmt.Errorf("%w; error getting database handler for save", err)
	}
	topicPrefix := jutil.CombineBytes([]byte(obj.GetPrefix()), []byte{Spacer})
	rng := util.BytesPrefix(jutil.CombineBytes(topicPrefix, prefix))
	iter := db.NewIterator(rng, nil)
	if !iter.Last() {
		return leveldb.ErrNotFound
	}
	obj.SetUid(iter.Key()[len(topicPrefix):])
	obj.Deserialize(iter.Value())
	return nil
}
