package bot

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/index/ref/bitcoin/wallet"
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
		botSeed := config.GetConfig().BotSeed
		mnemonic, err := wallet.GetMnemonicFromString(botSeed)
		if err != nil {
			jerr.Get("fatal error getting mnemonic from string", err).Fatal()
		}
		path := wallet.GetBip44Path(wallet.Bip44CoinTypeBTC, 0, false)
		botKey, err := mnemonic.GetPath(path)
		if err != nil {
			jerr.Get("fatal error getting path", err).Fatal()
		}
		botAddress := botKey.GetPublicKey().GetAddress().GetEncoded()
		scraper := twitterscraper.New()
		scraper.SetSearchMode(twitterscraper.SearchLatest)
		if err = scraper.Login(config.GetTwitterCreds().GetStrings()...); err != nil {
			jerr.Get("fatal error logging in to twitter", err).Fatal()
		}
		memoBot, err := bot.NewBot(mnemonic, scraper, []string{botAddress}, *botKey, verbose, false)
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
