package main

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/index/ref/bitcoin/util/testing/test_tx"
	"github.com/memocash/tweet/database"
	"github.com/memocash/tweet/tweets"
	"os"
)

func main() {
	args := os.Args[1:]
	command := args[0]
	key := test_tx.GetPrivateKey(args[1])
	account := args[2]
	address := key.GetAddress()
	if command == "TransferTweets"{
		tweets := tweets.GetTweets(account)
		err := database.TransferTweets(address,key,tweets)
		if err != nil{
			jerr.Get("error", err).Fatal()
		}
	}
	if command == "UpdateName"{
		name,_,_ := tweets.GetProfile(account)
		err := database.UpdateName(address,key,name)
		if err != nil{
			jerr.Get("error", err).Fatal()
		}
	}
	if command == "UpdateProfileText"{
		_,desc,_ := tweets.GetProfile(account)
		err := database.UpdateProfileText(address,key,desc)
		if err != nil{
			jerr.Get("error", err).Fatal()
		}
	}
	if command == "UpdateProfilePic"{
		_,_,url := tweets.GetProfile(account)
		err := database.UpdateProfilePic(address,key,url)
		if err != nil{
			jerr.Get("error", err).Fatal()
		}
	}
}
