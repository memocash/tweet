package bot

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/hasura/go-graphql-client"
	"github.com/hasura/go-graphql-client/pkg/jsonutil"
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/index/ref/bitcoin/memo"
	"github.com/memocash/index/ref/bitcoin/tx/hs"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/config"
	"github.com/memocash/tweet/database"
	"github.com/memocash/tweet/tweets"
	"github.com/memocash/tweet/tweets/obj"
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
	Key         wallet.PrivateKey
	TweetClient *twitter.Client
	Db          *leveldb.DB
	Stream      *tweets.Stream
	ErrorChan   chan error
	Mutex       sync.Mutex
	Crypt       string
	Timer       *time.Timer
}

func NewBot(mnemonic *wallet.Mnemonic, addresses []string, key wallet.PrivateKey, tweetClient *twitter.Client, db *leveldb.DB) *Bot {
	return &Bot{
		Mnemonic:    mnemonic,
		Addresses:   addresses,
		Key:         key,
		TweetClient: tweetClient,
		Db:          db,
		ErrorChan:   make(chan error),
	}
}

func (b *Bot) Listen() error {
	println("Listening to address: " + b.Addresses[0])
	client := graphql.NewSubscriptionClient("ws://127.0.0.1:26770/graphql")
	defer client.Close()
	var subscription = new(Subscription)
	client.OnError(func(sc *graphql.SubscriptionClient, err error) error {
		b.ErrorChan <- jerr.Get("error in client subscription", err)
		return nil
	})
	_, err := client.Subscribe(&subscription, map[string]interface{}{"addresses": b.Addresses}, b.ReceiveNewTx)
	if err != nil {
		return jerr.Get("error subscribing to graphql", err)
	}
	fmt.Println("Listening for memos...")
	//client.WithLog(log.Println)
	updateInterval := config.GetConfig().UpdateInterval
	if updateInterval == 0 {
		updateInterval = 180
		}
	go func() {
		if err = client.Run(); err != nil {
			b.ErrorChan <- jerr.Get("error running graphql client", err)
		}
	}()
	go func() {
		for {
			t := time.NewTimer(time.Duration(updateInterval) * time.Minute)
			select {
			case <-t.C:
			}
			streamArray, err := makeStreamArray(b)
			if err != nil {
				b.ErrorChan <- jerr.Get("error making stream array", err)
			}
			err = updateProfiles(streamArray, b)
			if err != nil {
				fmt.Printf("\n\nError updating profiles: %s\n\n", err.Error())
			}
		}
	}()
	if b.Stream, err = tweets.NewStream(b.Db); err != nil {
		return jerr.Get("error getting new tweet stream", err)
	}
	err = b.SafeUpdate()
	if err != nil {
		return jerr.Get("error updating stream", err)
	}
	return jerr.Get("error in listen", <-b.ErrorChan)
}

