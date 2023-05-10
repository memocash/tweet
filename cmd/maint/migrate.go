package maint

import (
	"github.com/jchavannes/btcd/chaincfg/chainhash"
	"github.com/jchavannes/jgo/jerr"
	"github.com/jchavannes/jgo/jutil"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/db"
	"github.com/spf13/cobra"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"log"
	"strconv"
	"strings"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "migrate",
	Run: func(c *cobra.Command, args []string) {
		levelDb, err := db.GetDb()
		if err != nil {
			log.Fatal(jerr.Get("error getting db", err))
		}
		err = migrateDB(levelDb)
		if err != nil {
			log.Fatal(jerr.Get("error migrating db", err))
		}
	},
}

func migrateDB(levelDb *leveldb.DB) error {
	err := MigrateAddressLinkedKey(levelDb)
	if err != nil {
		return jerr.Get("error migrating address linked key", err)
	}
	err = MigrateFlag(levelDb)
	if err != nil {
		return jerr.Get("error migrating address seen tx", err)
	}
	err = MigrateTxInput(levelDb)
	if err != nil {
		return jerr.Get("error migrating tx input", err)
	}
	err = MigrateTxOutput(levelDb)
	if err != nil {
		return jerr.Get("error migrating tx output", err)
	}
	err = MigrateSavedAddressTweet(levelDb)
	if err != nil {
		return jerr.Get("error migrating saved address tweet", err)
	}
	err = MigrateBlock(levelDb)
	if err != nil {
		return jerr.Get("error migrating block", err)
	}
	err = MigrateProfile(levelDb)
	if err != nil {
		return jerr.Get("error migrating profile", err)
	}
	err = MigrateTweetsTx(levelDb)
	if err != nil {
		return jerr.Get("error migrating tweets tx", err)
	}
	err = MigrateTxBlock(levelDb)
	if err != nil {
		return jerr.Get("error migrating tx block", err)
	}
	return nil
}

func MigrateTxBlock(levelDb *leveldb.DB) error {
	//old: txblock-<txHash>-<blockHash>
	//new: txblock-<txHash><blockHash>
	iter := levelDb.NewIterator(util.BytesPrefix([]byte(db.PrefixTxBlock)), nil)
	for iter.Next() {
		key := iter.Key()
		parts := strings.Split(string(key), string([]byte{db.Spacer}))
		if len(parts) != 3 {
			continue
		}
		txHash, err := chainhash.NewHashFromStr(parts[1])
		if err != nil {
			return jerr.Get("error parsing tx hash", err)
		}
		blockHash, err := chainhash.NewHashFromStr(parts[2])
		if err != nil {
			return jerr.Get("error parsing block hash", err)
		}
		uid := jutil.CombineBytes([]byte(db.PrefixTxBlock), []byte{db.Spacer}, txHash[:], blockHash[:])
		err = levelDb.Put(uid, iter.Value(), nil)
		if err != nil {
			return jerr.Get("error putting new key", err)
		}
		err = levelDb.Delete(key, nil)
		if err != nil {
			return jerr.Get("error deleting old key", err)
		}
	}
	iter.Release()
	return iter.Error()
}

func MigrateTweetsTx(levelDb *leveldb.DB) error {
	//old: tweets-<userId>-<tweetId>
	//new: tweets-<userId><tweetId>
	iter := levelDb.NewIterator(util.BytesPrefix([]byte(db.PrefixTweetTx)), nil)
	for iter.Next() {
		key := iter.Key()
		parts := strings.Split(string(key), string([]byte{db.Spacer}))
		if len(parts) != 3 {
			continue
		}
		userId, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return jerr.Get("error parsing user id", err)
		}
		tweetId, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			return jerr.Get("error parsing tweet id", err)
		}
		userIdBytes := jutil.GetInt64DataBig(userId)
		tweetIdBytes := jutil.GetInt64DataBig(tweetId)
		uid := jutil.CombineBytes([]byte(db.PrefixTweetTx), []byte{db.Spacer}, userIdBytes, tweetIdBytes)
		err = levelDb.Put(uid, iter.Value(), nil)
		if err != nil {
			return jerr.Get("error putting new key", err)
		}
		err = levelDb.Delete(key, nil)
		if err != nil {
			return jerr.Get("error deleting old key", err)
		}
	}
	iter.Release()
	return iter.Error()
}

