package maint

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/config"
	"github.com/memocash/tweet/tweets"
	twitterscraper "github.com/n0madic/twitter-scraper"
	"github.com/spf13/cobra"
)

const (
	USER_ID = 1585391045935710208
)

var getProfileCmd = &cobra.Command{
	Use:   "get-profile",
	Short: "Testing getting profiles from user ids through the scraper",
	Run: func(c *cobra.Command, args []string) {
		scraper := twitterscraper.New()
		err := scraper.Login(config.GetTwitterAPIConfig().UserName, config.GetTwitterAPIConfig().Password)
		if err != nil {
			jerr.Get("error logging in", err).Fatal()
		}
		_, err = tweets.GetProfile(USER_ID, scraper)
		if err != nil {
			jerr.Get("error getting profile", err).Fatal()
		}
	},
}
