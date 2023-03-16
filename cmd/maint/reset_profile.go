package maint

import (
	"github.com/jchavannes/jgo/jutil"
	"github.com/memocash/tweet/db"
	"github.com/spf13/cobra"
	"log"
	"strings"
)

var resetProfileCmd = &cobra.Command{
	Use:   "reset-profile",
	Short: "reset-profile <senderAddr> <userId>",
	Run: func(c *cobra.Command, args []string) {
		if len(args) < 2 {
			log.Fatal("must specify sender address and userId")
		}
		senderAddr := args[0]
		userId := jutil.GetInt64FromString(strings.TrimLeft(args[1], "0"))
		if err := db.Delete([]db.ObjectI{&db.Profile{
			Address: senderAddr,
			UserID:  userId,
		}}); err != nil {
			log.Fatalf("error removing profile from db for reset; %v", err)
		}
		log.Printf("reset %s profile linked to %s\n", userId, senderAddr)
	},
}