func MigrateProfile(levelDb *leveldb.DB) error {
	//old: profile-<address>-<userId>
	//new: profile-<address><userId>
	iter := levelDb.NewIterator(util.BytesPrefix([]byte(db.PrefixProfile)), nil)
	for iter.Next() {
		key := iter.Key()
		parts := strings.Split(string(key), string([]byte{db.Spacer}))
		if len(parts) != 3 {
			continue
		}
		address, err := wallet.GetAddrFromString(parts[1])
		if err != nil {
			log.Printf("error getting address from string, deleting: %s", parts[1])
			err = levelDb.Delete(key, nil)
			if err != nil {
				return jerr.Get("error deleting old key", err)
			}
			continue
		}
		userId, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			return jerr.Get("error parsing user id", err)
		}
		userIdBytes := jutil.GetInt64DataBig(userId)
		uid := jutil.CombineBytes([]byte(db.PrefixProfile), []byte{db.Spacer}, address[:], userIdBytes)
		err = levelDb.Put(uid, iter.Value(), nil)
		if err != nil {
			return jerr.Get("error putting new key", err)
		}
		err = levelDb.Delete(key, nil)
		if err != nil {
			return jerr.Get("error deleting old key", err)
		}
	}
	return nil
}

func MigrateBlock(levelDb *leveldb.DB) error {
	//old: block-<blockHash>
	//new: block-<blockHash>
	iter := levelDb.NewIterator(util.BytesPrefix([]byte(db.PrefixBlock)), nil)
	for iter.Next() {
		key := iter.Key()
		parts := strings.Split(string(key), string([]byte{db.Spacer}))
		if len(parts) != 2 {
			continue
		}
		blockHash, err := chainhash.NewHashFromStr(parts[1])
		if err != nil {
			return jerr.Get("error getting block hash", err)
		}
		uid := jutil.CombineBytes([]byte(db.PrefixBlock), []byte{db.Spacer}, blockHash[:])
		err = levelDb.Put(uid, iter.Value(), nil)
		if err != nil {
			return jerr.Get("error putting new key", err)
		}
		err = levelDb.Delete(key, nil)
		if err != nil {
			return jerr.Get("error deleting old key", err)
		}
	}
	return nil
}

func MigrateSavedAddressTweet(levelDb *leveldb.DB) error {
	//old: saved-<address>-<userId>-<tweetId>
	//new: saved-<address><userId><tweetId>
	iter := levelDb.NewIterator(util.BytesPrefix([]byte(db.PrefixSavedAddressTweet)), nil)
	for iter.Next() {
		key := iter.Key()
		parts := strings.Split(string(key), string([]byte{db.Spacer}))
		if len(parts) != 4 {
			continue
		}
		address, err := wallet.GetAddrFromString(parts[1])
		if err != nil {
			log.Printf("error getting address from string, deleting: %s", parts[1])
			err = levelDb.Delete(key, nil)
			if err != nil {
				return jerr.Get("error deleting old key", err)
			}
			continue
		}
		userId, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			return jerr.Get("error parsing user id", err)
		}
		userIdBytes := jutil.GetInt64DataBig(userId)
		tweetId, err := strconv.ParseInt(parts[3], 10, 64)
		if err != nil {
			return jerr.Get("error parsing tweet id", err)
		}
		tweetIdBytes := jutil.GetInt64DataBig(tweetId)
		uid := jutil.CombineBytes([]byte(db.PrefixSavedAddressTweet), []byte{db.Spacer}, address[:], userIdBytes, tweetIdBytes)
		err = levelDb.Put(uid, iter.Value(), nil)
		if err != nil {
			return jerr.Get("error putting new key", err)
		}
		err = levelDb.Delete(key, nil)
		if err != nil {
			return jerr.Get("error deleting old key", err)
		}
	}
	return nil
}

func MigrateTxOutput(levelDb *leveldb.DB) error {
	//old: output-<address>-<txHash>-<index>
	//new: output-<address><txHash><index>
	iter := levelDb.NewIterator(util.BytesPrefix([]byte(db.PrefixTxOutput)), nil)
	for iter.Next() {
		key := iter.Key()
		parts := strings.Split(string(key), string([]byte{db.Spacer}))
		if len(parts) != 4 {
			continue
		}
		if strings.Contains(parts[1], "unknown") {
			log.Printf("error getting address from string, deleting: %s", parts[1])
			err := levelDb.Delete(key, nil)
			if err != nil {
				return jerr.Get("error deleting old key", err)
			}
			continue
		}
		addr, err := wallet.GetAddrFromString(parts[1])
		if err != nil {
			log.Printf("error getting address from string, deleting: %s", parts[1])
			err = levelDb.Delete(key, nil)
			if err != nil {
				return jerr.Get("error deleting old key", err)
			}
			continue
		}
		bytesHash, err := chainhash.NewHashFromStr(parts[2])
		if err != nil {
			return jerr.Get("error parsing tx hash", err)
		}
		indexInt, err := strconv.Atoi(parts[3])
		if err != nil {
			return jerr.Get("error parsing index", err)
		}
		indexBytes := jutil.GetIntData(indexInt)
		uid := jutil.CombineBytes([]byte(db.PrefixTxOutput), []byte{db.Spacer}, addr[:], bytesHash[:], indexBytes)
		err = levelDb.Put(uid, iter.Value(), nil)
		if err != nil {
			return jerr.Get("error putting new key", err)
		}
		err = levelDb.Delete(key, nil)
		if err != nil {
			return jerr.Get("error deleting old key", err)
		}
	}
	return nil
}

