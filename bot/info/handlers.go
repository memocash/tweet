package info

import (
	"encoding/json"
	"fmt"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/db"
	"github.com/memocash/tweet/tweets"
	tweetWallet "github.com/memocash/tweet/wallet"
	"net/http"
)

type Handler struct {
	Pattern string
	Handler func(writer http.ResponseWriter, request *http.Request)
}

var handlerBalance = Handler{
	Pattern: "/balance",
	Handler: func(writer http.ResponseWriter, request *http.Request) {
		if err := request.ParseForm(); err != nil {
			writer.Write([]byte(fmt.Sprintf("error parsing form: %v", err)))
			return
		}
		address := request.FormValue("address")
		addr, err := wallet.GetAddrFromString(address)
		if err != nil {
			writer.Write([]byte(fmt.Sprintf("error getting address; %v", err)))
			return
		}
		walletDb := tweetWallet.Database{}
		utxos, err := walletDb.GetUtxos(*addr)
		var total int64
		for _, utxo := range utxos {
			writer.Write([]byte(fmt.Sprintf("utxo: %s:%d - %d\n", utxo.Hash, utxo.Index, utxo.Amount)))
			total += utxo.Amount
		}
		writer.Write([]byte(fmt.Sprintf("balance: %d", total)))
	},
}

var handlerProfile = Handler{
	Pattern: "/profile",
	Handler: func(writer http.ResponseWriter, request *http.Request) {
		if err := request.ParseForm(); err != nil {
			writer.Write([]byte(fmt.Sprintf("error parsing form: %v", err)))
			return
		}
		sender := request.FormValue("sender")
		twittername := request.FormValue("twittername")
		writer.Write([]byte(fmt.Sprintf("Searching for profile-%s-%s\n", sender, twittername)))
		dbProfile, err := db.GetProfile(sender, twittername)
		if err != nil {
			writer.Write([]byte(fmt.Sprintf("error getting profile; %v", err)))
			return
		}
		var profile tweets.Profile
		json.Unmarshal(dbProfile.Profile, &profile)
		writer.Write([]byte(fmt.Sprintf("name: %v\ndesc: %v\npicUrl: %v\n", profile.Name, profile.Description, profile.ProfilePic)))
	},
}
