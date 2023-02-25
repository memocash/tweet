package tweets

import "github.com/spf13/cobra"

const (
	FlagLink = "link"
	FlagDate = "date"
)

var tweetsCmd = &cobra.Command{
	Use: "tweets",
}

func GetCommand() *cobra.Command {
	getNewCmd.Flags().BoolP(FlagLink, "l", false, "link to tweet")
	getNewCmd.Flags().BoolP(FlagDate, "d", false, "add date to post")
	transferCmd.Flags().BoolP(FlagLink, "l", false, "link to tweet")
	transferCmd.Flags().BoolP(FlagDate, "d", false, "add date to post")
	tweetsCmd.AddCommand(
		getNewCmd,
		transferCmd,
	)
	return tweetsCmd
}
