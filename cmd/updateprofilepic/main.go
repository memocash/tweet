package updateprofilepic

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/index/ref/bitcoin/util/testing/test_tx"
	"github.com/memocash/tweet/database"
	"github.com/memocash/tweet/tweets"
	"github.com/spf13/cobra"
)

var nameCmd = &cobra.Command{
	Use:   "updateprofilepic",
	Short: "Update profile picture on Memo to match a Twitter account",
	Args: cobra.ExactArgs(2),
	RunE: func(c *cobra.Command, args []string) error {
		key := test_tx.GetPrivateKey(args[0])
		address := key.GetAddress()
		account := args[1]
		_,_,pic := tweets.GetProfile(account)
		err := database.UpdateProfilePic(address,key,pic)
		if err != nil{
			jerr.Get("error", err).Fatal()
		}
		return nil
	},
}


func GetCommand() *cobra.Command {
	return nameCmd
}