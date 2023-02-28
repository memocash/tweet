package maint

import "github.com/spf13/cobra"

const (
	FlagVerbose  = "verbose"
	FlagNoDryRun = "no-dry-run"
)

var maintCmd = &cobra.Command{
	Use: "maint",
}

func GetCommand() *cobra.Command {
	checkSavedTweetsCmd.Flags().BoolP(FlagVerbose, "v", false, "Verbose logging")
	checkSavedTweetCmd.Flags().BoolP(FlagVerbose, "v", false, "Verbose logging")
	fixSavedTweetCmd.Flags().BoolP(FlagNoDryRun, "", false, "No dry run")
	maintCmd.AddCommand(
		checkAddressSeenCmd,
		removeInvalidAddressSeenCmd,
		checkCompletedCmd,
		convertCompletedCmd,
		removeCompletedCmd,
		resetProfileCmd,
		checkSavedTweetsCmd,
		checkSavedTweetCmd,
		fixSavedTweetCmd,
	)
	return maintCmd
}
