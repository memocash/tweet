package info

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/hasura/go-graphql-client"
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/index/client/lib"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/bot"
	"github.com/memocash/tweet/config"
	"github.com/memocash/tweet/db"
	ses "github.com/memocash/tweet/email"
	"github.com/memocash/tweet/graph"
	"github.com/memocash/tweet/tweets"
	tweetWallet "github.com/memocash/tweet/wallet"
	"html/template"
	"log"
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
	graphQlClient := graphql.NewClient(graph.ServerUrlHttp, nil)
	client := lib.NewClient(graph.ServerUrlHttp, &tweetWallet.Database{})
	streams, err := bot.GetStreams(true)
	if err != nil {
		_, err2 := writer.Write([]byte(fmt.Sprintf("error getting address keys; %v", err)))
		if err2 != nil {
			l.ErrorChan <- jerr.Get("error writing response", err)
		}
		return
	}
	var report TweetReport
	for _, stream := range streams {
		botReport := l.CompileBotReport(&stream, client, graphQlClient)
		report.Bots = append(report.Bots, botReport)
		_, err := writer.Write([]byte(fmt.Sprintf("%s\n", botReport.String())))
		if err != nil {
			l.ErrorChan <- jerr.Get("error writing response", err)
		}
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

func (l *Server) CompileBotReport(stream *bot.Stream, client *lib.Client, graphqlClient *graphql.Client) BotReport {
	bal, err := client.GetBalance([]wallet.Addr{stream.Wallet.Key.GetAddr()})
	if err != nil {
		l.ErrorChan <- jerr.Get("error getting balance", err)
	}
	startTime := time.Now().Add(-time.Hour * 24)
	profiles, err := tweetWallet.GetProfile(stream.Wallet.Address.GetEncoded(), time.Time{}, graphqlClient)
	if err != nil {
		l.ErrorChan <- jerr.Get("error getting profile", err)
	}
	twitterProfile, err := tweets.GetProfile(stream.UserID, l.Bot.TweetScraper)
	if err != nil {
		l.ErrorChan <- jerr.Get("error getting twitter profile", err)
	}
	//get from graphQL query
	botProfile := profiles.Profiles[0]
	numSentPosts := 0
	numFollowers := 0
	numIncomingLikes := 0
	numIncomingReplies := 0
	profileUpdated := false
	createdAt := time.Now().Unix()
	var latestAction int64 = 0
	for _, post := range botProfile.Posts {
		if post.Tx.Seen.GetTime().Unix() > startTime.Unix() {
			numSentPosts++
		}
		if post.Tx.Seen.GetTime().Unix() > latestAction {
			latestAction = post.Tx.Seen.GetTime().Unix()
		}
		if post.Tx.Seen.GetTime().Unix() < createdAt {
			createdAt = post.Tx.Seen.GetTime().Unix()
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
	for _, follower := range botProfile.Followers {
		if follower.Tx.Seen.GetTime().Unix() > startTime.Unix() {
			numFollowers++
		}
	}
	if botProfile.Name.Tx.Seen.GetTime().Unix() > latestAction {
		latestAction = botProfile.Name.Tx.Seen.GetTime().Unix()
	} else if botProfile.Profile.Tx.Seen.GetTime().Unix() > latestAction {
		latestAction = botProfile.Profile.Tx.Seen.GetTime().Unix()
	} else if botProfile.Pic.Tx.Seen.GetTime().Unix() > latestAction {
		latestAction = botProfile.Pic.Tx.Seen.GetTime().Unix()
	}
	totalActions := numSentPosts
	if botProfile.Name.Tx.Seen.GetTime().Unix() > startTime.Unix() {
		profileUpdated = true
		totalActions++
	}
	if botProfile.Profile.Tx.Seen.GetTime().Unix() > startTime.Unix() {
		profileUpdated = true
		totalActions++
	}
	if botProfile.Pic.Tx.Seen.GetTime().Unix() > startTime.Unix() {
		profileUpdated = true
		totalActions++
	}
	totalInteractions := numIncomingLikes + numIncomingReplies + numFollowers
	if err != nil {
		l.ErrorChan <- jerr.Get("error getting wallet addr from string", err)
	}
	return BotReport{
		Name:               twitterProfile.Name,
		Address:            stream.Wallet.Address.GetEncoded(),
		ProfileLink:        MEMO_PROFILE_URL + stream.Wallet.Address.GetEncoded(),
		Balance:            bal.Balance,
		NumSentPosts:       numSentPosts,
		NumFollowers:       numFollowers,
		NumIncomingLikes:   numIncomingLikes,
		NumIncomingReplies: numIncomingReplies,
		ProfileUpdated:     profileUpdated,
		TotalActions:       totalActions,
		TotalInteractions:  totalInteractions,
		CreatedAt:          time.Unix(createdAt, 0).String(),
		LatestAction:       time.Unix(latestAction, 0).String(),
		Owner:              stream.Owner.String(),
	}
}
