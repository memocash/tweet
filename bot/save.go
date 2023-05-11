package bot

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/jchavannes/btcd/chaincfg/chainhash"
	"github.com/jchavannes/btcd/txscript"
	"github.com/jchavannes/jgo/jerr"
	"github.com/jchavannes/jgo/jlog"
	"github.com/memocash/index/ref/bitcoin/memo"
	"github.com/memocash/index/ref/bitcoin/tx/gen"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/config"
	"github.com/memocash/tweet/db"
	"github.com/memocash/tweet/graph"
	"github.com/memocash/tweet/tweets"
	"github.com/memocash/tweet/tweets/obj"
	tweetWallet "github.com/memocash/tweet/wallet"
	"github.com/syndtr/goleveldb/leveldb"
	"log"
	"regexp"
	"strconv"
	"strings"
)

type SaveTx struct {
	Bot           *Bot
	Tx            graph.Tx
	Message       string
	SenderAddress string
	CoinIndex     uint32
	TxHash        chainhash.Hash
	Handled       bool
}

func NewSaveTx(bot *Bot) *SaveTx {
	return &SaveTx{
		Bot: bot,
	}
}

func (s *SaveTx) Save(tx graph.Tx) error {
	s.Handled = false
	if err := s.SetVars(tx); err != nil {
		return jerr.Get("error setting vars for save tx", err)
	}
	defer s.FinishSave()
	hasCompletedTx, err := db.HasCompletedTx(s.TxHash)
	if err != nil {
		return jerr.Get("error getting completed tx", err)
	}
	if hasCompletedTx {
		jlog.Logf("Already completed tx: %s\n", tx.Hash)
		return nil
	}
	if err = s.HandleTxType(); err != nil {
		return jerr.Get("error handling request main bot", err)
	}
	return nil
}
func (s *SaveTx) HandleRequestMainBot() error {
	switch {
	case s.Bot.Down:
		if err := s.HandleDown(); err != nil {
			return jerr.Get("error handling down bot message", err)
		}
	case regexp.MustCompile("^CREATE @?([a-zA-Z0-9_]{1,15})(( --history( [0-9]+)?)?( --nolink)?( --date)?( --no-catch-up)?)*$").MatchString(s.Message):
		if err := s.HandleCreate(); err != nil {
			return jerr.Get("error handling create save tx", err)
		}
	case regexp.MustCompile("^WITHDRAW @?([a-zA-Z0-9_]{1,15})( [0-9]+)?$").MatchString(s.Message):
		if err := s.HandleWithdraw(); err != nil {
			return jerr.Get("error handling withdraw save tx", err)
		}
	default:
		if s.Message != "" {
			log.Printf("Invalid command: %s\n.", s.Message)
			errMsg := "Invalid command. Please use the following format: CREATE <twitterName> or WITHDRAW <twitterName>"
			if err := refund(s.Tx, s.Bot, s.CoinIndex, s.SenderAddress, errMsg); err != nil {
				return jerr.Get("error refunding", err)
			}
		}
	}
	if err := s.Bot.SafeUpdate(); err != nil {
		return jerr.Get("error updating bot", err)
	}
	return nil
}

func (s *SaveTx) SetVars(tx graph.Tx) error {
	s.Tx = tx
	var scriptArray []string
	for _, output := range tx.Outputs {
		scriptArray = append(scriptArray, output.Script)
	}
	s.Message = getMessageFromOutputScripts(scriptArray)
	for _, input := range tx.Inputs {
		if s.SenderAddress == "" {
			s.SenderAddress = input.Output.Lock.Address
		} else if input.Output.Lock.Address != s.Bot.Addresses[0] {
			s.SenderAddress = input.Output.Lock.Address
			break
		}
	}
	for i, output := range tx.Outputs {
		if output.Lock.Address == s.Bot.Addresses[0] {
			s.CoinIndex = uint32(i)
			break
		}
	}
	txHash, err := chainhash.NewHashFromStr(tx.Hash)
	if err != nil {
		return jerr.Get("error parsing address receive tx hash for save", err)
	}
	s.TxHash = *txHash
	return nil
}

