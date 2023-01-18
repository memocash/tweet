package update

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/database"
	"github.com/memocash/tweet/tweets"
	"github.com/memocash/tweet/tweets/obj"
	"github.com/spf13/cobra"
)

var profileCmd = &cobra.Command{
	Use:   "profiletext",
	Short: "Update profile description on Memo to match a Twitter account",
	Args:  cobra.ExactArgs(2),
	Run: func(c *cobra.Command, args []string) {
		accountKey := obj.GetAccountKeyFromArgs(args)
		profile, err := tweets.GetProfile(accountKey.Account, tweets.Connect())
		if err != nil {
			jerr.Get("fatal error getting profile", err).Fatal()
		}
		err = database.UpdateProfileText(database.NewWallet(accountKey.Address, accountKey.Key, nil), profile.Description)
		if err != nil {
			jerr.Get("fatal error updating profile text", err).Fatal()
		}
	},
}
