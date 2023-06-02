package bot

import (
	"github.com/memocash/tweet/cmd/bot/info"
	"github.com/spf13/cobra"
)

const (
	FlagVerbose = "verbose"
)

var botCmd = &cobra.Command{
	Use: "bot",
}

func GetCommand() *cobra.Command {
	runCmd.Flags().BoolP(FlagVerbose, "v", false, "Verbose logging")
	botCmd.AddCommand(
		runCmd,
		info.GetCommand(),
	)
	return botCmd
}
