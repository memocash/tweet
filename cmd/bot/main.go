package bot

import (
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
		infoCmd,
	)
	return botCmd
}
