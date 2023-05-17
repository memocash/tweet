package info

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/hasura/go-graphql-client"
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/index/client/lib"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/config"
	"github.com/memocash/tweet/db"
	ses "github.com/memocash/tweet/email"
	"github.com/memocash/tweet/graph"
	"github.com/memocash/tweet/tweets"
	tweetWallet "github.com/memocash/tweet/wallet"
	"html/template"
	"net/http"
	"os"
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
	var report TweetReport
	for _, addressKey := range addressKeys {
		botReport := l.CompileBotReport(addressKey, client, graphQlCLient)
		report.Bots = append(report.Bots, botReport)
	}
	templateText, err := os.ReadFile(config.GetConfig().TemplateDir + "/balance_report.html")
	if err != nil {
		l.ErrorChan <- fmt.Errorf("error reading balance report template; %w", err)
		return
	}
	tmpl, err := template.New("html").Parse(string(templateText))
	if err != nil {
		l.ErrorChan <- fmt.Errorf("error parsing balance report template; %w", err)
		return
	}
	var w = new(bytes.Buffer)
	if err := tmpl.Execute(w, report); err != nil {
		l.ErrorChan <- fmt.Errorf("error executing balance report template; %w", err)
		return
	}
	sender := ses.NewSender()
	if err := sender.Send(ses.Email{
		From:    config.GetConfig().AWS.FromEmail,
		To:      config.GetConfig().AWS.ToEmails,
		Subject: "Daily Twitter Bot Report",
		Body:    w.String(),
	}); err != nil {
		l.ErrorChan <- fmt.Errorf("error sending ses balance report email; %w", err)
		return
	}
	return
}

func (l *Server) CompileBotReport(addressKey *db.AddressLinkedKey, client *lib.Client, graphqlClient *graphql.Client) BotReport {
	decryptedKeyByte, err := tweetWallet.Decrypt(addressKey.Key, l.Bot.Crypt)
	if err != nil {
		l.ErrorChan <- jerr.Get("error decrypting key", err)
	}
	walletKey, err := wallet.ImportPrivateKey(string(decryptedKeyByte))
	if err != nil {
		l.ErrorChan <- jerr.Get("error importing key", err)
	}
	bal, err := client.GetBalance(walletKey.GetAddr())
	if err != nil {
		l.ErrorChan <- jerr.Get("error getting balance", err)
	}
	startTime := time.Now().Add(-time.Hour * 24)
	profiles, err := tweetWallet.GetProfile(walletKey.GetAddr().String(), startTime, graphqlClient)
	if err != nil {
		l.ErrorChan <- jerr.Get("error getting profile", err)
	}
	//get from graphQL query
	botProfile := profiles.Profiles[0]
	numSentPosts := 0
	numSentReplies := 0
	numFollowers := len(botProfile.Followers)
	numIncomingLikes := 0
	numIncomingReplies := 0
	profileUpdated := false
	createdAt := time.Now().Unix()
	var latestAction int64 = 0
	if botProfile.Name.Tx.Seen.GetTime().Unix() > startTime.Unix() ||
		botProfile.Profile.Tx.Seen.GetTime().Unix() > startTime.Unix() ||
		botProfile.Pic.Tx.Seen.GetTime().Unix() > startTime.Unix() {
		profileUpdated = true
	}
	for _, post := range botProfile.Posts {
		if post.Tx.Seen.GetTime().Unix() > latestAction {
			latestAction = post.Tx.Seen.GetTime().Unix()
		}
		if post.Tx.Seen.GetTime().Unix() < createdAt {
			createdAt = post.Tx.Seen.GetTime().Unix()
		}
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
	totalActions := numSentPosts + numSentReplies
	if profileUpdated {
		totalActions++
	}
	totalInteractions := numIncomingLikes + numIncomingReplies + numFollowers
	return BotReport{
		Name:               botProfile.Name.Name,
		Address:            walletKey.GetAddr().String(),
		ProfileLink:        MEMO_PROFILE_URL + walletKey.GetAddr().String(),
		Balance:            bal,
		NumSentPosts:       numSentPosts,
		NumSentReplies:     numSentReplies,
		NumFollowers:       numFollowers,
		NumIncomingLikes:   numIncomingLikes,
		NumIncomingReplies: numIncomingReplies,
		ProfileUpdated:     profileUpdated,
		TotalActions:       totalActions,
		TotalInteractions:  totalInteractions,
		CreatedAt:          time.Unix(createdAt, 0).String(),
		LatestAction:       time.Unix(latestAction, 0).String(),
		Owner:              wallet.GetAddrFromBytes(addressKey.Address[:]).String(),
	}
}
