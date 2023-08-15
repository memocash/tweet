package db

import (
	"github.com/memocash/tweet/db"
	"github.com/spf13/cobra"
	"log"
)

var deleteCookiesCmd = &cobra.Command{
	Use: "delete-cookies",
	Run: func(c *cobra.Command, args []string) {
		if err := db.Delete([]db.ObjectI{&db.Cookies{}}); err != nil {
			log.Fatalf("error deleting cookies; %v", err)
		}
		log.Printf("deleted cookies\n")
	},
}
