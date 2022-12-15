package memobot

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/config"
	"github.com/memocash/tweet/database/util"
	"github.com/memocash/tweet/tweets"
	"github.com/spf13/cobra"
	"github.com/syndtr/goleveldb/leveldb"
)

var memobotCmd = &cobra.Command{
	Use:   "memobot",
	Short: "Listens for new transactions on a memo account",
	Long:  "Prints out each new transaction as it comes in. ",
	RunE: func(c *cobra.Command, args []string) error {
		botSeed:= config.GetConfig().BotSeed
		//get base key and address from seed
		mnemonic,err := wallet.GetMnemonicFromString(botSeed)
		if err != nil {
			return jerr.Get("error getting mnemonic from string", err)
		}
		path := wallet.GetBip44Path(wallet.Bip44CoinTypeBTC, 0, false)
		botKey, err := mnemonic.GetPath(path)
		if err != nil {
			return jerr.Get("error getting path", err)
		}
		db, err := leveldb.OpenFile("tweets.db", nil)
		botAddress := botKey.GetPublicKey().GetAddress().GetEncoded()
		err = util.MemoListen(botSeed, []string{botAddress},*botKey,tweets.Connect(), db)
		if err != nil {
			return jerr.Get("error listening for transactions", err)
		}
		return nil
	},
}

func GetCommand() *cobra.Command {
	return memobotCmd
}