func (s *SaveTx) FinishSave() {
	if !s.Handled {
		return
	}
	var addressSeenTx = &db.AddressSeenTx{Address: s.Bot.Addr, Seen: s.Tx.Seen.GetTime(), TxHash: s.TxHash}
	var completed = &db.CompletedTx{TxHash: s.TxHash}
	if err := db.Save([]db.ObjectI{addressSeenTx, completed}); err != nil {
		s.Bot.ErrorChan <- jerr.Get("error adding tx hash to database", err)
	}
}
func (s *SaveTx) HandleTxType() error {
	for i, _ := range s.Tx.Outputs {
		s.CoinIndex = uint32(i)
		address := s.Tx.Outputs[s.CoinIndex].Lock.Address
		if address == s.Bot.Addresses[0] {
			err := s.HandleRequestMainBot()
			if err != nil {
				return jerr.Get("error handling request main bot for save tx", err)
			}
			s.Handled = true
		} else {
			botStreams, err := getBotStreams(s.Bot.Crypt, false)
			if err != nil {
				return jerr.Get("error getting bot streams", err)
			}
			var matchedStream *config.Stream = nil
			for _, botStream := range botStreams {
				if botStream.Wallet.Address.GetEncoded() == address && s.SenderAddress != botStream.Wallet.Address.GetEncoded() {
					matchedStream = &botStream
					err := s.HandleRequestSubBot(matchedStream)
					if err != nil {
						return jerr.Get("error handling request sub bot for save tx", err)
					}
					s.Handled = true
					break
				}
			}
		}
	}
	return nil
}
func (s *SaveTx) HandleRequestSubBot(matchedStream *config.Stream) error {
	log.Printf("Received tx for sub bot: %s\n", matchedStream.Wallet.Address.GetEncoded())
	//otherwise, one of the sub-bots has just been sent some funds, so based on the value of CatchUp, decide if we try to GetSkippedTweets
	flag, err := db.GetFlag(wallet.GetAddressFromString(matchedStream.Sender).GetAddr(), matchedStream.UserID)
	if err != nil || flag == nil {
		return jerr.Get("error getting flag", err)
	}
	accountKey := obj.AccountKey{
		UserID:  matchedStream.UserID,
		Key:     matchedStream.Wallet.Key,
		Address: matchedStream.Wallet.Address,
	}
	if s.SenderAddress == s.Bot.Addr.String() {
		log.Println("\n\nSub bot received funds from main bot\n\n")
		subBotCommand, err := db.GetSubBotCommand(s.TxHash)
		if err != nil {
			return jerr.Get("error getting sub bot command", err)
		}
		if subBotCommand == nil {
			return jerr.Get("sub bot command not found", errors.New("sub bot command not found"))
		}
		fmt.Printf("HistoryNum: %d\nBotExists: %b\n", subBotCommand.HistoryNum, subBotCommand.BotExists)
		if !subBotCommand.BotExists {
			log.Println("New bot, updating profile")
			_, err := updateProfile(s.Bot, matchedStream.Wallet.Address, matchedStream.Wallet.Key, matchedStream.UserID, matchedStream.Sender)
			if err != nil {
				return jerr.Get("error updating profile for sub bot", err)
			}
		}

		if subBotCommand.HistoryNum > 0 {
			log.Printf("History number was %d, getting that many skipped tweets\n", subBotCommand.HistoryNum)
			err = tweets.GetSkippedTweets(accountKey, &matchedStream.Wallet, s.Bot.TweetScraper, flag.Flags, subBotCommand.HistoryNum, !subBotCommand.BotExists)
			if err != nil && !jerr.HasErrorPart(err, gen.NotEnoughValueErrorText) {
				return jerr.Get("error getting skipped tweets on bot save tx", err)
			}
		} else if flag.Flags.CatchUp {
			log.Println("No history number passed, but CatchUp is true, getting 100 skipped tweets")
			err = tweets.GetSkippedTweets(accountKey, &matchedStream.Wallet, s.Bot.TweetScraper, flag.Flags, 100, !subBotCommand.BotExists)
			if err != nil && !jerr.HasErrorPart(err, gen.NotEnoughValueErrorText) {
				return jerr.Get("error getting skipped tweets on bot save tx", err)
			}
		} else {
			log.Printf("No history number passed, and CatchUp is false, not getting skipped tweets\n")
		}
	} else {
		log.Printf("\n\nSub bot received funds from %s\n\n", s.SenderAddress)
		if flag.Flags.CatchUp {
			log.Println("No history number passed, but CatchUp is true, getting 100 skipped tweets")
			err = tweets.GetSkippedTweets(accountKey, &matchedStream.Wallet, s.Bot.TweetScraper, flag.Flags, 100, false)
			if err != nil && !jerr.HasErrorPart(err, gen.NotEnoughValueErrorText) {
				return jerr.Get("error getting skipped tweets", err)
			}
		}
	}

	if err := s.Bot.SafeUpdate(); err != nil {
		return jerr.Get("error updating bot", err)
	}
	return nil
}

