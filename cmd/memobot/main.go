package memobot

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/bot"
	"github.com/memocash/tweet/config"
	"github.com/memocash/tweet/database"
	"github.com/memocash/tweet/tweets"
	"github.com/spf13/cobra"
	"strconv"
)

var memobotCmd = &cobra.Command{
	Use:   "memobot",
	Short: "Listens for new transactions on a memo account",
	Long:  "Prints out each new transaction as it comes in. ",
	RunE: func(c *cobra.Command, args []string) error {
		botSeed := config.GetConfig().BotSeed
		//get base key and address from seed
		mnemonic, err := wallet.GetMnemonicFromString(botSeed)
		if err != nil {
			return jerr.Get("error getting mnemonic from string", err)
		}
		path := wallet.GetBip44Path(wallet.Bip44CoinTypeBTC, 0, false)
		botKey, err := mnemonic.GetPath(path)
		if err != nil {
			return jerr.Get("error getting path", err)
		}
		db, err := database.GetDb()
		if err != nil {
			return jerr.Get("error opening db", err)
		}
		//check that memobot-num-streams field of the database exists
		fieldExists, err := db.Has([]byte("memobot-num-streams"), nil)
		if err != nil {
			return jerr.Get("error checking if memobot-num-streams field exists", err)
		}
		//if it doesn't, create it and set it to 0
		if !fieldExists {
			err = db.Put([]byte("memobot-num-streams"), []byte("0"), nil)
			if err != nil {
				return jerr.Get("error creating memobot-num-streams field", err)
			}
		}
		//get the number of streams from the database
		numStreamsBytes, err := db.Get([]byte("memobot-num-streams"), nil)
		if err != nil {
			return jerr.Get("error getting memobot-num-streams field", err)
		}
		numStreamsUint, err := strconv.ParseUint(string(numStreamsBytes), 10, 64)
		println("num streams: ", numStreamsUint)

		botAddress := botKey.GetPublicKey().GetAddress().GetEncoded()
		memoBot := bot.NewBot(mnemonic, []string{botAddress}, *botKey, tweets.Connect(), db)
		if err = memoBot.Listen(); err != nil {
			return jerr.Get("error listening for transactions", err)
		}
		return nil
	},
}

func GetCommand() *cobra.Command {
	return memobotCmd
}
