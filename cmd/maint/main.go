package maint

import "github.com/spf13/cobra"

var maintCmd = &cobra.Command{
	Use: "maint",
}

func GetCommand() *cobra.Command {
	maintCmd.AddCommand(
		checkCompletedCmd,
		removeCompletedCmd,
		resetProfileCmd,
	)
	return maintCmd
}
