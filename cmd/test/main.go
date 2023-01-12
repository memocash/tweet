package test

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/database"
	"github.com/spf13/cobra"
)

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Debugging (currently for database debugging)",
	Run: func(c *cobra.Command, args []string) {
		println("test")
		client,err := database.GetClient()
		if err != nil {
			jerr.Get("error getting database client", err).Fatal()
		}
		//create a wallet.Address object
		address,err := wallet.GetAddrFromString("1DHZohwLk7qQocABkHoZCU6vyhRPYY4njV")
		if err != nil {
			jerr.Get("error getting address", err).Fatal()
		}
		utxos,err  := client.GetUtxos(address)
		if err != nil {
			jerr.Get("error getting utxos", err).Fatal()
		}
		println("utxos",len(utxos))
		height, err := client.Database.GetAddressHeight(address)
		if err != nil {
			jerr.Get("error getting address height", err).Fatal()
		}
		println(height)

	},
}
func GetCommand() *cobra.Command {
	return testCmd
}

