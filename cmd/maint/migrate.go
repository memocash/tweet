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
	//err = MigrateTxInput(levelDb)
	//if err != nil {
	//	return jerr.Get("error migrating tx input", err)
	//}
	err = MigrateTxOutput(levelDb)
	if err != nil {
		return jerr.Get("error migrating tx output", err)
	}
	return nil
}

func MigrateTxOutput(levelDb *leveldb.DB) error {
	//old: output-<address>-<txHash>-<index>
	//new: output-<address><txHash><index>
	iter := levelDb.NewIterator(util.BytesPrefix([]byte(db.PrefixTxOutput)), nil)
	for iter.Next() {
		key := iter.Key()
		println(string(key))
		parts := strings.Split(string(key), string([]byte{db.Spacer}))
		if len(parts) != 4 {
			continue
		}
		address := parts[1]
		//if address contains "unknown" then skip
		//FIND OUT WHY WE ARE SAVING THESE
		if strings.Contains(address, "unknown") {
			continue
		}
		addr, err := wallet.GetAddrFromString(address)
		if err != nil {
			return jerr.Get("error parsing address", err)
		}
		txHash := parts[2]
		bytesHash, err := chainhash.NewHashFromStr(txHash)
		if err != nil {
			return jerr.Get("error parsing tx hash", err)
		}
		index := parts[3]
		indexInt, err := strconv.Atoi(index)
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
		strAddr := parts[1]
		strUserId := parts[2]
		addr, err := wallet.GetAddrFromString(strAddr)
		if err != nil {
			return jerr.Get("error getting address from string", err)
		}
		userId, err := strconv.ParseInt(strUserId, 10, 64)
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
		strAddr := parts[1]
		strUserId := parts[2]
		addr, err := wallet.GetAddrFromString(strAddr)
		if err != nil {
			return jerr.Get("error getting address from string", err)
		}
		userId, err := strconv.ParseInt(strUserId, 10, 64)
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
