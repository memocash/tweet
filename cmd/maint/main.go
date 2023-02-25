package maint

import "github.com/spf13/cobra"

const (
	FlagVerbose = "verbose"
)

var maintCmd = &cobra.Command{
	Use: "maint",
}

func GetCommand() *cobra.Command {
	checkSavedTweetsCmd.Flags().BoolP(FlagVerbose, "v", false, "Verbose logging")
	maintCmd.AddCommand(
		checkAddressSeenCmd,
		removeInvalidAddressSeenCmd,
		checkCompletedCmd,
		convertCompletedCmd,
		removeCompletedCmd,
		resetProfileCmd,
		checkSavedTweetsCmd,
	)
	return maintCmd
}
