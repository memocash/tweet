package bot

import (
	"github.com/spf13/cobra"
)

var botCmd = &cobra.Command{
	Use: "bot",
}

func GetCommand() *cobra.Command {
	botCmd.AddCommand(
		runCmd,
		infoCmd,
	)
	return botCmd
}
