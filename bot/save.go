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
}

func NewSaveTx(bot *Bot) *SaveTx {
	return &SaveTx{
		Bot: bot,
	}
}

func (s *SaveTx) Save(tx graph.Tx) error {
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
	if err := s.HandleTxType(); err != nil {
		return jerr.Get("error handling request main bot", err)
	}
	return nil
}
func (s *SaveTx) HandleRequestMainBot() error {
	switch {
	case regexp.MustCompile("^CREATE @?([a-zA-Z0-9_]{1,15})(( --history( [0-9]+)?)?( --nolink)?( --date)?( --no-catch-up)?)*$").MatchString(s.Message):
		if err := s.HandleCreate(); err != nil {
			return jerr.Get("error handling create save tx", err)
		}
	case regexp.MustCompile("^WITHDRAW @?([a-zA-Z0-9_]{1,15})( [0-9]+)?$").MatchString(s.Message):
		if err := s.HandleWithdraw(); err != nil {
			return jerr.Get("error handling withdraw save tx", err)
		}
	default:
		fmt.Printf("Invalid command: %s\n.", s.Message)
		errMsg := "Invalid command. Please use the following format: CREATE <twitterName> or WITHDRAW <twitterName>"
		if err := refund(s.Tx, s.Bot, s.CoinIndex, s.SenderAddress, errMsg); err != nil {
			return jerr.Get("error refunding", err)
		}
	}
	return nil
}

func (s *SaveTx) SetVars(tx graph.Tx) error {
	s.Tx = tx
	for _, input := range tx.Inputs {
		if input.Output.Lock.Address == s.Bot.Addresses[0] {
			return nil
		}
	}
	var scriptArray []string
	for _, output := range tx.Outputs {
		scriptArray = append(scriptArray, output.Script)
	}
	s.Message = getMessageFromOutputScripts(scriptArray)
	for _, input := range tx.Inputs {
		if input.Output.Lock.Address != s.Bot.Addresses[0] {
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
		} else {
			err := s.HandleRequestSubBot()
			if err != nil {
				return jerr.Get("error handling request sub bot for save tx", err)
			}
		}
	}
	return nil
}
func (s *SaveTx) HandleRequestSubBot() error {
	//otherwise, one of the sub-bots has just been sent some funds, so based on the value of CatchUp, decide if we try to GetSkippedTweets
	botStreams, err := getBotStreams(s.Bot.Crypt)
	if err != nil {
		return jerr.Get("error getting bot streams", err)
	}
	var matchedStream *config.Stream = nil
	address := s.Tx.Outputs[s.CoinIndex].Lock.Address
	for _, botStream := range botStreams {
		if botStream.Wallet.Address.GetEncoded() == address {
			stream := config.Stream{
				Key:    botStream.Key,
				UserID: botStream.UserID,
				Sender: botStream.Sender,
				Wallet: botStream.Wallet,
			}
			matchedStream = &stream
			break
		}
	}
	if matchedStream == nil {
		return nil
	}
	flag, err := db.GetFlag(matchedStream.Sender, matchedStream.UserID)
	if err != nil || flag == nil {
		return jerr.Get("error getting flag", err)
	}
	if flag.Flags.CatchUp {
		accountKey := obj.AccountKey{
			UserID:  matchedStream.UserID,
			Key:     matchedStream.Wallet.Key,
			Address: matchedStream.Wallet.Address,
		}
		wlt := matchedStream.Wallet
		client := tweets.Connect()
		err = tweets.GetSkippedTweets(accountKey, &wlt, client, flag.Flags, 100, false)
		if err != nil && !jerr.HasErrorPart(err, gen.NotEnoughValueErrorText) {
			return jerr.Get("error getting skipped tweets", err)
		}
		err = s.Bot.SafeUpdate()
		if err != nil {
			return jerr.Get("error updating stream", err)
		}
	}
	return nil
}

func (s *SaveTx) HandleCreate() error {
	botRunningCount, err := db.GetBotRunningCount()
	if err != nil && !errors.Is(err, leveldb.ErrNotFound) {
		return jerr.Get("error getting bot running count", err)
	}
	if botRunningCount != nil && botRunningCount.Count >= 25 {
		err := refund(s.Tx, s.Bot, s.CoinIndex, s.SenderAddress, "There are too many bots, please try again later")
		if err != nil {
			return jerr.Get("error refunding", err)
		}
		return nil
	}
	//split the message into an array of strings
	splitMessage := strings.Split(s.Message, " ")
	//get the twitter name from the message
	twitterName := splitMessage[1]
	if twitterName[0] == '@' {
		twitterName = twitterName[1:]
	}
	//check if the twitter account exists, if so get the user id
	twitterExists := false
	twitterAccount, _, err := s.Bot.TweetClient.Users.Show(&twitter.UserShowParams{ScreenName: twitterName})
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
	//check if --history is in the message
	history := false
	var flags = db.GetDefaultFlags()
	var historyNum = 100
	for index, word := range splitMessage {
		if word == "--history" {
			history = true
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
		Address: s.SenderAddress,
		UserID:  twitterAccount.IDStr,
		Flags:   flags,
	}}); err != nil {
		return jerr.Get("error saving flags to db", err)
	}
	accountKeyPointer, wlt, err := createBotStream(s.Bot, twitterAccount, s.SenderAddress, s.Tx, s.CoinIndex)
	if err != nil {
		return jerr.Get("error creating bot", err)
	}
	//transfer all the tweets from the twitter account to the new bot
	if accountKeyPointer != nil {
		accountKey := *accountKeyPointer
		if history {
			client := tweets.Connect()
			if err = tweets.GetSkippedTweets(accountKey, wlt, client, flags, historyNum, true); err != nil && !jerr.HasErrorPart(err, gen.NotEnoughValueErrorText) {
				return jerr.Get("error getting skipped tweets on bot save tx", err)
			}

		}
		if err = s.Bot.SafeUpdate(); err != nil {
			return jerr.Get("error updating stream", err)
		}
	} else {
		if s.Bot.Verbose {
			jlog.Log("account key pointer is nil, not transferring tweets, bot not created")
		}
		return nil
	}
	return nil
}

func (s *SaveTx) HandleWithdraw() error {
	twitterName := regexp.MustCompile("^WITHDRAW @?([a-zA-Z0-9_]{1,15})( [0-9]+)?$").FindStringSubmatch(s.Message)[1]
	if twitterName[0] == '@' {
		twitterName = twitterName[1:]
	}
	twitterExists := false
	twitterAccount, _, err := s.Bot.TweetClient.Users.Show(&twitter.UserShowParams{ScreenName: twitterName})
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
	addressKey, err := db.GetAddressKey(s.SenderAddress, twitterAccount.ID)
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
			if err := s.Bot.SafeUpdate(); err != nil {
				return jerr.Get("error updating bot", err)
			}
		}
		return nil
	}
	if maxSend > 0 {
		err := tweetWallet.WithdrawAll(outputs, key, wallet.GetAddressFromString(s.SenderAddress))
		if err != nil {
			return jerr.Get("error withdrawing all", err)
		}
		if err := s.Bot.SafeUpdate(); err != nil {
			return jerr.Get("error updating bot", err)
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
