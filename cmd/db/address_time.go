package db

import (
	"errors"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/db"
	"github.com/spf13/cobra"
	"github.com/syndtr/goleveldb/leveldb"
	"log"
)

var addressTimeCmd = &cobra.Command{
	Use:   "address-time",
	Short: "address-time [address]",
	Run: func(c *cobra.Command, args []string) {
		if len(args) == 0 {
			log.Fatalf("must specify address\n")
		}
		address, err := wallet.GetAddrFromString(args[0])
		if err != nil {
			log.Fatalf("error getting address from string; %v\n", err)
		}
		addressTime, err := db.GetAddressTime(*address)
		if err != nil && !errors.Is(err, leveldb.ErrNotFound) {
			log.Fatalf("error getting address time; %v\n", err)
		}
		if addressTime == nil {
			log.Println("address time not found")
			return
		}
		log.Printf("address-time: %s - %s\n",
			wallet.Addr(addressTime.Address), addressTime.Time.Format("2006-01-02 03:04:05"))
	},
}
