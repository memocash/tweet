package maint

import (
	"github.com/jchavannes/btcd/chaincfg/chainhash"
	"github.com/memocash/tweet/database"
	"github.com/spf13/cobra"
	"log"
)

var removeCompletedCmd = &cobra.Command{
	Use:   "remove-completed",
	Short: "remove-completed <tx_hash>",
	Run: func(c *cobra.Command, args []string) {
		if len(args) < 1 {
			log.Fatal("must specify tx hash to be removed")
		}
		txHash, err := chainhash.NewHashFromStr(args[0])
		if err != nil {
			log.Fatalf("error bad tx hash; %v", err)
		}
		db, err := database.GetDb()
		if err != nil {
			log.Fatalf("error opening db; %v", err)
		}
		if err := db.Delete([]byte("completed-"+txHash.String()), nil); err != nil {
			log.Fatalf("error removing completed tx item; %v", err)
		}
		log.Printf("removed completed tx: %s\n", txHash.String())
	},
}
