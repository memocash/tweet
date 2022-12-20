package tweets

import (
	"github.com/dghubble/go-twitter/twitter"
	"github.com/jchavannes/jgo/jerr"
	"strings"
)

type Profile struct {
	Name        string
	Description string
	ProfilePic  string
	ID          string
}

func GetProfile(screenName string, client *twitter.Client) (*Profile, error) {
	// Query to Twitter API for profile info
	// user show
	userShowParams := &twitter.UserShowParams{ScreenName: screenName}
	user, _, err := client.Users.Show(userShowParams)
	if err != nil {
		return nil, jerr.Get("error getting profile from twitter api", err)
	}
	desc := user.Description
	name := user.Name
	profilePic := user.ProfileImageURL
	ID := user.IDStr
	//resize the profile pic to full size
	profilePic = strings.Replace(profilePic, "_normal", "", 1)
	profilePic = strings.Replace(profilePic, "http:", "https:", 1)
	//println(profilePic)
	//fmt.Printf("USERS SHOW:\n%+v\n%+v\n%+v\n", name, desc, profilePic)
	return &Profile{
		Name:        name,
		Description: desc,
		ProfilePic:  profilePic,
		ID:          ID,
	}, nil
}
