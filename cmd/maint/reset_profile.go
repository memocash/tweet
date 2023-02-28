package maint

import (
	"github.com/memocash/tweet/db"
	"github.com/spf13/cobra"
	"log"
)

var resetProfileCmd = &cobra.Command{
	Use:   "reset-profile",
	Short: "reset-profile <senderAddr> <twittername>",
	Run: func(c *cobra.Command, args []string) {
		if len(args) < 2 {
			log.Fatal("must specify sender address and twittername")
		}
		senderAddr := args[0]
		twittername := args[1]
		if err := db.Delete([]db.ObjectI{&db.Profile{
			Address:     senderAddr,
			TwitterName: twittername,
		}}); err != nil {
			log.Fatalf("error removing profile from db for reset; %v", err)
		}
		log.Printf("reset %s profile linked to %s\n", twittername, senderAddr)
	},
}
