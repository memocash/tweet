package cmd

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/cmd/bot"
	"github.com/memocash/tweet/cmd/maint"
	"github.com/memocash/tweet/cmd/update"
	"github.com/memocash/tweet/config"
	"github.com/spf13/cobra"
	"log"
	"os"
)

var tweetCmd = &cobra.Command{
	Use:   "tweet",
	Short: "Twitter Content -> Memo Content",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if err := config.InitConfig(); err != nil {
			jerr.Get("error initializing config", err).Fatal()
		}
		log.SetOutput(os.Stdout)
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
