package info

import (
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/bot/info"
	"github.com/spf13/cobra"
	"log"
)

var balanceCmd = &cobra.Command{
	Use:   "balance",
	Short: "balance",
	Run: func(c *cobra.Command, args []string) {
		if len(args) < 1 {
			log.Fatalf("must specify address")
		}
		addr, err := wallet.GetAddrFromString(args[0])
		if err != nil {
			log.Fatalf("error getting address; %v", err)
		}
		if err := info.Balance(*addr); err != nil {
			log.Fatalf("error info balance; %v", err)
		}
	},
}
