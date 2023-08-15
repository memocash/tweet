package db

import (
	"encoding/json"
	"errors"
	"github.com/jchavannes/btcd/chaincfg/chainhash"
	"github.com/memocash/index/client/lib/graph"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/db"
	"github.com/spf13/cobra"
	"github.com/syndtr/goleveldb/leveldb"
	"log"
)

var outputsCmd = &cobra.Command{
	Use:   "outputs",
	Short: "outputs (address)",
	Run: func(c *cobra.Command, args []string) {
		var address *wallet.Addr
		var err error
		if len(args) > 0 {
			address, err = wallet.GetAddrFromString(args[0])
			if err != nil {
				log.Fatalf("error getting address from string; %v\n", err)
			}
		}
		outputs, err := db.GetTxOutputs([]wallet.Addr{*address})
		if err != nil {
			log.Fatalf("error getting tx outputs; %v\n", err)
		}
		for _, output := range outputs {
			input, err := db.GetTxInput(output.TxHash, output.Index)
			if err != nil && !errors.Is(err, leveldb.ErrNotFound) {
				log.Fatalf("error getting tx input; %v\n", err)
			}
			var graphOutput graph.Output
			if err := json.Unmarshal(output.Output, &graphOutput); err != nil {
				log.Fatalf("error unmarshalling tx output; %v", err)
			}
			log.Printf("output: %s:%d - %d (spent: %t)\n",
				chainhash.Hash(output.TxHash), output.Index, graphOutput.Amount, input != nil)
		}
	},
}
