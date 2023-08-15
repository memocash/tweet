package bot_report

import (
	"fmt"
	"github.com/hasura/go-graphql-client"
	"github.com/memocash/index/client/lib"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/config"
	"github.com/memocash/tweet/tweets"
	tweetWallet "github.com/memocash/tweet/wallet"
	twitterscraper "github.com/n0madic/twitter-scraper"
	"time"
)

type Bot struct {
	Owner   wallet.Addr
	Address wallet.Addr
	UserId  int64

	Name               string
	ProfileLink        string
	Balance            int64
	NumSentPosts       int
	NumFollowers       int
	NumIncomingLikes   int
	NumIncomingReplies int
	ProfileUpdated     bool
	TotalActions       int
	TotalInteractions  int
	CreatedAt          string
	LatestAction       string
}

func (b *Bot) SetInfo(client *lib.Client, graphqlClient *graphql.Client, scrapper *twitterscraper.Scraper) error {
	bal, err := client.GetBalance([]wallet.Addr{b.Address})
	if err != nil {
		return fmt.Errorf("error getting balance; %w", err)
	}
	startTime := time.Now().Add(-time.Hour * 24)
	profiles, err := tweetWallet.GetProfile(b.Address.String(), time.Time{}, graphqlClient)
	if err != nil {
		return fmt.Errorf("error getting profile; %w", err)
	}
	twitterProfile, err := tweets.GetProfile(b.UserId, scrapper)
	if err != nil {
		return fmt.Errorf("error getting twitter profile; %w", err)
	}
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
		return fmt.Errorf("error getting wallet addr from string; %w", err)
	}
	b.Name = twitterProfile.Name
	b.ProfileLink = config.GetMemoUrl(config.MemoProfileSuffix) + b.Address.String()
	b.Balance = bal.Balance
	b.NumSentPosts = numSentPosts
	b.NumFollowers = numFollowers
	b.NumIncomingLikes = numIncomingLikes
	b.NumIncomingReplies = numIncomingReplies
	b.ProfileUpdated = profileUpdated
	b.TotalActions = totalActions
	b.TotalInteractions = totalInteractions
	b.CreatedAt = time.Unix(createdAt, 0).String()
	b.LatestAction = time.Unix(latestAction, 0).String()
	return nil
}
