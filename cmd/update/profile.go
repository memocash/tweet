package update

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/tweets"
	"github.com/memocash/tweet/tweets/obj"
	"github.com/memocash/tweet/wallet"
	twitterscraper "github.com/n0madic/twitter-scraper"
	"github.com/spf13/cobra"
)

var profileCmd = &cobra.Command{
	Use:   "profiletext",
	Short: "Update profile description on Memo to match a Twitter account",
	Args:  cobra.ExactArgs(2),
	Run: func(c *cobra.Command, args []string) {
		accountKey := obj.GetAccountKeyFromArgs(args)
		scraper := twitterscraper.New()
		scraper.SetSearchMode(twitterscraper.SearchLatest)
		profile, err := tweets.GetProfile(accountKey.UserID, scraper)
		if err != nil {
			jerr.Get("fatal error getting profile", err).Fatal()
		}
		err = wallet.UpdateProfileText(wallet.NewWallet(accountKey.Address, accountKey.Key), profile.Description)
		if err != nil {
			jerr.Get("fatal error updating profile text", err).Fatal()
		}
	},
}
