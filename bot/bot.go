package bot

import (
	"errors"
	"fmt"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/jchavannes/btcd/chaincfg/chainhash"
	"github.com/jchavannes/jgo/jerr"
	"github.com/jchavannes/jgo/jlog"
	"github.com/memocash/index/ref/bitcoin/memo"
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
	"sync"
	"time"
)

type Bot struct {
	Mnemonic    *wallet.Mnemonic
	Addresses   []string
	Addr        wallet.Addr
	Key         wallet.PrivateKey
	TweetClient *twitter.Client
	Db          *leveldb.DB
	Stream      *tweets.Stream
	ErrorChan   chan error
	TxMutex     sync.Mutex
	UpdateMutex sync.Mutex
	Crypt       []byte
	Timer       *time.Timer
	Verbose     bool
}

func NewBot(mnemonic *wallet.Mnemonic, addresses []string, key wallet.PrivateKey, tweetClient *twitter.Client, db *leveldb.DB, verbose bool) (*Bot, error) {
	if len(addresses) == 0 {
		return nil, jerr.New("error new bot, no addresses")
	}
	addr, err := wallet.GetAddrFromString(addresses[0])
	if err != nil {
		return nil, jerr.Get("error getting address from string for new bot", err)
	}
	return &Bot{
		Mnemonic:    mnemonic,
		Addresses:   addresses,
		Addr:        *addr,
		Key:         key,
		TweetClient: tweetClient,
		Db:          db,
		ErrorChan:   make(chan error),
		Verbose:     verbose,
	}, nil
}

func (b *Bot) ProcessMissedTxs() error {
	recentAddressSeenTx, err := db.GetRecentAddressSeenTx(b.Addr)
	if err != nil && !errors.Is(err, leveldb.ErrNotFound) {
		return jerr.Get("error getting recent address seen tx for addr", err)
	}
	var start time.Time
	if recentAddressSeenTx != nil {
		start = recentAddressSeenTx.Seen
	}
	if b.Verbose {
		jlog.Logf("Processing missed txs using start: %s\n", start.Format(time.RFC3339))
	}
	txs, err := graph.GetAddressUpdates(b.Addresses[0], start)
	if err != nil {
		return jerr.Get("error getting address update txs", err)
	} else if b.Verbose {
		jlog.Logf("Found %d missed txs\n", len(txs))
	}
	for _, tx := range txs {
		if err := b.SaveTx(tx); err != nil {
			return jerr.Get("error saving missed process tx", err)
		} else if b.Verbose {
			jlog.Logf("Found missed process tx: %s, seen: %s\n", tx.Hash, tx.Seen.GetTime().Format(time.RFC3339))
		}
	}
	return nil
}

func (b *Bot) Listen() error {
	jlog.Logf("Bot listening to address: %s\n", b.Addresses[0])
	if err := graph.AddressListen(b.Addresses, b.SaveTx, b.ErrorChan); err != nil {
		return jerr.Get("error listening to address on graphql", err)
	}
	updateInterval := config.GetConfig().UpdateInterval
	if updateInterval == 0 {
		updateInterval = 180
	}
	go func() {
		for {
			t := time.NewTimer(time.Duration(updateInterval) * time.Minute)
			select {
			case <-t.C:
			}
			botStreams, err := getBotStreams(b.Crypt)
			if err != nil {
				b.ErrorChan <- jerr.Get("error making stream array bot listen", err)
			}
			if err = updateProfiles(botStreams, b); err != nil {
				b.ErrorChan <- jerr.Get("error updating profiles for bot listen", err)
			}
		}
	}()
	var err error
	if b.Stream, err = tweets.NewStream(); err != nil {
		return jerr.Get("error getting new tweet stream", err)
	}
	botStreams, err := getBotStreams(b.Crypt)
	if err != nil {
		return jerr.Get("error getting bot streams for listen skipped", err)
	}
	for _, stream := range botStreams {
		if err = tweets.GetSkippedTweets(obj.AccountKey{
			Account: stream.Name,
			Key:     stream.Wallet.Key,
			Address: stream.Wallet.Address,
		}, &stream.Wallet, b.TweetClient, db.GetDefaultFlags(), 100); err != nil {
			return jerr.Get("error getting skipped tweets on bot listen", err)
		}
	}
	if err = b.SafeUpdate(); err != nil {
		return jerr.Get("error updating stream 2nd time", err)
	}
	return jerr.Get("error in listen", <-b.ErrorChan)
}

