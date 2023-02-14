package maint

import (
	"github.com/jchavannes/btcd/chaincfg/chainhash"
	"github.com/jchavannes/jgo/jlog"
	"github.com/memocash/tweet/database"
	"github.com/memocash/tweet/db"
	"github.com/spf13/cobra"
	"github.com/syndtr/goleveldb/leveldb/util"
	"log"
	"strings"
)

var convertCompletedCmd = &cobra.Command{
	Use:   "convert-completed",
	Short: "convert-completed",
	Run: func(c *cobra.Command, args []string) {
		levelDb, err := database.GetDb()
		if err != nil {
			log.Fatalf("error opening db; %v", err)
		}
		iter := levelDb.NewIterator(util.BytesPrefix([]byte(db.PrefixCompletedTx+"-")), nil)
		var cnt int
		for iter.Next() {
			key := iter.Key()
			jlog.Logf("key: %x, len: %d\n", key, len(key))
			cnt++
			if len(key) != 74 {
				continue
			}
			keyStr := string(key)
			keyStr = strings.TrimPrefix(keyStr, "completed-")
			txHash, err := chainhash.NewHashFromStr(keyStr)
			if err != nil {
				log.Fatalf("error bad tx hash; %v", err)
			}
			var completedTx = db.CompletedTx{TxHash: *txHash}
			if err := db.Save([]db.ObjectI{&completedTx}); err != nil {
				log.Fatalf("error saving completed tx item; %v", err)
			}
			if err := levelDb.Delete([]byte("completed-"+txHash.String()), nil); err != nil {
				log.Fatalf("error removing completed tx item; %v", err)
			}
		}
		log.Printf("convert completed tx: %d\n", cnt)
	},
}
