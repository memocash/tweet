package db

import "github.com/spf13/cobra"

var dbCmd = &cobra.Command{
	Use: "db",
}

func GetCommand() *cobra.Command {
	dbCmd.AddCommand(
		outputsCmd,
	)
	return dbCmd
}