func (b *Bot) SaveTx(tx graph.Tx) error {
	b.TxMutex.Lock()
	defer b.TxMutex.Unlock()
	for _, input := range tx.Inputs {
		if input.Output.Lock.Address == b.Addresses[0] {
			return nil
		}
	}
	var scriptArray []string
	for _, output := range tx.Outputs {
		scriptArray = append(scriptArray, output.Script)
	}
	message := grabMessage(scriptArray)
	if message == "" {
		println("No message found, skipping")
		return nil
	}
	senderAddress := ""
	for _, input := range tx.Inputs {
		if input.Output.Lock.Address != b.Addresses[0] {
			senderAddress = input.Output.Lock.Address
			break
		}
	}
	coinIndex := uint32(0)
	for i, output := range tx.Outputs {
		if output.Lock.Address == b.Addresses[0] {
			coinIndex = uint32(i)
			break
		}
	}
	txHash, err := chainhash.NewHashFromStr(tx.Hash)
	if err != nil {
		return jerr.Get("error parsing address receive tx hash", err)
	}
	defer func() {
		var addressSeenTx = &db.AddressSeenTx{Address: b.Addr, Seen: tx.Seen.GetTime(), TxHash: *txHash}
		var completed = &db.CompletedTx{TxHash: *txHash}
		if err := db.Save([]db.ObjectI{addressSeenTx, completed}); err != nil {
			b.ErrorChan <- jerr.Get("error adding tx hash to database", err)
		}
	}()
	hasCompletedTx, err := db.HasCompletedTx(*txHash)
	if err != nil {
		return jerr.Get("error getting completed tx", err)
	}
	if hasCompletedTx {
		jlog.Logf("Already completed tx: %s\n", tx.Hash)
		return nil
	}
	match, _ := regexp.MatchString("^CREATE @?([a-zA-Z0-9_]{1,15})(( --history( [0-9]+)?)?( --nolink)?( --date)?)*$", message)
	if match {
		//check how many streams are running
		streams, err := b.Db.Get([]byte("memobot-running-count"), nil)
		if err != nil {
			return jerr.Get("error getting running count", err)
		}
		//convert the byte array to an int
		numStreams, err := strconv.Atoi(string(streams))
		//if there are more than 25 streams running, refund and return
		if numStreams >= 25 {
			err := refund(tx, b, coinIndex, senderAddress, "There are too many streams running, please try again later")
			if err != nil {
				return jerr.Get("error refunding", err)
			}
			return nil
		}
		//split the message into an array of strings
		splitMessage := strings.Split(message, " ")
		//get the twitter name from the message
		twitterName := splitMessage[1]
		if twitterName[0] == '@' {
			twitterName = twitterName[1:]
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
		}
		if historyNum > 1000 {
			err = refund(tx, b, coinIndex, senderAddress, "Number of tweets must be less than 1000")
			if err != nil {
				return jerr.Get("error refunding", err)
			}
			return nil
		}
		if err := db.Save([]db.ObjectI{&db.Flag{
			Address:     senderAddress,
			TwitterName: twitterName,
			Flags:       flags,
		}}); err != nil {
			return jerr.Get("error saving flags to db", err)
		}
		accountKeyPointer, wlt, err := createBotStream(b, twitterName, senderAddress, tx, coinIndex)
		if err != nil {
			return jerr.Get("error creating bot", err)
		}
		//transfer all the tweets from the twitter account to the new bot
		if accountKeyPointer != nil {
			accountKey := *accountKeyPointer
			if history {
				client := tweets.Connect()
				if err = tweets.GetSkippedTweets(accountKey, wlt, client, flags, historyNum); err != nil {
					return jerr.Get("error getting skipped tweets on bot save tx", err)
				}

			}
			if err = b.SafeUpdate(); err != nil {
				return jerr.Get("error updating stream", err)
			}
		} else {
			if b.Verbose {
				jlog.Log("account key pointer is nil, not transferring tweets, bot not created")
			}
			return nil
		}
	} else if regexp.MustCompile("^WITHDRAW @?([a-zA-Z0-9_]{1,15})( [0-9]+)?$").MatchString(message) {
		//check the database for each field that matches linked-<senderAddress>-<twitterName>
		//if there is a match, print out the address and key
		//if there is no match, print out an error message
		twitterName := regexp.MustCompile("^WITHDRAW @?([a-zA-Z0-9_]{1,15})( [0-9]+)?$").FindStringSubmatch(message)[1]
		if twitterName[0] == '@' {
			twitterName = twitterName[1:]
		}
		addressKey, err := db.GetAddressKey(senderAddress, twitterName)
		if err != nil {
			if !errors.Is(err, leveldb.ErrNotFound) {
				return jerr.Get("error getting linked-"+senderAddress+"-"+twitterName, err)
			}
			errMsg := "No linked address found for " + senderAddress + "-" + twitterName
			err = refund(tx, b, coinIndex, senderAddress, errMsg)
			if err != nil {
				return jerr.Get("error refunding no linked address key found", err)
			}
			return nil
		} else {
			decryptedKey, err := tweetWallet.Decrypt(addressKey.Key, b.Crypt)
			if err != nil {
				return jerr.Get("error decrypting key", err)
			}
			key, err := wallet.ImportPrivateKey(string(decryptedKey))
			if err != nil {
				return jerr.Get("error importing private key", err)
			}
			address := key.GetAddress()
			if b.Verbose {
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
			if regexp.MustCompile("^WITHDRAW @?([a-zA-Z0-9_]{1,15}) [0-9]+$").MatchString(message) {
				amount, _ = strconv.ParseInt(regexp.MustCompile("^WITHDRAW @?([a-zA-Z0-9_]{1,15}) ([0-9]+)$").FindStringSubmatch(message)[2], 10, 64)
				if amount > maxSend {
					err = refund(tx, b, coinIndex, senderAddress, "Cannot withdraw more than the total balance is capable of sending")
					if err != nil {
						return jerr.Get("error refunding", err)
					}
					return nil
				} else if amount+memo.DustMinimumOutput+memo.OutputFeeP2PKH > maxSend {
					errmsg := fmt.Sprintf("Not enough funds will be left over to send change to bot account, please withdraw less than %d", maxSend+1-memo.DustMinimumOutput-memo.OutputFeeP2PKH)
					err = refund(tx, b, coinIndex, senderAddress, errmsg)
					if err != nil {
						return jerr.Get("error refunding", err)
					}
					return nil
				} else {
					err := tweetWallet.WithdrawAmount(outputs, key, wallet.GetAddressFromString(senderAddress), amount)
					if err != nil {
						return jerr.Get("error withdrawing amount", err)
					}
				}
			} else {
				if maxSend > 0 {
					err := tweetWallet.WithdrawAll(outputs, key, wallet.GetAddressFromString(senderAddress))
					if err != nil {
						return jerr.Get("error withdrawing all", err)
					}
				} else {
					err = refund(tx, b, coinIndex, senderAddress, "Not enough balance to withdraw anything")
					if err != nil {
						return jerr.Get("error refunding", err)
					}
					return nil
				}
			}
		}
		if err = b.SafeUpdate(); err != nil {
			return jerr.Get("error updating stream", err)
		}
	} else {
		errMsg := "Invalid command. Please use the following format: CREATE <twitterName> or WITHDRAW <twitterName>"
		err = refund(tx, b, coinIndex, senderAddress, errMsg)
		if err != nil {
			return jerr.Get("error refunding", err)
		}
	}
	return nil
}

func (b *Bot) SafeUpdate() error {
	b.UpdateMutex.Lock()
	defer b.UpdateMutex.Unlock()
	var waitCount = 1
	err := b.UpdateStream()
	for err != nil && waitCount < 30 {
		if !jerr.HasErrorPart(err, "DuplicateRule") {
			return jerr.Get("error updating stream", err)
		}
		jlog.Logf("Error updating stream: %s\n", err.Error())
		err = b.UpdateStream()
		time.Sleep(time.Duration(waitCount) * time.Second)
		waitCount *= 2
	}
	if err != nil {
		return jerr.Get("error updating stream", err)
	}
	return nil
}

func (b *Bot) UpdateStream() error {
	//create an array of {twitterName, newKey} objects by searching through the linked-<senderAddress>-<twitterName> fields
	botStreams, err := getBotStreams(b.Crypt)
	if err != nil {
		return jerr.Get("error making stream array update", err)
	}
	for _, stream := range botStreams {
		streamKey, err := wallet.ImportPrivateKey(stream.Key)
		if err != nil {
			return jerr.Get("error importing private key", err)
		}
		streamAddress := streamKey.GetAddress()
		if b.Verbose {
			jlog.Logf("streaming %s to address %s\n", stream.Name, streamAddress.GetEncoded())
		}
	}
	err = b.Db.Put([]byte("memobot-running-count"), []byte(strconv.FormatUint(uint64(len(botStreams)), 10)), nil)
	if err != nil {
		return jerr.Get("error updating running count", err)
	}
	go func() {
		if len(botStreams) == 0 {
			return
		}
		if err := b.Stream.ListenForNewTweets(botStreams); err != nil {
			b.ErrorChan <- jerr.Get("error twitter initiate stream in update", err)
		}
	}()
	return nil
}
