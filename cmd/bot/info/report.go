package info

import (
	"github.com/memocash/tweet/bot/info"
	"github.com/spf13/cobra"
	"log"
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "report",
	Run: func(c *cobra.Command, args []string) {
		if len(args) != 0 {
			log.Fatalf("report takes no arguments")
		}
		log.Println("starting tweet info report")
		if err := info.Report(); err != nil {
			log.Fatalf("error info report; %v", err)
		}
		log.Println("finished tweet info report")
	},
}
