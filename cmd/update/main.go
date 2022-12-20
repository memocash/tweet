package update

import (
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use: "update",
}

func GetCommand() *cobra.Command {
	updateCmd.AddCommand(
		nameCmd,
		profileCmd,
		profilePicCmd,
	)
	return updateCmd
}
