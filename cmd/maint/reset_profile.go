package maint

import (
	"github.com/jchavannes/jgo/jutil"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/db"
	"github.com/spf13/cobra"
	"log"
	"strings"
)

var resetProfileCmd = &cobra.Command{
	Use:   "reset-profile",
	Short: "reset-profile <ownerAddr> <userId>",
	Run: func(c *cobra.Command, args []string) {
		if len(args) < 2 {
			log.Fatal("must specify owner address and userId")
		}
		owner, err := wallet.GetAddrFromString(args[0])
		if err != nil {
			log.Fatalf("error parsing owner address; %v", err)
		}
		userId := jutil.GetInt64FromString(strings.TrimLeft(args[1], "0"))
		if err := db.Delete([]db.ObjectI{&db.Profile{
			Owner:  *owner,
			UserID: userId,
		}}); err != nil {
			log.Fatalf("error removing profile from db for reset; %v", err)
		}
		log.Printf("reset %d profile linked to %s\n", userId, owner)
	},
}