func MigrateTxInput(levelDb *leveldb.DB) error {
	//old: input-<prevHash>-<PrevIndex>
	//new: input-<prevHash><PrevIndex>
	iter := levelDb.NewIterator(util.BytesPrefix([]byte(db.PrefixTxInput)), nil)
	for iter.Next() {
		key := iter.Key()
		parts := strings.Split(string(key), string([]byte{db.Spacer}))
		if len(parts) != 3 {
			continue
		}
		prevHash := parts[1]
		bytesHash, err := chainhash.NewHashFromStr(prevHash)
		if err != nil {
			return jerr.Get("error parsing prev hash", err)
		}
		prevIndex := parts[2]
		prevIndexInt, err := strconv.Atoi(prevIndex)
		if err != nil {
			return jerr.Get("error parsing prev index", err)
		}
		prevIndexBytes := jutil.GetIntData(prevIndexInt)
		uid := jutil.CombineBytes([]byte(db.PrefixTxInput), []byte{db.Spacer}, bytesHash[:], prevIndexBytes)
		err = levelDb.Put(uid, iter.Value(), nil)
		if err != nil {
			return jerr.Get("error putting new key", err)
		}
		err = levelDb.Delete(key, nil)
		if err != nil {
			return jerr.Get("error deleting old key", err)
		}
	}
	return nil
}

func MigrateFlag(levelDb *leveldb.DB) error {
	//old flag: flags-<address>-<userId>
	//new flag: flags-<address><userId>
	iter := levelDb.NewIterator(util.BytesPrefix([]byte(db.PrefixFlag)), nil)
	for iter.Next() {
		key := iter.Key()
		parts := strings.Split(string(key), string([]byte{db.Spacer}))
		if len(parts) != 3 {
			continue
		}
		addr, err := wallet.GetAddrFromString(parts[1])
		if err != nil {
			log.Printf("error getting address from string, deleting: %s", parts[1])
			err = levelDb.Delete(key, nil)
			if err != nil {
				return jerr.Get("error deleting old key", err)
			}
			continue
		}
		userId, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			return jerr.Get("error parsing user id", err)
		}
		userIdBytes := jutil.GetInt64DataBig(userId)
		uid := jutil.CombineBytes([]byte(db.PrefixFlag), []byte{db.Spacer}, addr[:], userIdBytes)
		err = levelDb.Put(uid, iter.Value(), nil)
		if err != nil {
			return jerr.Get("error putting new flag", err)
		}
		err = levelDb.Delete(key, nil)
		if err != nil {
			return jerr.Get("error deleting old flag", err)
		}
	}
	return nil
}

func MigrateAddressLinkedKey(levelDb *leveldb.DB) error {
	//old: linked-<address>-userId
	//new: linked-<address><userId>
	iter := levelDb.NewIterator(util.BytesPrefix([]byte(db.PrefixAddressKey)), nil)
	for iter.Next() {
		key := iter.Key()
		parts := strings.Split(string(key), string([]byte{db.Spacer}))
		if len(parts) != 3 {
			continue
		}
		addr, err := wallet.GetAddrFromString(parts[1])
		if err != nil {
			log.Printf("error getting address from string, deleting: %s", parts[1])
			err = levelDb.Delete(key, nil)
			if err != nil {
				return jerr.Get("error deleting old key", err)
			}
			continue
		}
		userId, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			return jerr.Get("error parsing user id", err)
		}
		userIdBytes := jutil.GetInt64DataBig(userId)
		uid := jutil.CombineBytes([]byte(db.PrefixAddressKey), []byte{db.Spacer}, addr[:], userIdBytes)
		err = levelDb.Put(uid, iter.Value(), nil)
		if err != nil {
			return jerr.Get("error putting new key", err)
		}
		err = levelDb.Delete(key, nil)
		if err != nil {
			return jerr.Get("error deleting old key", err)
		}
	}
	return nil
}
