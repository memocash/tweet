package tweets

import (
	"fmt"
	"github.com/jchavannes/jgo/jerr"
	twitterscraper "github.com/n0madic/twitter-scraper"
	"golang.org/x/net/html"
	"log"
	"net/http"
)

type Profile struct {
	Name        string
	Description string
	ProfilePic  string
	ID          string
}

func GetProfile(userId int64, scraper *twitterscraper.Scraper) (*Profile, error) {
	username, err := ConvertUserIDToUsername(userId)
	if err != nil {
		return nil, jerr.Get("error converting user id to username", err)
	}
	log.Printf("Username: %s\n", username)

	//scraper.SetSearchMode(twitterscraper.SearchUsers)
	//query := fmt.Sprintf("%d", userId)
	//for profile := range scraper.SearchProfiles(context.Background(), query, 1) {
	//	if profile == nil {
	//		return nil, jerr.New("nil profile")
	//	}
	//	if profile.Error != nil {
	//		return nil, jerr.Get("error getting profile", profile.Error)
	//	}
	//	profilePic := strings.Replace(profile.Avatar, "_normal", "", 1)
	//	profilePic = strings.Replace(profilePic, "http:", "https:", 1)
	//	scraper.SetSearchMode(twitterscraper.SearchLatest)
	//	result := &Profile{
	//		Name:        profile.Name,
	//		Description: profile.Biography,
	//		ProfilePic:  profilePic,
	//		ID:          profile.UserID,
	//	}
	//	log.Printf("Name: %s\nDescription: %s\nProfile Pic: %s\n", result.Name, result.Description, result.ProfilePic)
	//}
	//scraper.SetSearchMode(twitterscraper.SearchLatest)
	//log.Println("done searching profiles")
	return nil, nil
	//return nil, jerr.Get("no profile found", nil)
}

func ConvertUserIDToUsername(userId int64) (string, error) {
	//directly make a request to https://twitter.com/intent/user?user_id=userId
	//and parse the username from the html
	url := fmt.Sprintf("https://twitter.com/intent/user?user_id=%d", userId)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", jerr.Get("error getting new request", err)
	}
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", jerr.Get("error doing default client", err)
	}
	defer response.Body.Close()
	doc, err := html.Parse(response.Body)
	if err != nil {
		return "", jerr.Get("error parsing html", err)
	}
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "span" {
			log.Printf("span tag: %s\n", n.FirstChild.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	return "", nil
}
