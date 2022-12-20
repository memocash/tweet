package tweets

import (
	"github.com/dghubble/go-twitter/twitter"
)

type TweetStreamData struct {
	Data struct {
		Attachments struct {
			MediaKeys []string `json:"media_keys"`
		} `json:"attachments"`
		Text             string `json:"text"`
		ID               string `json:"id"`
		CreatedAt        string `json:"created_at"`
		AuthorID         string `json:"author_id"`
		ReferencedTweets []struct {
			Type string `json:"type"`
			ID   string `json:"id"`
		} `json:"referenced_tweets"`
	} `json:"data"`
	Includes struct {
		Users []struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			Username string `json:"username"`
		} `json:"users"`
		Media []struct {
			MediaKey string `json:"media_key"`
			Type     string `json:"type"`
			URL      string `json:"url"`
		} `json:"media"`
	} `json:"includes"`
	MatchingRules []struct {
		ID  string `json:"id"`
		Tag string `json:"tag"`
	} `json:"matching_rules"`
}

type TweetTx struct {
	Tweet  *twitter.Tweet
	TxHash []byte
}

type IdInfo struct {
	ArchivedID int64
	NewestID   int64
}

type Archive struct {
	TweetList []TweetTx
	//number of tweets already archived
	Archived int
}
