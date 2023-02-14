package db

import (
	"fmt"
	"github.com/jchavannes/jgo/jerr"
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
		return jerr.Get("error getting database handler for save", err)
	}
	batch := new(leveldb.Batch)
	for _, object := range objects {
		batch.Put(GetObjectCombinedUid(object), object.Serialize())
	}
	if err := db.Write(batch, nil); err != nil {
		return jerr.Get("error saving leveldb objects", err)
	}
	return nil
}

func GetItem(obj ObjectI) error {
	db, err := database.GetDb()
	if err != nil {
		return jerr.Get("error getting database handler for save", err)
	}
	val, err := db.Get(GetObjectCombinedUid(obj), nil)
	if err != nil {
		return jerr.Get("error getting db item single", err)
	}
	obj.Deserialize(val)
	return nil
}

func GetLastItem(obj ObjectI, prefix []byte) error {
	db, err := database.GetDb()
	if err != nil {
		return fmt.Errorf("%w; error getting database handler for save", err)
	}
	rng := util.BytesPrefix(jutil.CombineBytes([]byte(obj.GetPrefix()), []byte{Spacer}, prefix))
	iter := db.NewIterator(rng, nil)
	if !iter.Last() {
		return leveldb.ErrNotFound
	}
	obj.SetUid(iter.Key())
	obj.Deserialize(iter.Value())
	return nil
}
