package util

import (
	"github.com/dghubble/go-twitter/twitter"
	"github.com/jchavannes/jgo/jlog"
	"github.com/memocash/index/ref/bitcoin/wallet"
)

func Setup(args []string) (wallet.PrivateKey, wallet.Address, string) {
	key, _ := wallet.ImportPrivateKey(args[0])
	address := key.GetAddress()
	jlog.Logf("Using address: %s\n", address.GetEncoded())
	account := args[1]
	return key, address, account
}

type TweetTx struct {
	Tweet  *twitter.Tweet
	TxHash []byte
}

type Archive struct {
	TweetList []TweetTx
	//number of tweets already archived
	Archived int
}
