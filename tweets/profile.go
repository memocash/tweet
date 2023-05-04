package tweets

import (
	"context"
	"fmt"
	"github.com/jchavannes/jgo/jerr"
	twitterscraper "github.com/n0madic/twitter-scraper"
	"log"
	"strings"
)

type Profile struct {
	Name        string
	Description string
	ProfilePic  string
	ID          string
}

func GetProfile(userId int64, scraper *twitterscraper.Scraper) (*Profile, error) {
	query := fmt.Sprintf("user_id:%d", userId)
	println("query: " + query)
	for profile := range scraper.SearchProfiles(context.Background(), query, 1) {
		println("got a profile")
		if profile == nil {
			println("nil profile")
			return nil, jerr.New("nil profile")
		}
		if profile.Error != nil {
			return nil, jerr.Get("error getting profile", profile.Error)
		}
		log.Printf("profile error: %s\n", profile.Error)
		profilePic := strings.Replace(profile.Avatar, "_normal", "", 1)
		profilePic = strings.Replace(profilePic, "http:", "https:", 1)
		return &Profile{
			Name:        profile.Name,
			Description: profile.Biography,
			ProfilePic:  profilePic,
			ID:          profile.UserID,
		}, nil
	}
	return nil, jerr.Get("no profile found", nil)
}
