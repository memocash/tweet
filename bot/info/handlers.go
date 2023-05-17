package info

import (
	"encoding/json"
	"fmt"
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/db"
	"github.com/memocash/tweet/tweets"
	tweetWallet "github.com/memocash/tweet/wallet"
	"net/http"
	"strconv"
)

func (l *Server) balanceHandler(writer http.ResponseWriter, request *http.Request) {
	if err := request.ParseForm(); err != nil {
		_, err2 := writer.Write([]byte(fmt.Sprintf("error parsing form: %v", err)))
		if err2 != nil {
			l.ErrorChan <- jerr.Get("error writing response", err)
		}
		return
	}
	address := request.FormValue("address")
	addr, err := wallet.GetAddrFromString(address)
	if err != nil {
		_, err2 := writer.Write([]byte(fmt.Sprintf("error getting address; %v", err)))
		if err2 != nil {
			l.ErrorChan <- jerr.Get("error writing response", err)
		}
		return
	}
	walletDb := tweetWallet.Database{}
	utxos, err := walletDb.GetUtxos(*addr)
	var total int64
	for _, utxo := range utxos {
		_, err := writer.Write([]byte(fmt.Sprintf("utxo: %s:%d - %d\n", utxo.Hash, utxo.Index, utxo.Amount)))
		if err != nil {
			l.ErrorChan <- jerr.Get("error writing response", err)
		}
		total += utxo.Amount
	}
	_, err = writer.Write([]byte(fmt.Sprintf("balance: %d", total)))
	if err != nil {
		l.ErrorChan <- jerr.Get("error writing response", err)
	}
	return
}

func (l *Server) profileHandler(writer http.ResponseWriter, request *http.Request) {
	if err := request.ParseForm(); err != nil {
		_, err2 := writer.Write([]byte(fmt.Sprintf("error parsing form: %v", err)))
		if err2 != nil {
			l.ErrorChan <- jerr.Get("error writing response", err)
		}
		return
	}
	sender := request.FormValue("sender")
	userIdstr := request.FormValue("userId")
	_, err := writer.Write([]byte(fmt.Sprintf("Searching for profile-%s-%s\n", sender, userIdstr)))
	if err != nil {
		l.ErrorChan <- jerr.Get("error writing response", err)
	}
	userId, err := strconv.ParseInt(userIdstr, 10, 64)
	dbProfile, err := db.GetProfile(wallet.GetAddressFromString(sender).GetAddr(), userId)
	if err != nil {
		_, err2 := writer.Write([]byte(fmt.Sprintf("error getting profile; %v", err)))
		if err2 != nil {
			l.ErrorChan <- jerr.Get("error writing response", err)
		}
		return
	}
	var profile tweets.Profile
	err = json.Unmarshal(dbProfile.Profile, &profile)
	if err != nil {
		_, err2 := writer.Write([]byte(fmt.Sprintf("error unmarshalling profile; %v", err)))
		if err2 != nil {
			l.ErrorChan <- jerr.Get("error writing response", err)
		}
		return
	}
	_, err = writer.Write([]byte(fmt.Sprintf("name: %v\ndesc: %v\npicUrl: %v\n", profile.Name, profile.Description, profile.ProfilePic)))
	if err != nil {
		l.ErrorChan <- jerr.Get("error writing response", err)
	}
	return
}

func (l *Server) reportHandler(writer http.ResponseWriter, request *http.Request) {
	_, err := writer.Write([]byte("report"))
	if err != nil {
		l.ErrorChan <- jerr.Get("error writing response", err)
	}
	return
}
