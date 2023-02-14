package maint

import (
	"github.com/jchavannes/btcd/chaincfg/chainhash"
	"github.com/jchavannes/jgo/jlog"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/database"
	"github.com/memocash/tweet/db"
	"github.com/spf13/cobra"
	"github.com/syndtr/goleveldb/leveldb/util"
	"log"
	"time"
)

var checkAddressSeenCmd = &cobra.Command{
	Use:   "check-address-seen",
	Short: "check-address-seen",
	Run: func(c *cobra.Command, args []string) {
		levelDb, err := database.GetDb()
		if err != nil {
			log.Fatalf("error opening db; %v", err)
		}
		prefix := []byte(db.PrefixAddressSeenTx + "-")
		iter := levelDb.NewIterator(util.BytesPrefix(prefix), nil)
		var cnt int
		for iter.Next() {
			key := iter.Key()
			var addressSeenTx = &db.AddressSeenTx{}
			key = key[len(prefix):]
			addressSeenTx.SetUid(key)
			jlog.Logf("key: %x, address: %s, tx: %s, seen: %s\n", key, wallet.Addr(addressSeenTx.Address),
				chainhash.Hash(addressSeenTx.TxHash), addressSeenTx.Seen.Format(time.RFC3339))
			cnt++
		}
		log.Printf("check addresse seen: %d\n", cnt)
	},
}
