package maint

import (
	"github.com/jchavannes/btcd/chaincfg/chainhash"
	"github.com/jchavannes/jgo/jerr"
	"github.com/jchavannes/jgo/jlog"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/db"
	"github.com/spf13/cobra"
	"log"
	"time"
)

var checkAddressSeenCmd = &cobra.Command{
	Use:   "check-address-seen",
	Short: "check-address-seen",
	Run: func(c *cobra.Command, args []string) {
		allAddressSeenTxs, err := db.GetAllAddressSeenTx()
		if err != nil {
			jerr.Get("error getting all address seen tx", err).Fatal()
		}
		var cnt int
		for _, addressSeenTx := range allAddressSeenTxs {
			jlog.Logf("address: %s, tx: %s, seen: %s\n", wallet.Addr(addressSeenTx.Address),
				chainhash.Hash(addressSeenTx.TxHash), addressSeenTx.Seen.Format(time.RFC3339))
			cnt++
		}
		log.Printf("check addresse seen: %d\n", cnt)
	},
}

var removeInvalidAddressSeenCmd = &cobra.Command{
	Use:   "remove-invalid-address-seen",
	Short: "remove-invalid-address-seen",
	Run: func(c *cobra.Command, args []string) {
		allAddressSeenTxs, err := db.GetAllAddressSeenTx()
		if err != nil {
			jerr.Get("error getting all address seen tx", err).Fatal()
		}
		var cnt int
		for _, addressSeenTx := range allAddressSeenTxs {
			if addressSeenTx.Seen.Before(time.Date(2009, 1, 1, 1, 0, 0, 0, time.Local)) || addressSeenTx.Seen.After(time.Now()) {
				jlog.Logf("invalid address seen found address: %s, tx: %s, seen: %s\n", wallet.Addr(addressSeenTx.Address),
					chainhash.Hash(addressSeenTx.TxHash), addressSeenTx.Seen.Format(time.RFC3339))
				cnt++
				if err := db.Delete([]db.ObjectI{addressSeenTx}); err != nil {
					jerr.Get("error deleting invalid address seen; %v\n", err).Fatal()
				}
			}
		}
		log.Printf("invalid address seens removed: %d\n", cnt)
	},
}
