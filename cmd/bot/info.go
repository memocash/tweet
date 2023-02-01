package bot

import (
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/bot"
	"github.com/spf13/cobra"
	"log"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "info",
	Run: func(c *cobra.Command, args []string) {
		if len(args) < 1 {
			log.Fatalf("must specify info command")
		}
		switch args[0] {
		case "bal":
			if len(args) < 2 {
				log.Fatalf("must specify address")
			}
			addr, err := wallet.GetAddrFromString(args[1])
			if err != nil {
				log.Fatalf("error getting address; %v", err)
			}
			if err := bot.InfoBalance(*addr); err != nil {
				log.Fatalf("error info balance; %v", err)
			}
		default:
			log.Fatalf("unknown info command: %s", args[0])
		}
	},
}
