package maint

import "github.com/spf13/cobra"

var maintCmd = &cobra.Command{
	Use: "maint",
}

func GetCommand() *cobra.Command {
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
