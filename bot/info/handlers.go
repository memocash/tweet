package info

import (
	"encoding/json"
	"fmt"
	"github.com/hasura/go-graphql-client"
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/index/client/lib"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/db"
	"github.com/memocash/tweet/graph"
	"github.com/memocash/tweet/tweets"
	tweetWallet "github.com/memocash/tweet/wallet"
	"net/http"
	"strconv"
	"time"
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
	graphQlCLient := graphql.NewClient(graph.ServerUrlHttp, nil)
	client := lib.NewClient(graph.ServerUrlHttp, &tweetWallet.Database{})
	addressKeys, err := db.GetAllAddressKey()
	if err != nil {
		_, err2 := writer.Write([]byte(fmt.Sprintf("error getting address keys; %v", err)))
		if err2 != nil {
			l.ErrorChan <- jerr.Get("error writing response", err)
		}
		return
	}
	//addressKey.Address is the senderAddress
	//addressKey.Key is the private key of the bot itself
	for _, addressKey := range addressKeys {
		decryptedKeyByte, err := tweetWallet.Decrypt(addressKey.Key, l.Bot.Crypt)
		if err != nil {
			_, err2 := writer.Write([]byte(fmt.Sprintf("error decrypting key; %v", err)))
			if err2 != nil {
				l.ErrorChan <- jerr.Get("error writing response", err)
			}
		}
		walletKey, err := wallet.ImportPrivateKey(string(decryptedKeyByte))
		if err != nil {
			_, err2 := writer.Write([]byte(fmt.Sprintf("error importing key; %v", err)))
			if err2 != nil {
				l.ErrorChan <- jerr.Get("error writing response", err)
			}
		}
		bal, err := client.GetBalance(walletKey.GetAddr())
		if err != nil {
			_, err2 := writer.Write([]byte(fmt.Sprintf("error getting balance; %v", err)))
			if err2 != nil {
				l.ErrorChan <- jerr.Get("error writing response", err)
			}
			return
		}
		startTime := time.Now().Add(-time.Hour * 24)
		profiles, err := tweetWallet.GetProfile(walletKey.GetAddr().String(), startTime, graphQlCLient)
		if err != nil {
			_, err2 := writer.Write([]byte(fmt.Sprintf("error getting profile; %v", err)))
			if err2 != nil {
				l.ErrorChan <- jerr.Get("error writing response", err)
			}
			return
		}
		//get from graphQL query
		botProfile := profiles.Profiles[0]
		numSentPosts := 0
		numSentReplies := 0
		numFollowers := len(botProfile.Followers)
		numIncomingLikes := 0
		//get from database
		numIncomingReplies := 0
		profileUpdated := false
		if botProfile.Name.Tx.Seen.GetTime().Unix() > startTime.Unix() ||
			botProfile.Profile.Tx.Seen.GetTime().Unix() > startTime.Unix() ||
			botProfile.Pic.Tx.Seen.GetTime().Unix() > startTime.Unix() {
			profileUpdated = true
		}
		for _, post := range botProfile.Posts {
			if post.Tx.Seen.GetTime().Unix() > startTime.Unix() {
				if post.Parent.Address == "" {
					numSentPosts++
				} else {
					numSentReplies++
				}
			}
			for _, like := range post.Likes {
				if like.Tx.Seen.GetTime().Unix() > startTime.Unix() {
					numIncomingLikes++
				}
			}
			for _, reply := range post.Replies {
				if reply.Tx.Seen.GetTime().Unix() > startTime.Unix() {
					numIncomingReplies++
				}
			}
		}

		_, err = writer.Write([]byte(fmt.Sprintf("address: %s\n balance: %d\n numSentPosts: %d\n"+
			"numSentReplies: %d\n numFollowers: %d\n numIncomingLikes: %d\n numIncomingReplies: %d\n profileUpdated: %v\n",
			walletKey.GetAddr().String(), bal, numSentPosts, numSentReplies, numFollowers,
			numIncomingLikes, numIncomingReplies, profileUpdated)))
		if err != nil {
			l.ErrorChan <- jerr.Get("error writing response", err)
		}
	}
	return
}
