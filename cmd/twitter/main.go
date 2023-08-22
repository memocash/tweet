package twitter

import "github.com/spf13/cobra"

var twitterCmd = &cobra.Command{
	Use: "twitter",
}

func GetCommand() *cobra.Command {
	twitterCmd.AddCommand(
		tweetCmd,
		tweetByGobCmd,
	)
	return twitterCmd
}
