package maint

import (
	"github.com/jchavannes/btcd/chaincfg/chainhash"
	"github.com/memocash/tweet/db"
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
		hasCompleted, err := db.HasCompletedTx(*txHash)
		if err != nil {
			log.Fatalf("error getting completed tx item; %v", err)
		}
		log.Printf("checked completed tx: %s, %t\n", txHash.String(), hasCompleted)
	},
}
