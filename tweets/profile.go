package tweets

import (
	"github.com/jchavannes/jgo/jerr"
	twitterscraper "github.com/n0madic/twitter-scraper"
	"net/http"
	"strconv"
	"strings"
)

type Profile struct {
	Name        string
	Description string
	ProfilePic  string
	ID          string
}

func GetProfile(userId int64, scraper *twitterscraper.Scraper) (*Profile, error) {
	var jsn user
	req, err := http.NewRequest("GET", "https://twitter.com/i/api/graphql/GazOglcBvgLigl3ywt6b3Q/UserByRestId?variables=%7B%22userId%22%3A%22"+strconv.FormatInt(userId, 10)+"%22%2C%22withSafetyModeUserFields%22%3Atrue%7D&features=%7B%22blue_business_profile_image_shape_enabled%22%3Atrue%2C%22responsive_web_graphql_exclude_directive_enabled%22%3Atrue%2C%22verified_phone_label_enabled%22%3Afalse%2C%22responsive_web_graphql_skip_user_profile_image_extensions_enabled%22%3Afalse%2C%22responsive_web_graphql_timeline_navigation_enabled%22%3Atrue%7D", nil)
	if err != nil {
		return nil, jerr.Get("error creating request", err)
	}
	err = scraper.RequestAPI(req, &jsn)
	if err != nil {
		return nil, jerr.Get("error requesting api", err)
	}
	name := jsn.Data.User.Result.Legacy.ScreenName
	description := jsn.Data.User.Result.Legacy.Description
	profilePic := strings.Replace(jsn.Data.User.Result.Legacy.ProfileImageURLHttps, "_normal", "", 1)
	return &Profile{
		Name:        name,
		Description: description,
		ProfilePic:  profilePic,
		ID:          strconv.FormatInt(userId, 10),
	}, nil
}

type user struct {
	Data struct {
		User struct {
			Result struct {
				RestID string `json:"rest_id"`
				Legacy struct {
					ScreenName           string `json:"screen_name"`
					ProfileImageURLHttps string `json:"profile_image_url_https"`
					Description          string `json:"description"`
				} `json:"legacy"`
			} `json:"result"`
		} `json:"user"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}
