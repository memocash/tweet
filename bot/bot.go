package bot

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/jchavannes/btcd/chaincfg/chainhash"
	"github.com/jchavannes/jgo/jerr"
	"github.com/jchavannes/jgo/jlog"
	"github.com/memocash/index/ref/bitcoin/memo"
	"github.com/memocash/index/ref/bitcoin/tx/hs"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/config"
	"github.com/memocash/tweet/db"
	"github.com/memocash/tweet/graph"
	"github.com/memocash/tweet/tweets"
	"github.com/memocash/tweet/tweets/obj"
	tweetWallet "github.com/memocash/tweet/wallet"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
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
	Mutex       sync.Mutex
	Crypt       string
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
			streamArray, err := makeStreamArray(b)
			if err != nil {
				b.ErrorChan <- jerr.Get("error making stream array bot listen", err)
			}
			err = updateProfiles(streamArray, b)
			if err != nil {
				fmt.Printf("\n\nError updating profiles: %s\n\n", err.Error())
			}
		}
	}()
	var err error
	if b.Stream, err = tweets.NewStream(); err != nil {
		return jerr.Get("error getting new tweet stream", err)
	}
	streamArray, err := b.SafeUpdate()
	if err != nil {
		return jerr.Get("error updating stream", err)
	}
	for _, stream := range streamArray {
		if err = tweets.GetSkippedTweets(obj.AccountKey{
			Account: stream.Name,
			Key:     stream.Wallet.Key,
			Address: stream.Wallet.Address,
		}, &stream.Wallet, b.TweetClient, db.GetDefaultFlags(), 100); err != nil {
			return jerr.Get("error getting skipped tweets on bot listen", err)
		}
	}
	if _, err = b.SafeUpdate(); err != nil {
		return jerr.Get("error updating stream 2nd time", err)
	}
	return jerr.Get("error in listen", <-b.ErrorChan)
}

