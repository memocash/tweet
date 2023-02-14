package bot

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/bot"
	"github.com/memocash/tweet/config"
	"github.com/memocash/tweet/database"
	"github.com/memocash/tweet/tweets"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "run",
	Long:  "Listens for new transactions on a memo account. Prints out each new transaction as it comes in. ",
	RunE: func(c *cobra.Command, args []string) error {
		botSeed := config.GetConfig().BotSeed
		//get base key and address from seed
		mnemonic, err := wallet.GetMnemonicFromString(botSeed)
		if err != nil {
			return jerr.Get("error getting mnemonic from string", err)
		}
		path := wallet.GetBip44Path(wallet.Bip44CoinTypeBTC, 0, false)
		botKey, err := mnemonic.GetPath(path)
		if err != nil {
			return jerr.Get("error getting path", err)
		}
		db, err := database.GetDb()
		if err != nil {
			return jerr.Get("error opening db", err)
		}
		//check that memobot-num-streams field of the database exists
		fieldExists, err := db.Has([]byte("memobot-num-streams"), nil)
		if err != nil {
			return jerr.Get("error checking if memobot-num-streams field exists", err)
		}
		//if it doesn't, create it and set it to 0
		if !fieldExists {
			err = db.Put([]byte("memobot-num-streams"), []byte("0"), nil)
			if err != nil {
				return jerr.Get("error creating memobot-num-streams field", err)
			}
		}
		fieldExists, err = db.Has([]byte("memobot-running-count"), nil)
		if err != nil {
			return jerr.Get("error checking if memobot-running-count field exists", err)
		}
		//if it doesn't, create it and set it to 0
		if !fieldExists {
			err = db.Put([]byte("memobot-running-count"), []byte("0"), nil)
			if err != nil {
				return jerr.Get("error creating memobot-num-streams field", err)
			}
		}
		botAddress := botKey.GetPublicKey().GetAddress().GetEncoded()
		memoBot, err := bot.NewBot(mnemonic, []string{botAddress}, *botKey, tweets.Connect(), db)
		if err != nil {
			return jerr.Get("error creating new bot", err)
		}
		cryptBytes, err := database.GenerateEncryptionKeyFromPassword(config.GetConfig().BotCrypt)
		if err != nil {
			return jerr.Get("error generating encryption key", err)
		}
		memoBot.Crypt = string(cryptBytes)
		if err := memoBot.ProcessMissedTxs(); err != nil {
			return jerr.Get("error updating bot", err)
		}
		var errorChan = make(chan error)
		go func() {
			err = memoBot.Listen()
			errorChan <- jerr.Get("error listening for transactions", err)
		}()
		go func() {
			infoServer := bot.NewInfoServer(db)
			err = infoServer.Listen()
			errorChan <- jerr.Get("error info server listener", err)
		}()
		return <-errorChan
	},
}
