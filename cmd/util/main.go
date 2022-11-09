package util

import (
	"github.com/dghubble/go-twitter/twitter"
	"github.com/memocash/index/ref/bitcoin/util/testing/test_tx"
	"github.com/memocash/index/ref/bitcoin/wallet"
)

func Setup(args []string) (wallet.PrivateKey,wallet.Address,string){
	key := test_tx.GetPrivateKey(args[0])
	address := key.GetAddress()
	account := args[1]
	return key,address,account
}

type TweetObject struct {
	TweetList []twitter.Tweet
	//number of tweets already archived
	Archived  int
}
