package memobot

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/config"
	"github.com/memocash/tweet/database/util"
	"github.com/spf13/cobra"
)

var memobotCmd = &cobra.Command{
	Use:   "memobot",
	Short: "Listens for new transactions on a memo account",
	Long:  "Prints out each new transaction as it comes in. ",
	RunE: func(c *cobra.Command, args []string) error {
		botKey := config.GetConfig().BotKey
		err := util.MemoListen([]string{botKey})
		if err != nil {
			return jerr.Get("error listening for transactions", err)
		}
		return nil
	},
}

func GetCommand() *cobra.Command {
	return memobotCmd
}
