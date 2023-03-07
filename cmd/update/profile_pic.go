package update

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/tweets"
	"github.com/memocash/tweet/tweets/obj"
	"github.com/memocash/tweet/wallet"
	"github.com/spf13/cobra"
)

var profilePicCmd = &cobra.Command{
	Use:   "profilepic",
	Short: "Update profile picture on Memo to match a Twitter account",
	Args:  cobra.ExactArgs(2),
	Run: func(c *cobra.Command, args []string) {
		accountKey := obj.GetAccountKeyFromArgs(args)
		profile, err := tweets.GetProfile(accountKey.UserID, tweets.Connect())
		if err != nil {
			jerr.Get("fatal error getting profile", err).Fatal()
		}
		err = wallet.UpdateProfilePic(wallet.NewWallet(accountKey.Address, accountKey.Key), profile.ProfilePic)
		if err != nil {
			jerr.Get("fatal error updating profile pic", err).Fatal()
		}
	},
}
