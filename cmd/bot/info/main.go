package info

import (
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use: "info",
}

func GetCommand() *cobra.Command {
	infoCmd.AddCommand(
		balanceCmd,
		profileCmd,
		reportCmd,
	)
	return infoCmd
}
