package maint

import (
	"github.com/dghubble/go-twitter/twitter"
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/db"
	"github.com/memocash/tweet/tweets"
	"github.com/spf13/cobra"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"log"
	"strings"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "migrate",
	Run: func(c *cobra.Command, args []string) {
		levelDb, err := db.GetDb()
		if err != nil {
			log.Fatalf("error opening db; %v", err)
		}
		client := tweets.Connect()
		twitterNames, err := GetAllTwitterNames(levelDb)
		if err != nil {
			log.Fatalf("error getting twitter names; %v", err)
		}
		for _, twittername := range twitterNames {
			println("migrating", twittername)
			userShowParams := &twitter.UserShowParams{ScreenName: twittername}
			user, _, err := client.Users.Show(userShowParams)
			if err != nil {
				log.Fatalf("error getting user; %v", err)
			}
			userId := user.IDStr
			if err := migrateDB(twittername, userId, levelDb); err != nil {
				log.Fatalf("error migrating db; %v", err)
			}
		}
	},
}

func migrateDB(twittername string, userId string, levelDb *leveldb.DB) error {
	if err := migratePrefix(db.PrefixAddressKey, twittername, userId, levelDb); err != nil {
		return jerr.Get("error migrating address key", err)
	}
	if err := migratePrefix(db.PrefixFlag, twittername, userId, levelDb); err != nil {
		return jerr.Get("error migrating flag", err)
	}
	if err := migratePrefix(db.PrefixProfile, twittername, userId, levelDb); err != nil {
		return jerr.Get("error migrating profile", err)
	}
	if err := migratePrefix(db.PrefixSavedAddressTweet, twittername, userId, levelDb); err != nil {
		return jerr.Get("error migrating saved address tweet", err)
	}
	if err := migratePrefix(db.PrefixTweetTx, twittername, userId, levelDb); err != nil {
		return jerr.Get("error migrating tweet tx", err)
	}
	return nil
}

func migratePrefix(prefix string, twittername string, userId string, levelDb *leveldb.DB) error {
	iter := levelDb.NewIterator(util.BytesPrefix([]byte(prefix)), nil)
	defer iter.Release()
	for iter.Next() {
		key := string(iter.Key())
		if !strings.Contains(key, twittername) {
			continue
		}
		newKey := strings.Replace(key, twittername, userId, 1)
		err := levelDb.Put([]byte(newKey), iter.Value(), nil)
		if err != nil {
			return err
		}
		err = levelDb.Delete(iter.Key(), nil)
		if err != nil {
			return err
		}
	}
	return nil
}

func GetAllTwitterNames(levelDb *leveldb.DB) ([]string, error) {
	iter := levelDb.NewIterator(util.BytesPrefix([]byte(db.PrefixProfile)), nil)
	defer iter.Release()
	var twitterNames []string
	for iter.Next() {
		//twittername is the last part of the key, separated by a dash
		key := string(iter.Key())
		parts := strings.Split(key, "-")
		twitterNames = append(twitterNames, parts[len(parts)-1])
	}
	return twitterNames, nil
}
