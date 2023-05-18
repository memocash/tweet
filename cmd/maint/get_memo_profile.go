package maint

import (
	"github.com/hasura/go-graphql-client"
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/graph"
	tweetWallet "github.com/memocash/tweet/wallet"
	"github.com/spf13/cobra"
	"log"
	"time"
)

var getMemoProfileCmd = &cobra.Command{
	Use:   "get-memo-profile",
	Short: "Testing getting profiles from memo addresses through graphQL",
	Run: func(c *cobra.Command, args []string) {
		if len(args) < 1 {
			jerr.Get("must specify address", nil).Fatal()
		}
		profiles, err := tweetWallet.GetProfile(args[0], time.Time{}, graphql.NewClient(graph.ServerUrlHttp, nil))
		if err != nil {
			jerr.Get("error getting profile", err).Fatal()
		}
		botProfile := profiles.Profiles[0]
		for _, post := range botProfile.Posts {
			log.Printf("Post: %s\n Seen at: %s", post.Text, post.Tx.Seen.GetTime().String())
		}
	},
}
