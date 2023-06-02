package cmd

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/cmd/bot"
	"github.com/memocash/tweet/cmd/maint"
	"github.com/memocash/tweet/cmd/update"
	"github.com/memocash/tweet/config"
	tweetWallet "github.com/memocash/tweet/wallet"
	"github.com/spf13/cobra"
	"log"
	"os"
)

var tweetCmd = &cobra.Command{
	Use:   "tweet",
	Short: "Twitter Content -> Memo Content",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		log.SetOutput(os.Stdout)
		if err := config.InitConfig(); err != nil {
			log.Fatalf("fatal error initializing config; %v", err)
		}
		dbEncryptionKey, err := tweetWallet.GenerateEncryptionKeyFromPassword(config.GetDbEncryptionKey())
		if err != nil {
			log.Fatalf("fatal error generating db encryption key; %v", err)
		}
		tweetWallet.SetDbEncryptionKey(dbEncryptionKey)
	},
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
}

func Execute() error {
	tweetCmd.AddCommand(
		bot.GetCommand(),
		maint.GetCommand(),
		update.GetCommand(),
	)
	if err := tweetCmd.Execute(); err != nil {
		return jerr.Get("error executing tweet command", err)
	}
	return nil
}
