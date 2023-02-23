package maint

import (
	"encoding/json"
	"github.com/memocash/tweet/database"
	"github.com/memocash/tweet/tweets/obj"
	"github.com/spf13/cobra"
	"github.com/syndtr/goleveldb/leveldb/util"
	"log"
)

var checkSavedTweetsCmd = &cobra.Command{
	Use:   "check-saved-tweets",
	Short: "check-saved-tweets",
	Run: func(c *cobra.Command, args []string) {
		if len(args) != 1 {
			log.Fatalf("must specify the prefix")
		}
		prefix := args[0]
		//open the database print out every row in the database that matches saved-address-twitterName
		db, err := database.GetDb()
		if err != nil {
			panic(err)
		}
		defer db.Close()
		iter := db.NewIterator(util.BytesPrefix([]byte(prefix)), nil)
		for iter.Next() {
			key := iter.Key()
			value := iter.Value()
			TweetTx := obj.TweetTx{}
			//unmarshal the value into a TweetTx
			err := json.Unmarshal(value, &TweetTx)
			if err != nil {
				panic(err)
			}

			log.Printf("%s: %s\n", key, TweetTx.Tweet.ID)
		}
	},
}
