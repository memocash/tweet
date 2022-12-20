package update

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/database"
	"github.com/memocash/tweet/tweets"
	"github.com/spf13/cobra"
)

var nameCmd = &cobra.Command{
	Use:   "name",
	Short: "Update profile name on Memo to match a Twitter account",
	Args:  cobra.ExactArgs(2),
	Run: func(c *cobra.Command, args []string) {
		accountKey := tweets.GetAccountKeyFromArgs(args)
		profile, err := tweets.GetProfile(accountKey.Account, tweets.Connect())
		if err != nil {
			jerr.Get("fatal error getting profile", err).Fatal()
		}
		err = database.UpdateName(database.NewWallet(accountKey.Address, accountKey.Key), profile.Name)
		if err != nil {
			jerr.Get("error", err).Fatal()
		}
	},
}
