package updatename

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/cmd/util"
	"github.com/memocash/tweet/database"
	"github.com/memocash/tweet/tweets"
	"github.com/spf13/cobra"
)

var nameCmd = &cobra.Command{
	Use:   "updatename",
	Short: "Update profile name on Memo to match a Twitter account",
	Args:  cobra.ExactArgs(2),
	RunE: func(c *cobra.Command, args []string) error {
		key, address, account := util.Setup(args)
		name, _, _, _ := tweets.GetProfile(account, tweets.Connect())
		err := database.UpdateName(database.NewWallet(address, key), name)
		if err != nil {
			jerr.Get("error", err).Fatal()
		}
		return nil
	},
}

func GetCommand() *cobra.Command {
	return nameCmd
}
