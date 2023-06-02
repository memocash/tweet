package info

import (
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/bot/info"
	"github.com/spf13/cobra"
	"log"
	"strconv"
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "profile",
	Run: func(c *cobra.Command, args []string) {
		if len(args) < 2 {
			log.Fatalf("must specify the owner address and twittername")
		}
		ownerAddr, err := wallet.GetAddrFromString(args[0])
		if err != nil {
			log.Fatalf("error getting owner address; %v", err)
		}
		userId, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			log.Fatalf("error parsing userId; %v", err)
		}
		if err := info.Profile(*ownerAddr, userId); err != nil {
			log.Fatalf("error info profile; %v", err)
		}
	},
}
