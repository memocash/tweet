package bot

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/bot"
	"github.com/memocash/tweet/bot/info"
	"github.com/memocash/tweet/config"
	"github.com/memocash/tweet/tweets"
	tweetWallet "github.com/memocash/tweet/wallet"
	twitterscraper "github.com/n0madic/twitter-scraper"
	"github.com/spf13/cobra"
	"log"
	"os"
	"os/signal"
	"syscall"
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
		if config.GetTwitterAPIConfig().Email == "" {
			err = scraper.Login(config.GetTwitterAPIConfig().UserName, config.GetTwitterAPIConfig().Password)
			if err != nil {
				jerr.Get("fatal error logging in to twitter", err).Fatal()
			}
		} else {
			err = scraper.Login(config.GetTwitterAPIConfig().UserName, config.GetTwitterAPIConfig().Password, config.GetTwitterAPIConfig().Email)
			if err != nil {
				jerr.Get("fatal error logging in to twitter", err).Fatal()
			}
		}
		memoBot, err := bot.NewBot(mnemonic, scraper, []string{botAddress}, *botKey, verbose, false)
		if err != nil {
			jerr.Get("fatal error creating new bot", err).Fatal()
		}
		cryptBytes, err := tweetWallet.GenerateEncryptionKeyFromPassword(config.GetConfig().BotCrypt)
		if err != nil {
			jerr.Get("fatal error generating encryption key", err).Fatal()
		}
		memoBot.Crypt = cryptBytes
		if err := memoBot.ProcessMissedTxs(); err != nil {
			jerr.Get("fatal error updating bot", err).Fatal()
		}
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigc
			log.Println("Tweet Bot caught SIGINT, saving cookies and stopping...")
			err := tweets.SaveCookies(memoBot.TweetScraper.GetCookies())
			if err != nil {
				jerr.Get("error saving cookies", err).Print()
			}
			os.Exit(0)
		}()
		var errorChan = make(chan error)
		go func() {
			err = memoBot.Listen()
			logoutError := memoBot.TweetScraper.Logout()
			if logoutError != nil {
				jerr.Get("error logging out", err).Print()
			}
			cookieError := tweets.SaveCookies(memoBot.TweetScraper.GetCookies())
			if cookieError != nil {
				jerr.Get("error saving cookies", err).Print()
			}
			errorChan <- jerr.Get("error listening for transactions", err)
		}()
		go func() {
			infoServer := info.NewServer(memoBot)
			err = infoServer.Listen()
			errorChan <- jerr.Get("error info server listener", err)
		}()
		jerr.Get("fatal error running memo bot", <-errorChan).Fatal()
	},
}
