package maint

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/bot"
	"github.com/memocash/tweet/bot/info"
	"github.com/spf13/cobra"
)

var autoReplyCmd = &cobra.Command{
	Use:   "auto-reply",
	Short: "auto-reply",
	Run: func(c *cobra.Command, args []string) {
		verbose, _ := c.Flags().GetBool(FlagVerbose)
		botKey, err := bot.GetKey(0)
		if err != nil {
			jerr.Get("fatal error getting bot key", err).Fatal()
		}
		botAddress := botKey.GetPublicKey().GetAddress().GetEncoded()
		memoBot, err := bot.NewBot(nil, []string{botAddress}, *botKey, verbose, true)
		if err != nil {
			jerr.Get("fatal error creating new bot", err).Fatal()
		}
		if err := memoBot.ProcessMissedTxs(); err != nil {
			jerr.Get("fatal error updating bot", err).Fatal()
		}
		var errorChan = make(chan error)
		go func() {
			err = memoBot.MaintenanceListen()
			errorChan <- jerr.Get("error listening for transactions while under maintenance", err)
		}()
		go func() {
			infoServer := info.NewServer(memoBot.TweetScraper)
			err = infoServer.Listen()
			errorChan <- jerr.Get("error info server listener", err)
		}()
		jerr.Get("fatal error running memo bot", <-errorChan).Fatal()
	},
}
