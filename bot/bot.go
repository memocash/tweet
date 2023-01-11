package bot

import (
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
	println("Listening to key: " + b.Key.GetBase58Compressed())
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
	if b.Stream, err = tweets.NewStream(b.Db); err != nil {
		return jerr.Get("error getting new tweet stream", err)
	}
	if err := b.UpdateStream(); err != nil {
		return jerr.Get("error updating stream bot start", err)
	}
	fmt.Println("Listening for memos...")
	//client.WithLog(log.Println)
	go func() {
		if err = client.Run(); err != nil {
			b.ErrorChan <- jerr.Get("error running graphql client", err)
		}
	}()
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
					historyNum,err = strconv.Atoi(splitMessage[index+1])
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
		accountKeyPointer, err := createBot(b, twitterName, senderAddress, data, coinIndex)
		if err != nil {
			return jerr.Get("error creating bot", err)
		}
		//transfer all the tweets from the twitter account to the new bot
		if accountKeyPointer != nil {
			accountKey := *accountKeyPointer
			if history {
				client := tweets.Connect()
				err = tweets.GetSkippedTweets(accountKey, client, b.Db, link, date, historyNum)
				if err != nil {
					return jerr.Get("error getting skipped tweets", err)
				}

			}
			err = b.UpdateStream()
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
		searchValue,err := b.Db.Get([]byte(searchString), nil)
		if err != nil {
			if err == leveldb.ErrNotFound {
				//handle refund
				errMsg := "No linked address found for " + senderAddress + "-"+ twitterName
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
			println("Private Key: " + string(stringKey))
			//get the address object from the private key
			key,err := wallet.ImportPrivateKey(string(stringKey))
			if err != nil {
				return jerr.Get("error importing private key", err)
			}
			address := key.GetAddress()
			println("Address: " + address.GetEncoded())
			inputGetter := database.InputGetter{
				Address: address,
				UTXOs:   nil,
			}
			//use the address object of the spawned key to get the outputs array
			outputs,err := inputGetter.GetUTXOs(nil)
			if err != nil {
				return jerr.Get("error getting utxos", err)
			}
			//check if the message contains a number
			var amount int64
			var totalBalance int64 = 0
			for _, output := range outputs {
				if output.Input.Value < 546 {
					continue
				}
				totalBalance += output.Input.Value
			}
			if regexp.MustCompile("^WITHDRAW TWITTER ([a-zA-Z0-9_]{1,15}) [0-9]+$").MatchString(message) {
				amount, _ = strconv.ParseInt(regexp.MustCompile("^WITHDRAW TWITTER ([a-zA-Z0-9_]{1,15}) ([0-9]+)$").FindStringSubmatch(message)[2], 10, 64)
				if amount > totalBalance {
					err = refund(data, b, coinIndex, senderAddress, "Cannot withdraw more than the total balance")
					if err != nil {
						return jerr.Get("error refunding", err)
					}
					return nil
				}
			} else {
				amount = totalBalance
			}
			for _,output := range outputs {
				if output.Input.Value < 546 {
					continue
				}
				if amount <= 0 {
					break
				}
				var value int64
				if output.Input.Value >= amount {
					value = amount
				}
				if output.Input.Value < amount {
					value = output.Input.Value
				}
				if err := database.PartialFund(memo.UTXO{Input: memo.TxInput{
					Value:        output.Input.Value,
					PrevOutHash:  output.Input.PrevOutHash,
					PrevOutIndex: output.Input.PrevOutIndex,
					PkHash:       output.Input.PkHash,
				}}, key, wallet.GetAddressFromString(senderAddress), value); err != nil {
					return jerr.Get("error sending funds back", err)
				}
				amount -= value
			}
		}
		err = b.UpdateStream()
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
	err := b.UpdateStream()
	if err != nil {
		return jerr.Get("error updating stream", err)
	}
	fmt.Printf("\n\nSending error message: %s\n\n", errMsg)
	sentToMainBot := false
	//check all the outputs to see if any of them match the bot's address, if not, return nil, if so, continue with the function
	for _,output := range data.Addresses.Outputs {
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
	if data.Addresses.Outputs[coinIndex].Amount < 546 {
		return nil
	}
	//create a transaction with the sender address and the amount of the transaction
	if err := database.SendToTwitterAddress(memo.UTXO{Input: memo.TxInput{
		Value:        data.Addresses.Outputs[coinIndex].Amount,
		PrevOutHash:  hs.GetTxHash(data.Addresses.Hash),
		PrevOutIndex: coinIndex,
		PkHash:       wallet.GetAddressFromString(b.Addresses[0]).GetPkHash(),
	}}, b.Key, wallet.GetAddressFromString(senderAddress), errMsg); err != nil {
		return jerr.Get("error sending money back", err)
	}
	return nil
}
func (b *Bot) UpdateStream() error {
	//create an array of {twitterName, newKey} objects by searching through the linked-<senderAddress>-<twitterName> fields
	streamArray := make([]config.Stream, 0)
	iter := b.Db.NewIterator(util.BytesPrefix([]byte("linked-")), nil)
	for iter.Next() {
		//find the twitterName at the end of the linked-<senderAddress>-<twitterName> field
		senderAddress := strings.Split(string(iter.Key()), "-")[1]
		twitterName := strings.Split(string(iter.Key()), "-")[2]
		newKey := string(iter.Value())
		walletKey,err := wallet.ImportPrivateKey(newKey)
		if err != nil {
			return jerr.Get("error importing private key", err)
		}
		//check the balance of the new key
		inputGetter := database.InputGetter{
			Address: walletKey.GetAddress(),
			UTXOs:   nil,
		}
		outputs,err := inputGetter.GetUTXOs(nil)
		if err != nil {
			return jerr.Get("error getting utxos", err)
		}
		//if the balance is greater than 800, add the twitterName and newKey to the streamArray
		balance := int64(0)
		for _,output := range outputs {
			balance += output.Input.Value
		}
		if balance > 800 {
			streamArray = append(streamArray, config.Stream{Key: newKey, Name: twitterName, Sender: senderAddress})
		}
	}
	for _, stream := range streamArray {
		println("streaming " + stream.Name + " to key " + stream.Key)
	}
	iter.Release()
	if err := iter.Error(); err != nil {
		return jerr.Get("error iterating through database", err)
	}
	err := b.Db.Put([]byte("memobot-running-count"), []byte(strconv.FormatUint(uint64(len(streamArray)), 10)), nil)
	if err != nil {
		return jerr.Get("error updating running count", err)
	}
	go func() {
		if err := b.Stream.InitiateStream(streamArray); err != nil {
			b.ErrorChan <- jerr.Get("error twitter initiate stream in update", err)
		}
	}()
	return nil
}

func createBot(b *Bot, twitterName string, senderAddress string, data Subscription, coinIndex uint32) (*obj.AccountKey, error) {
	//check if the value of the transaction is less than 5,000 or this address already has a bot for this account in the database
	botExists := false
	_, err := b.Db.Get([]byte("linked-"+senderAddress+"-"+twitterName), nil)
	if err != nil && err != leveldb.ErrNotFound {
		return nil, jerr.Get("error getting bot from database", err)
	}else if err != leveldb.ErrNotFound{
		botExists = true
	}
	//check if this twitter account actually exists
	twitterExists := false
	if _, _, err := b.TweetClient.Users.Show(&twitter.UserShowParams{ScreenName: twitterName}); err == nil {
		twitterExists = true
	}
	if !twitterExists || data.Addresses.Outputs[coinIndex].Amount < 5000 {
		if data.Addresses.Outputs[coinIndex].Amount < 546 {
			return nil, nil
		}
		errMsg := ""
		if !twitterExists {
			errMsg = fmt.Sprintf("Twitter account @%s does not exist", twitterName)
		}else {
			errMsg = fmt.Sprintf("You need to send at least 5,000 satoshis to create a bot for the account @%s", twitterName)
		}
		print("\n\n\nSending error message: " + errMsg + "\n\n\n")
		if err := database.SendToTwitterAddress(memo.UTXO{Input: memo.TxInput{
			Value:        data.Addresses.Outputs[coinIndex].Amount,
			PrevOutHash:  hs.GetTxHash(data.Addresses.Hash),
			PrevOutIndex: coinIndex,
			PkHash:       wallet.GetAddressFromString(b.Addresses[0]).GetPkHash(),
		}}, b.Key, wallet.GetAddressFromString(senderAddress), errMsg); err != nil {
			return nil, jerr.Get("error sending money back", err)
		}
		return nil, nil
	}
	println(b.Addresses[0])
	var newKey wallet.PrivateKey
	var newAddr wallet.Address
	numStreamBytes, err := b.Db.Get([]byte("memobot-num-streams"), nil)
	if err != nil {
		return nil, jerr.Get("error getting num-streams", err)
	}
	numStream, err := strconv.ParseUint(string(numStreamBytes), 10, 64)
	if err != nil {
		return nil, jerr.Get("error parsing num-streams", err)
	}
	//convert numStream to a uint
	numStreamUint := uint(numStream)
	if botExists {
		//get the key from the database
		rawKey,err := b.Db.Get([]byte("linked-"+senderAddress+"-"+twitterName), nil)
		if err != nil {
			return nil, jerr.Get("error getting key from database", err)
		}
		newKey,err = wallet.ImportPrivateKey(string(rawKey))
		if err != nil {
			return nil, jerr.Get("error importing private key", err)
		}
		newAddr = newKey.GetAddress()
	}else{
		path := wallet.GetBip44Path(wallet.Bip44CoinTypeBTC, numStreamUint+1, false)
		keyPointer, err := b.Mnemonic.GetPath(path)
		newKey = *keyPointer
		if err != nil {
			return nil, jerr.Get("error getting path", err)
		}
		newAddr = newKey.GetAddress()
	}
	if err := database.FundTwitterAddress(memo.UTXO{Input: memo.TxInput{
		Value:        data.Addresses.Outputs[coinIndex].Amount,
		PrevOutHash:  hs.GetTxHash(data.Addresses.Hash),
		PrevOutIndex: coinIndex,
		PkHash:       wallet.GetAddressFromString(b.Addresses[0]).GetPkHash(),
	}}, b.Key, newAddr); err != nil {
		return nil, jerr.Get("error funding twitter address", err)
	}
	if !botExists {
		newWallet := database.NewWallet(newAddr, newKey)

		profile, err := tweets.GetProfile(twitterName, b.TweetClient)
		if err != nil {
			return nil, jerr.Get("fatal error getting profile", err)
		}
		fmt.Printf("Name: %s\nDesc: %s\nProfile Pic Link: %s\n",
			profile.Name, profile.Description, profile.ProfilePic)
		err = database.UpdateName(newWallet, profile.Name)
		if err != nil {
			return nil, jerr.Get("error updating name", err)
		} else {
			println("updated name")
		}
		err = database.UpdateProfileText(newWallet, profile.Description)
		if err != nil {
			return nil, jerr.Get("error updating profile text", err)
		} else {
			println("updated profile text")
		}
		err = database.UpdateProfilePic(newWallet, profile.ProfilePic)
		if err != nil {
			return nil, jerr.Get("error updating profile pic", err)
		} else {
			println("updated profile pic")
		}
	}
	println("Stream Key: " + newKey.GetBase58Compressed())
	println("Stream Address: " + newAddr.GetEncoded())
	if !botExists{
		err = b.Db.Put([]byte("memobot-num-streams"), []byte(strconv.FormatUint(uint64(numStreamUint+1), 10)), nil)
		if err != nil {
			return nil, jerr.Get("error putting num-streams", err)
		}
		//add a field to the database that links the sending address and twitter name to the new key
		err = b.Db.Put([]byte("linked-"+senderAddress+"-"+twitterName), []byte(newKey.GetBase58Compressed()), nil)
		if err != nil {
			return nil, jerr.Get("error updating linked-"+senderAddress+"-"+twitterName, err)
		}
	}
	println("done")
	accountKey := obj.GetAccountKeyFromArgs([]string{newKey.GetBase58Compressed(), twitterName})
	return &accountKey, nil
}
