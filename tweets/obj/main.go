package obj

import (
	"github.com/dghubble/go-twitter/twitter"
)

type TweetTx struct {
	Tweet  *twitter.Tweet
	TxHash []byte
}
