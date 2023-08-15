package info

import (
	"encoding/json"
	"fmt"
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/bot/strm"
	"github.com/memocash/tweet/db"
	"github.com/memocash/tweet/email/bot_report"
	"github.com/memocash/tweet/tweets"
	tweetWallet "github.com/memocash/tweet/wallet"
	"log"
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
	utxos, err := walletDb.GetUtxos([]wallet.Addr{*addr})
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
	userIdStr := request.FormValue("userId")
	_, err := writer.Write([]byte(fmt.Sprintf("Searching for profile-%s-%s\n", sender, userIdStr)))
	if err != nil {
		l.ErrorChan <- jerr.Get("error writing response", err)
	}
	userId, err := strconv.ParseInt(userIdStr, 10, 64)
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

func (l *Server) reportHandler(writer http.ResponseWriter, _ *http.Request) {
	log.Println("Running Balance Report")
	streams, err := strm.GetStreams(true)
	if err != nil {
		_, err2 := writer.Write([]byte(fmt.Sprintf("error getting streams for bot report; %v", err)))
		if err2 != nil {
			l.ErrorChan <- jerr.Get("error writing bot report get streams error response", err2)
		}
		return
	}
	var reportBots []*bot_report.Bot
	for _, stream := range streams {
		reportBots = append(reportBots, &bot_report.Bot{
			Owner:   stream.Owner,
			Address: stream.Wallet.Address.GetAddr(),
			UserId:  stream.UserID,
		})
	}
	report := bot_report.New(reportBots)
	if err = report.Run(l.Scraper); err != nil {
		_, err2 := writer.Write([]byte(fmt.Sprintf("error running bot email report; %v", err)))
		if err2 != nil {
			l.ErrorChan <- jerr.Get("error writing bot report error response", err2)
		}
	}
	return
}
