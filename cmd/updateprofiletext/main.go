package updateprofiletext

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/cmd/util"
	"github.com/memocash/tweet/database"
	"github.com/memocash/tweet/tweets"
	"github.com/spf13/cobra"
)

var nameCmd = &cobra.Command{
	Use:   "updateprofiletext",
	Short: "Update profile description on Memo to match a Twitter account",
	Args:  cobra.ExactArgs(2),
	RunE: func(c *cobra.Command, args []string) error {
		key, address, account := util.Setup(args)
		_, desc, _, _ := tweets.GetProfile(account, tweets.Connect())
		err := database.UpdateProfileText(database.NewWallet(address, key), desc)
		if err != nil {
			jerr.Get("error", err).Fatal()
		}
		return nil
	},
}

func GetCommand() *cobra.Command {
	return nameCmd
}
