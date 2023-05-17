package bot

import (
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/bot/info"
	"github.com/spf13/cobra"
	"log"
	"strconv"
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
			if err := info.Balance(*addr); err != nil {
				log.Fatalf("error info balance; %v", err)
			}
		case "profile":
			if len(args) < 3 {
				log.Fatalf("must specify the sender address and twittername")
			}
			senderAdder, err := wallet.GetAddrFromString(args[1])
			if err != nil {
				log.Fatalf("error getting address; %v", err)
			}
			userId, err := strconv.ParseInt(args[2], 10, 64)
			if err != nil {
				log.Fatalf("error parsing userId; %v", err)
			}
			if err := info.Profile(*senderAdder, userId); err != nil {
				log.Fatalf("error info profile; %v", err)
			}
		case "report":
			if len(args) != 1 {
				log.Fatalf("report takes no arguments")
			}
			if err := info.Report(); err != nil {
				log.Fatalf("error info report; %v", err)
			}
		default:
			log.Fatalf("unknown info command: %s", args[0])
		}
	},
}
