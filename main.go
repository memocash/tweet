package main

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/index/ref/bitcoin/util/testing/test_tx"
	"github.com/memocash/tweet/database"
	"github.com/memocash/tweet/tweets"
)

func main() {
	address := test_tx.Address3
	key := test_tx.Address3key
	tweets := tweets.GetTweets("MemoCashAbdel")
	err := database.TransferTweets(address,key,tweets)
	if err != nil{
		jerr.Get("error", err).Fatal()
	}
}