func (b *Bot) SaveTx(tx graph.Tx) error {
	b.Mutex.Lock()
	defer b.Mutex.Unlock()
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
		accountKeyPointer, wlt, err := createBot(b, twitterName, senderAddress, tx, coinIndex)
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
			_, err = b.SafeUpdate()
			if err != nil {
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
		searchString := "linked-" + senderAddress + "-" + twitterName
		//refund if this field doesn't exist
		searchValue, err := b.Db.Get([]byte(searchString), nil)
		if err != nil {
			if err == leveldb.ErrNotFound {
				//handle refund
				errMsg := "No linked address found for " + senderAddress + "-" + twitterName
				err = refund(tx, b, coinIndex, senderAddress, errMsg)
				if err != nil {
					return jerr.Get("error refunding", err)
				}
				return nil
			}
			errMsg := "Error accessing database looking for existing linked address"
			_ = refund(tx, b, coinIndex, senderAddress, errMsg)
			return jerr.Get("error getting linked-"+senderAddress+"-"+twitterName, err)
		} else {
			fieldName := searchString
			stringKey := searchValue
			println("Field name: " + string(fieldName))
			//get the address object from the private key
			decryptedKey, err := tweetWallet.Decrypt(stringKey, []byte(b.Crypt))
			if err != nil {
				return jerr.Get("error decrypting key", err)
			}
			key, err := wallet.ImportPrivateKey(string(decryptedKey))
			if err != nil {
				return jerr.Get("error importing private key", err)
			}
			address := key.GetAddress()
			println("Address: " + address.GetEncoded())
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
		_, err = b.SafeUpdate()
		if err != nil {
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
func refund(tx graph.Tx, b *Bot, coinIndex uint32, senderAddress string, errMsg string) error {
	_, err := b.SafeUpdate()
	if err != nil {
		return jerr.Get("error updating stream", err)
	}
	jlog.Logf("Sending refund error message to %s: %s\n", senderAddress, errMsg)
	sentToMainBot := false
	//check all the outputs to see if any of them match the bot's address, if not, return nil, if so, continue with the function
	for _, output := range tx.Outputs {
		if output.Lock.Address == b.Addresses[0] {
			sentToMainBot = true
			break
		}
	}
	if !sentToMainBot {
		return nil
	}
	//handle sending back money
	//not enough to send back
	if memo.GetMaxSendFromCount(tx.Outputs[coinIndex].Amount, 1) <= 0 {
		if b.Verbose {
			jlog.Log("Not enough funds to refund")
		}
		return nil
	}
	//create a transaction with the sender address and the amount of the transaction
	pkScript, err := hex.DecodeString(tx.Outputs[coinIndex].Script)
	if err != nil {
		return jerr.Get("error decoding script pk script for refund", err)
	}
	if err := tweetWallet.SendToTwitterAddress(memo.UTXO{Input: memo.TxInput{
		Value:        tx.Outputs[coinIndex].Amount,
		PrevOutHash:  hs.GetTxHash(tx.Hash),
		PrevOutIndex: coinIndex,
		PkHash:       wallet.GetAddressFromString(b.Addresses[0]).GetPkHash(),
		PkScript:     pkScript,
	}}, b.Key, wallet.GetAddressFromString(senderAddress), errMsg); err != nil {
		return jerr.Get("error sending money back", err)
	}
	return nil
}
func (b *Bot) SafeUpdate() ([]config.Stream, error) {
	streamArray, err := b.UpdateStream()
	var waitCount = 1
	for err != nil && waitCount < 30 {
		if !jerr.HasErrorPart(err, "DuplicateRule") {
			return nil, jerr.Get("error updating stream", err)
		}
		fmt.Printf("\n\n\nError updating stream: %s\n\n\n", err.Error())
		streamArray, err = b.UpdateStream()
		time.Sleep(time.Duration(waitCount) * time.Second)
		waitCount *= 2
	}
	if err != nil {
		return nil, jerr.Get("error updating stream", err)
	}
	return streamArray, nil
}
func (b *Bot) UpdateStream() ([]config.Stream, error) {
	//create an array of {twitterName, newKey} objects by searching through the linked-<senderAddress>-<twitterName> fields
	streamArray, err := makeStreamArray(b)
	if err != nil {
		return nil, jerr.Get("error making stream array update", err)
	}
	for _, stream := range streamArray {
		streamKey, err := wallet.ImportPrivateKey(stream.Key)
		if err != nil {
			return nil, jerr.Get("error importing private key", err)
		}
		streamAddress := streamKey.GetAddress()
		if b.Verbose {
			jlog.Logf("streaming %s to address %s\n", stream.Name, streamAddress.GetEncoded())
		}
	}
	err = b.Db.Put([]byte("memobot-running-count"), []byte(strconv.FormatUint(uint64(len(streamArray)), 10)), nil)
	if err != nil {
		return nil, jerr.Get("error updating running count", err)
	}
	go func() {
		if len(streamArray) == 0 {
			return
		}
		if err := b.Stream.ListenForNewTweets(streamArray); err != nil {
			b.ErrorChan <- jerr.Get("error twitter initiate stream in update", err)
		}
	}()
	return streamArray, nil
}

func createBot(b *Bot, twitterName string, senderAddress string, tx graph.Tx, coinIndex uint32) (*obj.AccountKey, *tweetWallet.Wallet, error) {
	//check if the value of the transaction is less than 5,000 or this address already has a bot for this account in the database
	botExists := false
	_, err := b.Db.Get([]byte("linked-"+senderAddress+"-"+twitterName), nil)
	if err != nil && err != leveldb.ErrNotFound {
		return nil, nil, jerr.Get("error getting bot from database", err)
	} else if err == nil {
		botExists = true
	}
	//check if this twitter account actually exists
	twitterExists := false
	if _, _, err := b.TweetClient.Users.Show(&twitter.UserShowParams{ScreenName: twitterName}); err == nil {
		twitterExists = true
	}
	if !twitterExists || tx.Outputs[coinIndex].Amount < 5000 {
		if tx.Outputs[coinIndex].Amount < 546 {
			return nil, nil, nil
		}
		errMsg := ""
		if !twitterExists {
			errMsg = fmt.Sprintf("Twitter account @%s does not exist", twitterName)
		} else {
			errMsg = fmt.Sprintf("You need to send at least 5,000 satoshis to create a bot for the account @%s", twitterName)
		}
		err = refund(tx, b, coinIndex, senderAddress, errMsg)
		if err != nil {
			return nil, nil, jerr.Get("error refunding", err)
		}
		return nil, nil, nil
	}
	println(b.Addresses[0])
	var newKey wallet.PrivateKey
	var newAddr wallet.Address
	numStreamBytes, err := b.Db.Get([]byte("memobot-num-streams"), nil)
	if err != nil {
		return nil, nil, jerr.Get("error getting num-streams", err)
	}
	numStream, err := strconv.ParseUint(string(numStreamBytes), 10, 64)
	if err != nil {
		return nil, nil, jerr.Get("error parsing num-streams", err)
	}
	//convert numStream to a uint
	numStreamUint := uint(numStream)
	if botExists {
		//get the key from the database
		//decrypt
		rawKey, err := b.Db.Get([]byte("linked-"+senderAddress+"-"+twitterName), nil)
		if err != nil {
			return nil, nil, jerr.Get("error getting key from database", err)
		}
		decryptedKey, err := tweetWallet.Decrypt(rawKey, []byte(b.Crypt))
		if err != nil {
			return nil, nil, jerr.Get("error decrypting key", err)
		}
		newKey, err = wallet.ImportPrivateKey(string(decryptedKey))
		if err != nil {
			return nil, nil, jerr.Get("error importing private key", err)
		}
		newAddr = newKey.GetAddress()
	} else {
		path := wallet.GetBip44Path(wallet.Bip44CoinTypeBTC, numStreamUint+1, false)
		keyPointer, err := b.Mnemonic.GetPath(path)
		newKey = *keyPointer
		if err != nil {
			return nil, nil, jerr.Get("error getting path", err)
		}
		newAddr = newKey.GetAddress()
	}
	pkScript, err := hex.DecodeString(tx.Outputs[coinIndex].Script)
	if err != nil {
		return nil, nil, jerr.Get("error decoding script pk script for create bot", err)
	}
	if err := tweetWallet.FundTwitterAddress(memo.UTXO{Input: memo.TxInput{
		Value:        tx.Outputs[coinIndex].Amount,
		PrevOutHash:  hs.GetTxHash(tx.Hash),
		PrevOutIndex: coinIndex,
		PkHash:       b.Key.GetAddress().GetPkHash(),
		PkScript:     pkScript,
	}}, b.Key, newAddr); err != nil {
		return nil, nil, jerr.Get("error funding twitter address", err)
	}
	newWallet := tweetWallet.NewWallet(newAddr, newKey)
	if !botExists {
		err = updateProfile(b, newWallet, twitterName, senderAddress)
		if err != nil {
			return nil, nil, jerr.Get("error updating profile", err)
		}
	}
	if b.Verbose {
		jlog.Logf("Create bot stream Address: " + newAddr.GetEncoded())
	}
	if !botExists {
		err = b.Db.Put([]byte("memobot-num-streams"), []byte(strconv.FormatUint(uint64(numStreamUint+1), 10)), nil)
		if err != nil {
			return nil, nil, jerr.Get("error putting num-streams", err)
		}
		//add a field to the database that links the sending address and twitter name to the new key
		//encrypt
		encryptedKey, err := tweetWallet.Encrypt([]byte(newKey.GetBase58Compressed()), []byte(b.Crypt))
		if err != nil {
			return nil, nil, jerr.Get("error encrypting key", err)
		}
		err = b.Db.Put([]byte("linked-"+senderAddress+"-"+twitterName), encryptedKey, nil)
		if err != nil {
			return nil, nil, jerr.Get("error updating linked-"+senderAddress+"-"+twitterName, err)
		}
	}
	accountKey := obj.GetAccountKeyFromArgs([]string{newKey.GetBase58Compressed(), twitterName})
	return &accountKey, &newWallet, nil
}
func updateProfiles(streamAray []config.Stream, b *Bot) error {
	for _, stream := range streamAray {
		streamKey, err := wallet.ImportPrivateKey(stream.Key)
		if err != nil {
			return jerr.Get("error importing private key", err)
		}
		streamAddress := streamKey.GetAddress()
		newWallet := tweetWallet.NewWallet(streamAddress, streamKey)
		err = updateProfile(b, newWallet, stream.Name, stream.Sender)
		time.Sleep(1 * time.Second)
	}
	return nil
}
func updateProfile(b *Bot, newWallet tweetWallet.Wallet, twitterName string, senderAddress string) error {
	profile, err := tweets.GetProfile(twitterName, b.TweetClient)
	if err != nil {
		return jerr.Get("fatal error getting profile", err)
	}
	existingDbProfile, err := db.GetProfile(senderAddress, twitterName)
	if err != nil && err != leveldb.ErrNotFound {
		return jerr.Get("error getting profile from database", err)
	}
	if existingDbProfile == nil {
		if err = tweetWallet.UpdateName(newWallet, profile.Name); err != nil {
			return jerr.Get("error updating name", err)
		}
		if err = tweetWallet.UpdateProfileText(newWallet, profile.Description); err != nil {
			return jerr.Get("error updating profile text", err)
		}
		if err = tweetWallet.UpdateProfilePic(newWallet, profile.ProfilePic); err != nil {
			return jerr.Get("error updating profile pic", err)
		}
		if b.Verbose {
			jlog.Log("updated profile info for the first time")
		}
	} else {
		dbProfile := new(tweets.Profile)
		if err = json.Unmarshal(existingDbProfile.Profile, &dbProfile); err != nil {
			return jerr.Getf(err, "error unmarshalling profile: %s", dbProfile.Name)
		}
		if dbProfile.Name != profile.Name {
			if err = tweetWallet.UpdateName(newWallet, profile.Name); err != nil {
				return jerr.Get("error updating name", err)
			} else if b.Verbose {
				jlog.Logf("updated profile name for %s: %s to %s\n", profile.ID, dbProfile.Name, profile.Name)
			}
		}
		if dbProfile.Description != profile.Description {
			if err = tweetWallet.UpdateProfileText(newWallet, profile.Description); err != nil {
				return jerr.Get("error updating profile text", err)
			} else if b.Verbose {
				jlog.Logf("updated profile text for %s: %s to %s\n", profile.ID, dbProfile.Description, profile.Description)
			}
		}
		if dbProfile.ProfilePic != profile.ProfilePic {
			if err = tweetWallet.UpdateProfilePic(newWallet, profile.ProfilePic); err != nil {
				return jerr.Get("error updating profile pic", err)
			} else if b.Verbose {
				jlog.Logf("updated profile pic for %s: %s to %s\n", profile.ID, dbProfile.ProfilePic, profile.ProfilePic)
			}
		}
	}
	profileBytes, err := json.Marshal(profile)
	if err != nil {
		return jerr.Get("error marshalling profile", err)
	}
	if err := db.Save([]db.ObjectI{&db.Profile{
		Address:     senderAddress,
		TwitterName: twitterName,
		Profile:     profileBytes,
	}}); err != nil {
		return jerr.Get("error saving profile to database", err)
	}
	if b.Verbose {
		jlog.Logf("checked for profile updates: %s (%s)", twitterName, senderAddress)
	}
	return nil
}

func makeStreamArray(b *Bot) ([]config.Stream, error) {
	streamArray := make([]config.Stream, 0)
	iter := b.Db.NewIterator(util.BytesPrefix([]byte("linked-")), nil)
	for iter.Next() {
		//find the twitterName at the end of the linked-<senderAddress>-<twitterName> field
		senderAddress := strings.Split(string(iter.Key()), "-")[1]
		twitterName := strings.Split(string(iter.Key()), "-")[2]
		//decrypt
		decryptedKeyByte, err := tweetWallet.Decrypt(iter.Value(), []byte(b.Crypt))
		if err != nil {
			return nil, jerr.Get("error decrypting", err)
		}
		decryptedKey := string(decryptedKeyByte)
		walletKey, err := wallet.ImportPrivateKey(decryptedKey)
		if err != nil {
			return nil, jerr.Get("error importing private key", err)
		}
		//check the balance of the new key
		inputGetter := tweetWallet.InputGetter{
			Address: walletKey.GetAddress(),
		}
		outputs, err := inputGetter.GetUTXOs(nil)
		if err != nil {
			return nil, jerr.Get("error getting utxos", err)
		}
		//if the balance is greater than 800, add the twitterName and newKey to the streamArray
		balance := int64(0)
		for _, output := range outputs {
			balance += output.Input.Value
		}
		if balance > 800 {
			wlt := tweetWallet.NewWallet(walletKey.GetAddress(), walletKey)
			streamArray = append(streamArray, config.Stream{Key: decryptedKey, Name: twitterName, Sender: senderAddress, Wallet: wlt})
		}
	}
	iter.Release()
	err := iter.Error()
	if err != nil {
		return nil, jerr.Get("error iterating", err)
	}
	return streamArray, nil
}
