package maint

import (
	"github.com/jchavannes/btcd/chaincfg/chainhash"
	"github.com/memocash/tweet/database"
	"github.com/spf13/cobra"
	"log"
)

var checkCompletedCmd = &cobra.Command{
	Use:   "check-completed",
	Short: "check-completed <tx_hash>",
	Run: func(c *cobra.Command, args []string) {
		if len(args) < 1 {
			log.Fatal("must specify tx hash to be checked")
		}
		txHash, err := chainhash.NewHashFromStr(args[0])
		if err != nil {
			log.Fatalf("error bad tx hash; %v", err)
		}
		db, err := database.GetDb()
		if err != nil {
			log.Fatalf("error opening db; %v", err)
		}
		val, err := db.Get([]byte("completed-"+txHash.String()), nil)
		if err != nil {
			log.Fatalf("error getting completed tx item; %v", err)
		}
		log.Printf("checked completed tx: %s, %x\n", txHash.String(), val)
	},
}
