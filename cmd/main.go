package cmd

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/cmd/bot"
	"github.com/memocash/tweet/cmd/getnewtweets"
	"github.com/memocash/tweet/cmd/maint"
	"github.com/memocash/tweet/cmd/test"
	"github.com/memocash/tweet/cmd/transfertweets"
	"github.com/memocash/tweet/cmd/update"
	"github.com/memocash/tweet/config"
	"github.com/spf13/cobra"
)

var indexCmd = &cobra.Command{
	Use:   "memotweet",
	Short: "Twitter Content -> Memo Content",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if err := config.InitConfig(); err != nil {
			jerr.Get("error initializing config", err).Fatal()
		}
	},
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
}

func Execute() error {
	indexCmd.AddCommand(
		transfertweets.GetCommand(),
		update.GetCommand(),
		getnewtweets.GetCommand(),
		bot.GetCommand(),
		test.GetCommand(),
		maint.GetCommand(),
	)
	if err := indexCmd.Execute(); err != nil {
		return jerr.Get("error executing server command", err)
	}
	return nil
}