func (b *Bot) ReceiveNewTx(dataValue []byte, errValue error) error {
	b.Mutex.Lock()
	defer b.Mutex.Unlock()
	if errValue != nil {
		return jerr.Get("error in subscription", errValue)
	}
	data := Subscription{}
	err := jsonutil.UnmarshalGraphQL(dataValue, &data)
	if err != nil {
		return jerr.Get("error marshaling subscription", err)
	}
	for _, input := range data.Addresses.Inputs {
		if input.Output.Lock.Address == b.Addresses[0] {
			return nil
		}
	}
	var scriptArray []string
	for _, output := range data.Addresses.Outputs {
		scriptArray = append(scriptArray, output.Script)
	}
	message := grabMessage(scriptArray)
	senderAddress := ""
	for _, input := range data.Addresses.Inputs {
		if input.Output.Lock.Address != b.Addresses[0] {
			senderAddress = input.Output.Lock.Address
			break
		}
	}
	coinIndex := uint32(0)
	for i, output := range data.Addresses.Outputs {
		if output.Lock.Address == b.Addresses[0] {
			coinIndex = uint32(i)
			break
		}
	}
	defer func() {
		//add the transaction hash to the database
		if err := b.Db.Put([]byte("completed-"+data.Addresses.Hash), []byte{}, nil); err != nil {
			b.ErrorChan <- jerr.Get("error adding tx hash to database", err)
		}
	}()
	//check if the transaction is already in the database
	if _, err := b.Db.Get([]byte("completed-"+data.Addresses.Hash), nil); err == nil {
		println("Already completed tx: " + data.Addresses.Hash)
		return nil
	}

	match, _ := regexp.MatchString("^CREATE TWITTER ([a-zA-Z0-9_]{1,15})(( --history( [0-9]+)?)?( --nolink)?( --date)?)*$", message)
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
			err := refund(data, b, coinIndex, senderAddress, "There are too many streams running, please try again later")
			if err != nil {
				return jerr.Get("error refunding", err)
			}
			return nil
		}
		//split the message into an array of strings
		splitMessage := strings.Split(message, " ")
		//get the twitter name from the message
		twitterName := splitMessage[2]
		//check if --history is in the message
		history := false
		link := true
		date := false
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
				link = false
			}
			if word == "--date" {
				date = true
			}
		}
		if historyNum > 1000 {
			err = refund(data, b, coinIndex, senderAddress, "Number of tweets must be less than 1000")
			if err != nil {
				return jerr.Get("error refunding", err)
			}
			return nil
		}
		//write date and link into flags-senderAddress-twitterName
		type Flags struct {
			Link bool `json:"link"`
			Date bool `json:"date"`
		}
		flags := Flags{Link: link, Date: date}
		flagsBytes, err := json.Marshal(flags)
		if err != nil {
			return jerr.Get("error marshaling flags", err)
		}
		if err := b.Db.Put([]byte("flags-"+senderAddress+"-"+twitterName), flagsBytes, nil); err != nil {
			return jerr.Get("error putting flags into database", err)
		}
		accountKeyPointer, wlt,  err := createBot(b, twitterName, senderAddress, data, coinIndex)
		if err != nil {
			return jerr.Get("error creating bot", err)
		}
		//transfer all the tweets from the twitter account to the new bot
		if accountKeyPointer != nil {
			accountKey := *accountKeyPointer
			if history {
				client := tweets.Connect()
				err = tweets.GetSkippedTweets(accountKey, wlt, client, b.Db, link, date, historyNum)
				if err != nil {
					return jerr.Get("error getting skipped tweets", err)
				}

			}
			err = b.SafeUpdate()
			if err != nil {
				return jerr.Get("error updating stream", err)
			}
		} else {
			println("account key pointer is nil, not transferring tweets")
			return nil
		}
	} else if regexp.MustCompile("^WITHDRAW TWITTER ([a-zA-Z0-9_]{1,15})( [0-9]+)?$").MatchString(message) {
		//check the database for each field that matches linked-<senderAddress>-<twitterName>
		//if there is a match, print out the address and key
		//if there is no match, print out an error message
		twitterName := regexp.MustCompile("^WITHDRAW TWITTER ([a-zA-Z0-9_]{1,15})( [0-9]+)?$").FindStringSubmatch(message)[1]
		searchString := "linked-" + senderAddress + "-" + twitterName
		//refund if this field doesn't exist
		searchValue, err := b.Db.Get([]byte(searchString), nil)
		if err != nil {
			if err == leveldb.ErrNotFound {
				//handle refund
				errMsg := "No linked address found for " + senderAddress + "-" + twitterName
				err = refund(data, b, coinIndex, senderAddress, errMsg)
				if err != nil {
					return jerr.Get("error refunding", err)
				}
				return nil
			}
			errMsg := "Error accessing database looking for existing linked address"
			_ = refund(data, b, coinIndex, senderAddress, errMsg)
			return jerr.Get("error getting linked-"+senderAddress+"-"+twitterName, err)
		} else {
			fieldName := searchString
			stringKey := searchValue
			println("Field name: " + string(fieldName))
			//get the address object from the private key
			decryptedKey, err := database.Decrypt(stringKey, []byte(b.Crypt))
			if err != nil {
				return jerr.Get("error decrypting key", err)
			}
			key, err := wallet.ImportPrivateKey(string(decryptedKey))
			if err != nil {
				return jerr.Get("error importing private key", err)
			}
			address := key.GetAddress()
			println("Address: " + address.GetEncoded())
			inputGetter := database.InputGetter{
				Address: address,
				UTXOs:   nil,
				Db:      b.Db,
			}
			//use the address object of the spawned key to get the outputs array
			outputs, err := inputGetter.GetUTXOs(nil)
			if err != nil {
				return jerr.Get("error getting utxos", err)
			}
			//check if the message contains a number
			var amount int64
			var maxSend = memo.GetMaxSendForUTXOs(outputs)
			if regexp.MustCompile("^WITHDRAW TWITTER ([a-zA-Z0-9_]{1,15}) [0-9]+$").MatchString(message) {
				amount, _ = strconv.ParseInt(regexp.MustCompile("^WITHDRAW TWITTER ([a-zA-Z0-9_]{1,15}) ([0-9]+)$").FindStringSubmatch(message)[2], 10, 64)
				if amount > maxSend{
					err = refund(data, b, coinIndex, senderAddress, "Cannot withdraw more than the total balance is capable of sending")
					if err != nil {
						return jerr.Get("error refunding", err)
					}
					return nil
				} else if amount+memo.DustMinimumOutput+memo.OutputFeeP2PKH > maxSend {
					errmsg := fmt.Sprintf("Not enough funds will be left over to send change to bot account, please withdraw less than %d", maxSend+1-memo.DustMinimumOutput-memo.OutputFeeP2PKH)
					err = refund(data, b, coinIndex, senderAddress,errmsg)
					if err != nil {
						return jerr.Get("error refunding", err)
					}
					return nil
				} else {
					err := database.WithdrawAmount(outputs, key, wallet.GetAddressFromString(senderAddress), amount)
					if err != nil {
						return jerr.Get("error withdrawing amount", err)
					}
				}
			} else {
				if maxSend > 0 {
					err := database.WithdrawAll(outputs, key, wallet.GetAddressFromString(senderAddress))
					if err != nil {
						return jerr.Get("error withdrawing all", err)
					}
				} else {
					err = refund(data, b, coinIndex, senderAddress, "Not enough balance to withdraw anything")
					if err != nil {
						return jerr.Get("error refunding", err)
					}
					return nil
				}
			}
		}
		err = b.SafeUpdate()
		if err != nil {
			return jerr.Get("error updating stream", err)
		}
	} else {
		errMsg := "Invalid command. Please use the following format: CREATE TWITTER <twitterName> --history <numTweets> or WITHDRAW TWITTER <twitterName>"
		err = refund(data, b, coinIndex, senderAddress, errMsg)
		if err != nil {
			return jerr.Get("error refunding", err)
		}
	}
	return nil
}
func refund(data Subscription, b *Bot, coinIndex uint32, senderAddress string, errMsg string) error {
	err := b.SafeUpdate()
	if err != nil {
		return jerr.Get("error updating stream", err)
	}
	fmt.Printf("\n\nSending error message: %s\n\n", errMsg)
	sentToMainBot := false
	//check all the outputs to see if any of them match the bot's address, if not, return nil, if so, continue with the function
	for _, output := range data.Addresses.Outputs {
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
	if memo.GetMaxSendFromCount(data.Addresses.Outputs[coinIndex].Amount, 1) <= 0 {
		println("Not enough funds to refund")
		return nil
	}
	//create a transaction with the sender address and the amount of the transaction
	pkScript, err := hex.DecodeString(data.Addresses.Outputs[coinIndex].Script)
	if err != nil {
		return jerr.Get("error decoding script pk script for create bot", err)
	}
	if err := database.SendToTwitterAddress(memo.UTXO{Input: memo.TxInput{
		Value:        data.Addresses.Outputs[coinIndex].Amount,
		PrevOutHash:  hs.GetTxHash(data.Addresses.Hash),
		PrevOutIndex: coinIndex,
		PkHash:       wallet.GetAddressFromString(b.Addresses[0]).GetPkHash(),
		PkScript:     pkScript,
	}}, b.Key, wallet.GetAddressFromString(senderAddress), errMsg); err != nil {
		return jerr.Get("error sending money back", err)
	}
	return nil
}
func (b *Bot) SafeUpdate() error {
	err := b.UpdateStream()
	var waitCount = 1
	for err != nil && waitCount < 30 {
		if !jerr.HasErrorPart(err, "DuplicateRule") {
			return jerr.Get("error updating stream", err)
		}
		fmt.Printf("\n\n\nError updating stream: %s\n\n\n", err.Error())
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
	streamArray, err := makeStreamArray(b)
	if err != nil {
		return jerr.Get("error making stream array", err)
	}
	for _, stream := range streamArray {
		streamKey, err := wallet.ImportPrivateKey(stream.Key)
		if err != nil {
			return jerr.Get("error importing private key", err)
		}
		streamAddress := streamKey.GetAddress()
		println("streaming " + stream.Name + " to address " + streamAddress.GetEncoded())
	}
	err = b.Db.Put([]byte("memobot-running-count"), []byte(strconv.FormatUint(uint64(len(streamArray)), 10)), nil)
	if err != nil {
		return jerr.Get("error updating running count", err)
	}
	go func() {
		if len(streamArray) == 0 {
			return
		}
		if err := b.Stream.InitiateStream(streamArray); err != nil {
			b.ErrorChan <- jerr.Get("error twitter initiate stream in update", err)
		}
	}()
	return nil
}

func createBot(b *Bot, twitterName string, senderAddress string, data Subscription, coinIndex uint32) (*obj.AccountKey, *database.Wallet, error) {
	//check if the value of the transaction is less than 5,000 or this address already has a bot for this account in the database
	botExists := false
	_, err := b.Db.Get([]byte("linked-"+senderAddress+"-"+twitterName), nil)
	if err != nil && err != leveldb.ErrNotFound {
		return nil, nil, jerr.Get("error getting bot from database", err)
	} else if err != leveldb.ErrNotFound {
		botExists = true
	}
	//check if this twitter account actually exists
	twitterExists := false
	if _, _, err := b.TweetClient.Users.Show(&twitter.UserShowParams{ScreenName: twitterName}); err == nil {
		twitterExists = true
	}
	if !twitterExists || data.Addresses.Outputs[coinIndex].Amount < 5000 {
		if data.Addresses.Outputs[coinIndex].Amount < 546 {
			return nil, nil, nil
		}
		errMsg := ""
		if !twitterExists {
			errMsg = fmt.Sprintf("Twitter account @%s does not exist", twitterName)
		} else {
			errMsg = fmt.Sprintf("You need to send at least 5,000 satoshis to create a bot for the account @%s", twitterName)
		}
		print("\n\n\nSending error message: " + errMsg + "\n\n\n")
		if err != nil {
			return nil, nil, jerr.Get("error decoding script pk script for create bot", err)
		}
		err = refund(data, b, coinIndex, senderAddress, errMsg)
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
		return nil,nil, jerr.Get("error parsing num-streams", err)
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
		decryptedKey, err := database.Decrypt(rawKey, []byte(b.Crypt))
		if err != nil {
			return nil,nil, jerr.Get("error decrypting key", err)
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
	pkScript, err := hex.DecodeString(data.Addresses.Outputs[coinIndex].Script)
	if err != nil {
		return nil, nil, jerr.Get("error decoding script pk script for create bot", err)
	}
	if err := database.FundTwitterAddress(memo.UTXO{Input: memo.TxInput{
		Value:        data.Addresses.Outputs[coinIndex].Amount,
		PrevOutHash:  hs.GetTxHash(data.Addresses.Hash),
		PrevOutIndex: coinIndex,
		PkHash:       b.Key.GetAddress().GetPkHash(),
		PkScript:     pkScript,
	}}, b.Key, newAddr); err != nil {
		return nil, nil, jerr.Get("error funding twitter address", err)
	}
	newWallet := database.NewWallet(newAddr, newKey, b.Db)
	if !botExists {
		err = updateProfile(b, newWallet, twitterName, senderAddress)
		if err != nil {
			return nil, nil, jerr.Get("error updating profile", err)
		}
	}
	println("Stream Address: " + newAddr.GetEncoded())
	if !botExists {
		err = b.Db.Put([]byte("memobot-num-streams"), []byte(strconv.FormatUint(uint64(numStreamUint+1), 10)), nil)
		if err != nil {
			return nil, nil, jerr.Get("error putting num-streams", err)
		}
		//add a field to the database that links the sending address and twitter name to the new key
		//encrypt
		encryptedKey, err := database.Encrypt([]byte(newKey.GetBase58Compressed()), []byte(b.Crypt))
		if err != nil {
			return nil, nil, jerr.Get("error encrypting key", err)
		}
		err = b.Db.Put([]byte("linked-"+senderAddress+"-"+twitterName), encryptedKey, nil)
		if err != nil {
			return nil, nil, jerr.Get("error updating linked-"+senderAddress+"-"+twitterName, err)
		}
	}
	println("done")
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
		newWallet := database.NewWallet(streamAddress, streamKey, b.Db)
		err = updateProfile(b, newWallet, stream.Name, stream.Sender)
		time.Sleep(1 * time.Second)
	}
	return nil
}
func updateProfile(b *Bot, newWallet database.Wallet, twitterName string, senderAddress string) error {
	profile, err := tweets.GetProfile(twitterName, b.TweetClient)
	if err != nil {
		return jerr.Get("fatal error getting profile", err)
	}
	//look for the profile-senderAddress-twitterName key in the database
	profileExists, err := b.Db.Get([]byte("profile-"+senderAddress+"-"+twitterName), nil)
	if err != nil && err != leveldb.ErrNotFound {
		return jerr.Get("error getting profile from database", err)
	}
	if err == leveldb.ErrNotFound {
		err = database.UpdateName(newWallet, profile.Name)
		if err != nil {
			return jerr.Get("error updating name", err)
		} else {
			println("updated name for the first time")
		}
		err = database.UpdateProfileText(newWallet, profile.Description)
		if err != nil {
			return jerr.Get("error updating profile text", err)
		} else {
			println("updated profile text for the first time")
		}
		err = database.UpdateProfilePic(newWallet, profile.ProfilePic)
		if err != nil {
			return jerr.Get("error updating profile pic", err)
		} else {
			println("updated profile pic for the first time")
		}
		newProfile := database.Profile{
			Name:        profile.Name,
			Description: profile.Description,
			ProfilePic:  profile.ProfilePic,
		}
		profileBytes, err := json.Marshal(newProfile)
		if err != nil {
			return jerr.Get("error marshalling profile", err)
		}
		err = b.Db.Put([]byte("profile-"+senderAddress+"-"+twitterName), profileBytes, nil)
		if err != nil {
			return jerr.Get("error putting profile in database", err)
		}
	}else if err == nil {
		//make the profileExists into a string and print it
		var dbProfile database.Profile
		err = json.Unmarshal(profileExists, &dbProfile)
		if err != nil {
			println("Profile name: " + dbProfile.Name)
			return jerr.Get("error unmarshalling profile", err)
		}
		if dbProfile.Name != profile.Name {
			err = database.UpdateName(newWallet, profile.Name)
			if err != nil {
				return jerr.Get("error updating name", err)
			} else {
				println("updated name")
			}
			dbProfile.Name = profile.Name
		}
		if dbProfile.Description != profile.Description {
			err = database.UpdateProfileText(newWallet, profile.Description)
			if err != nil {
				return jerr.Get("error updating profile text", err)
			} else {
				println("updated profile text")
			}
			dbProfile.Description = profile.Description
		}
		if dbProfile.ProfilePic != profile.ProfilePic {
			err = database.UpdateProfilePic(newWallet, profile.ProfilePic)
			if err != nil {
				return jerr.Get("error updating profile pic", err)
			} else {
				println("updated profile pic")
			}
			dbProfile.ProfilePic = profile.ProfilePic
		}
		profileBytes, err := json.Marshal(profile)
		if err != nil {
			return jerr.Get("error marshalling profile", err)
		}
		err = b.Db.Put([]byte("profile-"+senderAddress+"-"+twitterName), profileBytes, nil)
		if err != nil {
			return jerr.Get("error putting profile", err)
		}
	}
	println("checked for updates")
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
		decryptedKeyByte, err := database.Decrypt(iter.Value(), []byte(b.Crypt))
		if err != nil {
			return nil, jerr.Get("error decrypting", err)
		}
		decryptedKey := string(decryptedKeyByte)
		walletKey, err := wallet.ImportPrivateKey(decryptedKey)
		if err != nil {
			return nil, jerr.Get("error importing private key", err)
		}
		//check the balance of the new key
		inputGetter := database.InputGetter{
			Address: walletKey.GetAddress(),
			UTXOs:   nil,
			Db:      b.Db,
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
			wlt := database.NewWallet(walletKey.GetAddress(), walletKey, b.Db)
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
