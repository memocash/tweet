package cmd

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/cmd/getnewtweets"
	"github.com/memocash/tweet/cmd/memobot"
	"github.com/memocash/tweet/cmd/transfertweets"
	"github.com/memocash/tweet/cmd/updatename"
	"github.com/memocash/tweet/cmd/updateprofilepic"
	"github.com/memocash/tweet/cmd/updateprofiletext"
	"github.com/memocash/tweet/config"
	"github.com/spf13/cobra"
)

var pf interface {
	Stop()
}

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
		updatename.GetCommand(),
		updateprofiletext.GetCommand(),
		updateprofilepic.GetCommand(),
		getnewtweets.GetCommand(),
		memobot.GetCommand(),
	)
	if err := indexCmd.Execute(); err != nil {
		return jerr.Get("error executing server command", err)
	}
	return nil
}