func (s *SaveTx) HandleDown() error {
	err := refund(s.Tx, s.Bot, s.CoinIndex, s.SenderAddress, "Sorry, the bot is currently down for maintenance. Please try again later.")
	if err != nil {
		return jerr.Get("error refunding", err)
	}
	return nil

}
func (s *SaveTx) HandleCreate() error {
	//split the message into an array of strings
	splitMessage := strings.Split(s.Message, " ")
	//get the twitter name from the message
	twitterName := splitMessage[1]
	if twitterName[0] == '@' {
		twitterName = twitterName[1:]
	}
	//check if the twitter account exists, if so get the user id
	twitterExists := false
	twitterProfile, err := s.Bot.TweetScraper.GetProfile(twitterName)
	if err == nil {
		twitterExists = true
	}
	if !twitterExists {
		err := refund(s.Tx, s.Bot, s.CoinIndex, s.SenderAddress, "Twitter account does not exist")
		if err != nil {
			return jerr.Get("error refunding", err)
		}
		return nil
	}
	userId, err := strconv.ParseInt(twitterProfile.UserID, 10, 64)
	if err != nil {
		return jerr.Get("error parsing user id", err)
	}
	twitterAccount := twitter.User{
		ID:         userId,
		ScreenName: twitterProfile.Username,
		IDStr:      twitterProfile.UserID,
	}
	//check if --history is in the message
	var flags = db.GetDefaultFlags()
	var historyNum = 0
	for index, word := range splitMessage {
		if word == "--history" {
			historyNum = 100
			if len(splitMessage) > index+1 {
				historyNum, err = strconv.Atoi(splitMessage[index+1])
				if err != nil {
					continue
				}
			}
		}
		if word == "--nolink" {
			flags.Link = false
		}
		if word == "--date" {
			flags.Date = true
		}
		if word == "--no-catch-up" {
			flags.CatchUp = false
		}
	}
	if historyNum > 1000 {
		err = refund(s.Tx, s.Bot, s.CoinIndex, s.SenderAddress, "Number of tweets must be less than 1000")
		if err != nil {
			return jerr.Get("error refunding", err)
		}
		return nil
	}
	if err := db.Save([]db.ObjectI{&db.Flag{
		Address: wallet.GetAddressFromString(s.SenderAddress).GetAddr(),
		UserID:  twitterAccount.ID,
		Flags:   flags,
	}}); err != nil {
		return jerr.Get("error saving flags to db", err)
	}
	err = createBotStream(s.Bot, &twitterAccount, s.SenderAddress, s.Tx, s.CoinIndex, historyNum)
	if err != nil {
		return jerr.Get("error creating bot", err)
	}
	return nil
}

