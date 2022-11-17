package util

import (
	"github.com/dghubble/go-twitter/twitter"
	"github.com/jchavannes/jgo/jlog"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"time"
)

func Setup(args []string) (wallet.PrivateKey, wallet.Address, string) {
	key, _ := wallet.ImportPrivateKey(args[0])
	address := key.GetAddress()
	jlog.Logf("Using address: %s\n", address.GetEncoded())
	account := args[1]
	return key, address, account
}
type StreamDataExample struct {
	Data struct {
		Text      string    `json:"text"`
		ID        string    `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		AuthorID  string    `json:"author_id"`
	} `json:"data"`
	Includes struct {
		Users []struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			Username string `json:"username"`
		} `json:"users"`
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
