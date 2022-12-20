package update

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/database"
	"github.com/memocash/tweet/tweets"
	"github.com/spf13/cobra"
)

var profileCmd = &cobra.Command{
	Use:   "profiletext",
	Short: "Update profile description on Memo to match a Twitter account",
	Args:  cobra.ExactArgs(2),
	RunE: func(c *cobra.Command, args []string) error {
		accountKey := tweets.GetAccountKeyFromArgs(args)
		_, desc, _, _ := tweets.GetProfile(accountKey.Account, tweets.Connect())
		if desc == "" {
			desc = " "
		}
		err := database.UpdateProfileText(database.NewWallet(accountKey.Address, accountKey.Key), desc)
		if err != nil {
			jerr.Get("error", err).Fatal()
		}
		return nil
	},
}