func (s *SaveTx) HandleWithdraw() error {
	twitterName := regexp.MustCompile("^WITHDRAW @?([a-zA-Z0-9_]{1,15})( [0-9]+)?$").FindStringSubmatch(s.Message)[1]
	if twitterName[0] == '@' {
		twitterName = twitterName[1:]
	}
	twitterExists := false
	twitterProfile, err := s.Bot.TweetScraper.GetProfile(twitterName)
	if err == nil {
		twitterExists = true
	}
	if !twitterExists {
		err := refund(s.Tx, s.Bot, s.CoinIndex, s.SenderAddress, "Twitter account does not exist")
		if err != nil {
			return jerr.Get("error refunding", err)
		}
		return nil
	}
	userID, err := strconv.ParseInt(twitterProfile.UserID, 10, 64)
	if err != nil {
		return jerr.Get("error parsing user id", err)
	}
	twitterAccount := twitter.User{
		ID:         userID,
		ScreenName: twitterProfile.Name,
		IDStr:      twitterProfile.UserID,
	}
	addressKey, err := db.GetAddressKey(wallet.GetAddressFromString(s.SenderAddress).GetAddr(), twitterAccount.ID)
	if err != nil {
		if !errors.Is(err, leveldb.ErrNotFound) {
			return jerr.Get("error getting linked-"+s.SenderAddress+"-"+twitterAccount.IDStr, err)
		}
		errMsg := "No linked address found for " + s.SenderAddress + "-" + twitterAccount.IDStr
		err = refund(s.Tx, s.Bot, s.CoinIndex, s.SenderAddress, errMsg)
		if err != nil {
			return jerr.Get("error refunding no linked address key found", err)
		}
		return nil
	}
	decryptedKey, err := tweetWallet.Decrypt(addressKey.Key, s.Bot.Crypt)
	if err != nil {
		return jerr.Get("error decrypting key", err)
	}
	key, err := wallet.ImportPrivateKey(string(decryptedKey))
	if err != nil {
		return jerr.Get("error importing private key", err)
	}
	address := key.GetAddress()
	if s.Bot.Verbose {
		jlog.Logf("Withdrawing from address: %s\n", address.GetEncoded())
	}
	inputGetter := tweetWallet.InputGetter{Address: address}
	//use the address object of the spawned key to get the outputs array
	outputs, err := inputGetter.GetUTXOs(nil)
	if err != nil {
		return jerr.Get("error getting utxos", err)
	}
	//check if the message contains a number
	var amount int64
	var maxSend = memo.GetMaxSendForUTXOs(outputs)
	if regexp.MustCompile("^WITHDRAW @?([a-zA-Z0-9_]{1,15}) [0-9]+$").MatchString(s.Message) {
		amount, _ = strconv.ParseInt(regexp.MustCompile("^WITHDRAW @?([a-zA-Z0-9_]{1,15}) ([0-9]+)$").FindStringSubmatch(s.Message)[2], 10, 64)
		if amount > maxSend {
			err = refund(s.Tx, s.Bot, s.CoinIndex, s.SenderAddress, "Cannot withdraw more than the total balance is capable of sending")
			if err != nil {
				return jerr.Get("error refunding", err)
			}
			return nil
		} else if amount+memo.DustMinimumOutput+memo.OutputFeeP2PKH > maxSend {
			errmsg := fmt.Sprintf("Not enough funds will be left over to send change to bot account, please withdraw less than %d", maxSend+1-memo.DustMinimumOutput-memo.OutputFeeP2PKH)
			err = refund(s.Tx, s.Bot, s.CoinIndex, s.SenderAddress, errmsg)
			if err != nil {
				return jerr.Get("error refunding", err)
			}
			return nil
		} else {
			err := tweetWallet.WithdrawAmount(outputs, key, wallet.GetAddressFromString(s.SenderAddress), amount)
			if err != nil {
				return jerr.Get("error withdrawing amount", err)
			}
		}
		return nil
	}
	if maxSend > 0 {
		err := tweetWallet.WithdrawAll(outputs, key, wallet.GetAddressFromString(s.SenderAddress))
		if err != nil {
			return jerr.Get("error withdrawing all", err)
		}
	} else {
		err = refund(s.Tx, s.Bot, s.CoinIndex, s.SenderAddress, "Not enough balance to withdraw anything")
		if err != nil {
			return jerr.Get("error refunding", err)
		}
	}
	return nil
}

func getMessageFromOutputScripts(outputScripts []string) string {
	for _, script := range outputScripts {
		lockScript, err := hex.DecodeString(script)
		if err != nil {
			panic(err)
		}
		pushData, err := txscript.PushedData(lockScript)
		if err != nil {
			panic(err)
		}

		if len(pushData) > 2 && bytes.Equal(pushData[0], memo.PrefixSendMoney) {
			message := string(pushData[2])
			return message
		}
	}
	return ""
}
