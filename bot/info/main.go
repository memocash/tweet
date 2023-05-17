package info

import "fmt"

const (
	MEMO_PROFILE_URL = "https://memo.cash/profile/"
)

type TweetReport struct {
	Bots []BotReport
}

type BotReport struct {
	Name               string
	Address            string
	ProfileLink        string
	Balance            int64
	NumSentPosts       int
	NumSentReplies     int
	NumFollowers       int
	NumIncomingLikes   int
	NumIncomingReplies int
	ProfileUpdated     bool
	TotalActions       int
	TotalInteractions  int
	CreatedAt          string
	LatestAction       string
	Owner              string
}

func (b *BotReport) String() string {
	return fmt.Sprintf("Name: %s\nAddress: %s\nProfileLink: %s\nBalance: %d\nNumSentPosts: %d\nNumSentReplies: %d\nNumFollowers: %d\nNumIncomingLikes: %d\nNumIncomingReplies: %d\nProfileUpdated: %t\nTotalActions: %d\nTotalInteractions: %d\nCreatedAt: %s\nLatestAction: %s\nOwner: %s\n",
		b.Name, b.Address, b.ProfileLink, b.Balance, b.NumSentPosts, b.NumSentReplies, b.NumFollowers, b.NumIncomingLikes, b.NumIncomingReplies, b.ProfileUpdated, b.TotalActions, b.TotalInteractions, b.CreatedAt, b.LatestAction, b.Owner)
}
