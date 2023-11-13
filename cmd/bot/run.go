package bot

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/bot"
	"github.com/memocash/tweet/config"
	twitterscraper "github.com/n0madic/twitter-scraper"
	"github.com/spf13/cobra"
	"os"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "run",
	Long:  "Listens for new transactions on a memo account. Prints out each new transaction as it comes in. ",
	Run: func(c *cobra.Command, args []string) {
		verbose, _ := c.Flags().GetBool(FlagVerbose)
		botKey, err := bot.GetKey(0)
		if err != nil {
			jerr.Get("fatal error getting bot key", err).Fatal()
		}
		botAddress := botKey.GetPublicKey().GetAddress().GetEncoded()
		scraper := twitterscraper.New()
		scraper.SetSearchMode(twitterscraper.SearchLatest)
		if err = scraper.Login(config.GetTwitterCreds().GetStrings()...); err != nil {
			jerr.Get("fatal error logging in to twitter", err).Fatal()
		}
		memoBot, err := bot.NewBot(scraper, []string{botAddress}, *botKey, verbose, false)
		if err != nil {
			jerr.Get("fatal error creating new bot", err).Fatal()
		}
		if err = memoBot.Run(); err != nil {
			jerr.Get("fatal error running memo bot", err).Fatal()
		} else {
			os.Exit(0)
		}
	},
}
