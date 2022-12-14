package memobot

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/config"
	"github.com/memocash/tweet/database/util"
	"github.com/memocash/tweet/tweets"
	"github.com/spf13/cobra"
)

var memobotCmd = &cobra.Command{
	Use:   "memobot",
	Short: "Listens for new transactions on a memo account",
	Long:  "Prints out each new transaction as it comes in. ",
	RunE: func(c *cobra.Command, args []string) error {
		botKey,err := wallet.ImportPrivateKey(config.GetConfig().BotKey)
		if err != nil {
			return jerr.Get("error importing bot key", err)
		}
		println(botKey.GetBase58Compressed())
		botAddress := botKey.GetAddress().GetEncoded()
		println(botAddress)
		err = util.MemoListen([]string{botAddress},botKey,tweets.Connect())
		if err != nil {
			return jerr.Get("error listening for transactions", err)
		}
		return nil
	},
}

func GetCommand() *cobra.Command {
	return memobotCmd
}
