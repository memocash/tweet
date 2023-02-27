package bot

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/bot"
	"github.com/memocash/tweet/bot/info"
	"github.com/memocash/tweet/config"
	"github.com/memocash/tweet/db"
	"github.com/memocash/tweet/tweets"
	tweetWallet "github.com/memocash/tweet/wallet"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "run",
	Long:  "Listens for new transactions on a memo account. Prints out each new transaction as it comes in. ",
	Run: func(c *cobra.Command, args []string) {
		verbose, _ := c.Flags().GetBool(FlagVerbose)
		botSeed := config.GetConfig().BotSeed
		//get base key and address from seed
		mnemonic, err := wallet.GetMnemonicFromString(botSeed)
		if err != nil {
			jerr.Get("fatal error getting mnemonic from string", err).Fatal()
		}
		path := wallet.GetBip44Path(wallet.Bip44CoinTypeBTC, 0, false)
		botKey, err := mnemonic.GetPath(path)
		if err != nil {
			jerr.Get("fatal error getting path", err).Fatal()
		}
		db, err := db.GetDb()
		if err != nil {
			jerr.Get("fatal error opening db", err).Fatal()
		}
		//check that memobot-num-streams field of the database exists
		fieldExists, err := db.Has([]byte("memobot-num-streams"), nil)
		if err != nil {
			jerr.Get("fatal error checking if memobot-num-streams field exists", err).Fatal()
		}
		//if it doesn't, create it and set it to 0
		if !fieldExists {
			err = db.Put([]byte("memobot-num-streams"), []byte("0"), nil)
			if err != nil {
				jerr.Get("fatal error creating memobot-num-streams field", err).Fatal()
			}
		}
		fieldExists, err = db.Has([]byte("memobot-running-count"), nil)
		if err != nil {
			jerr.Get("fatal error checking if memobot-running-count field exists", err).Fatal()
		}
		//if it doesn't, create it and set it to 0
		if !fieldExists {
			err = db.Put([]byte("memobot-running-count"), []byte("0"), nil)
			if err != nil {
				jerr.Get("fatal error creating memobot-num-streams field", err).Fatal()
			}
		}
		botAddress := botKey.GetPublicKey().GetAddress().GetEncoded()
		memoBot, err := bot.NewBot(mnemonic, []string{botAddress}, *botKey, tweets.Connect(), db, verbose)
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
		var errorChan = make(chan error)
		go func() {
			err = memoBot.Listen()
			errorChan <- jerr.Get("error listening for transactions", err)
		}()
		go func() {
			infoServer := info.NewServer()
			err = infoServer.Listen()
			errorChan <- jerr.Get("error info server listener", err)
		}()
		jerr.Get("fatal error running memo bot", <-errorChan).Fatal()
	},
}
