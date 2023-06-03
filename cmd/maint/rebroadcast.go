package maint

import (
	"github.com/memocash/index/client/lib/graph"
	"github.com/memocash/tweet/config"
	"github.com/spf13/cobra"
	"log"
	"os"
	"strings"
)

var rebroadcastCmd = &cobra.Command{
	Use:   "rebroadcast",
	Short: "Rebroadcasts all unsaved txs",
	Run: func(c *cobra.Command, args []string) {
		if len(args) != 1 {
			log.Fatal("Please provide a file path to a file containing raw txs")
		}
		//read the file into a string
		f, err := os.ReadFile(args[0])
		if err != nil {
			log.Fatal(err)
		}
		txs := strings.Split(string(f), "\n")
		//it's ok for rebroadcast to fail, so we don't crash on errors
		for _, tx := range txs {
			err := graph.Broadcast(config.GetGraphQlUrl(), tx)
			if err != nil {
				log.Println(err)
			}
		}
	},
}
