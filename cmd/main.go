package cmd

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/cmd/transfertweets"
	"github.com/memocash/tweet/cmd/updatename"
	"github.com/memocash/tweet/cmd/updateprofilepic"
	"github.com/memocash/tweet/cmd/updateprofiletext"
	"github.com/spf13/cobra"
)

var pf interface {
	Stop()
}

var indexCmd = &cobra.Command{
	Use:   "memotweet",
	Short: "Twitter Content -> Memo Content",
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
	)
	if err := indexCmd.Execute(); err != nil {
		return jerr.Get("error executing server command", err)
	}
	return nil
}

