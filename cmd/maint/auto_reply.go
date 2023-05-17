package maint

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/bot"
	"github.com/memocash/tweet/bot/info"
	"github.com/memocash/tweet/config"
	tweetWallet "github.com/memocash/tweet/wallet"
	"github.com/spf13/cobra"
)

var autoReplyCmd = &cobra.Command{
	Use:   "auto-reply",
	Short: "auto-reply",
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
		memoBot, err := bot.NewBot(mnemonic, nil, []string{botAddress}, *botKey, verbose, true)
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
			err = memoBot.MaintenanceListen()
			errorChan <- jerr.Get("error listening for transactions while under maintenance", err)
		}()
		go func() {
			infoServer := info.NewServer(memoBot)
			err = infoServer.Listen()
			errorChan <- jerr.Get("error info server listener", err)
		}()
		jerr.Get("fatal error running memo bot", <-errorChan).Fatal()
	},
}
