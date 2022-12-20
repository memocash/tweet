package update

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/database"
	"github.com/memocash/tweet/tweets"
	"github.com/spf13/cobra"
)

var profilePicCmd = &cobra.Command{
	Use:   "profilepic",
	Short: "Update profile picture on Memo to match a Twitter account",
	Args:  cobra.ExactArgs(2),
	RunE: func(c *cobra.Command, args []string) error {
		accountKey := tweets.GetAccountKeyFromArgs(args)
		_, _, pic, _ := tweets.GetProfile(accountKey.Account, tweets.Connect())
		err := database.UpdateProfilePic(database.NewWallet(accountKey.Address, accountKey.Key), pic)
		if err != nil {
			jerr.Get("error", err).Fatal()
		}
		return nil
	},
}
