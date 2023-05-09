package maint

import (
	twitterscraper "github.com/AbdelSallam/twitter-scraper"
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/config"
	"github.com/memocash/tweet/tweets"
	"github.com/spf13/cobra"
	"log"
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
		profile, err := tweets.GetProfile(USER_ID, scraper)
		if err != nil {
			jerr.Get("error getting profile", err).Fatal()
		}
		log.Printf("Name: %s\n Description: %s\n Profile Image: %s\n", profile.Name, profile.Description, profile.ProfilePic)
	},
}
