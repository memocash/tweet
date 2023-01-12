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
		address,err := wallet.GetAddrFromString("1EVenAyKGr1TgEetVVKg5x2GysDhoGmBPM")
		if err != nil {
			jerr.Get("error getting address", err).Fatal()
		}
		balance, err := client.Database.GetAddressBalance(address)
		if err != nil {
			jerr.Get("error getting address balance", err).Fatal()
		}
		println(balance)

	},
}
func GetCommand() *cobra.Command {
	return testCmd
}

