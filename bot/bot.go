package bot

import (
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
		}
	}
	coinIndex := uint32(0)
	for i, output := range data.Addresses.Outputs {
		if output.Lock.Address == b.Addresses[0] {
			coinIndex = uint32(i)
		}
	}
	//use regex library to check if message matches the format "CREATE TWITTER {twittername}" tweet names are a maximum of 15 characters
	match, _ := regexp.MatchString("^CREATE TWITTER \\{[a-zA-Z0-9_]{1,15}}$", message)
	if match {
		twitterName := regexp.MustCompile("^CREATE TWITTER \\{([a-zA-Z0-9_]{1,15})}$").FindStringSubmatch(message)[1]
		//check if the value of the transaction is less than 5,000 or this address already has a bot for this account in the database
		botExists := false
		iter := b.Db.NewIterator(util.BytesPrefix([]byte("linked-"+senderAddress+"-"+twitterName)), nil)
		for iter.Next() {
			botExists = true
		}
		if botExists || data.Addresses.Outputs[coinIndex].Amount < 5000 {
			if data.Addresses.Outputs[coinIndex].Amount < 546 {
				return nil
			}
			errMsg := ""
			if botExists {
				errMsg = fmt.Sprintf("You already have a bot for the account @%s", twitterName)
			} else {
				errMsg = fmt.Sprintf("You need to send at least 5,000 satoshis to create a bot for the account @%s", twitterName)
			}
			print("\n\n\nSending error message: " + errMsg + "\n\n\n")
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
		fmt.Printf("\n\n%s\n\n", message)
		println(b.Addresses[0])
		numStreamBytes, err := b.Db.Get([]byte("memobot-num-streams"), nil)
		if err != nil {
			return jerr.Get("error getting num-streams", err)
		}
		numStream, err := strconv.ParseUint(string(numStreamBytes), 10, 64)
		if err != nil {
			return jerr.Get("error parsing num-streams", err)
		}
		//convert numStream to a uint
		numStreamUint := uint(numStream)
		path := wallet.GetBip44Path(wallet.Bip44CoinTypeBTC, numStreamUint+1, false)
		newKey, err := b.Mnemonic.GetPath(path)
		if err != nil {
			return jerr.Get("error getting path", err)
		}
		newAddr := newKey.GetAddress()
		println("New Key: " + newKey.GetBase58Compressed())
		println("New Address: " + newAddr.GetEncoded())

		if err := database.FundTwitterAddress(memo.UTXO{Input: memo.TxInput{
			Value:        data.Addresses.Outputs[coinIndex].Amount,
			PrevOutHash:  hs.GetTxHash(data.Addresses.Hash),
			PrevOutIndex: coinIndex,
			PkHash:       wallet.GetAddressFromString(b.Addresses[0]).GetPkHash(),
		}}, b.Key, newAddr); err != nil {
			return jerr.Get("error funding twitter address", err)
		}
		newWallet := database.NewWallet(newAddr, *newKey)
		profile, err := tweets.GetProfile(twitterName, b.TweetClient)
		if err != nil {
			return jerr.Get("fatal error getting profile", err)
		}
		fmt.Printf("Name: %s\nDesc: %s\nProfile Pic Link: %s\n",
			profile.Name, profile.Description, profile.ProfilePic)
		err = database.UpdateName(newWallet, profile.Name)
		if err != nil {
			return jerr.Get("error updating name", err)
		} else {
			println("updated name")
		}
		err = database.UpdateProfileText(newWallet, profile.Description)
		if err != nil {
			return jerr.Get("error updating profile text", err)
		} else {
			println("updated profile text")
		}
		err = database.UpdateProfilePic(newWallet, profile.ProfilePic)
		if err != nil {
			return jerr.Get("error updating profile pic", err)
		} else {
			println("updated profile pic")
		}
		println("New Key: " + newKey.GetBase58Compressed())
		println("New Address: " + newAddr.GetEncoded())
		err = b.Db.Put([]byte("memobot-num-streams"), []byte(strconv.FormatUint(uint64(numStreamUint+1), 10)), nil)
		if err != nil {
			return jerr.Get("error updating memobot-num-streams", err)
		}
		//add a field to the database that links the sending address and twitter name to the new key
		err = b.Db.Put([]byte("linked-"+senderAddress+"-"+twitterName), []byte(newKey.GetBase58Compressed()), nil)
		if err != nil {
			return jerr.Get("error updating linked-"+senderAddress+"-"+twitterName, err)
		}
		if err := b.UpdateStream(); err != nil {
			return jerr.Get("error updating stream after new tx", err)
		}
		println("done")
	} else if regexp.MustCompile("^WITHDRAW TWITTER \\{([a-zA-Z0-9_]{1,15})}$").MatchString(message) {
		//check the database for each field that matches linked-<senderAddress>-<twitterName>
		//if there is a match, print out the address and key
		//if there is no match, print out an error message
		twitterName := regexp.MustCompile("^WITHDRAW TWITTER \\{([a-zA-Z0-9_]{1,15})}$").FindStringSubmatch(message)[1]
		searchString := "linked-" + senderAddress + "-" + twitterName
		//refund if this field doesn't exist
		println("here")
		searchValue,err := b.Db.Get([]byte(searchString), nil)
		if err != nil {
			if err == leveldb.ErrNotFound {
				//handle refund
				errMsg := "No linked address found for " + senderAddress + "-" + twitterName
				print("\n\n\nSending error message: " + errMsg + "\n\n\n")
				if data.Addresses.Outputs[coinIndex].Amount < 546 {
					println("not enough money to send error message")
					return nil
				}
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
			//use the outputs array to get the amount (the function only returns outputs without a spend)
			for _,output := range outputs {
				if err := database.FundTwitterAddress(memo.UTXO{Input: memo.TxInput{
					Value:        output.Input.Value,
					PrevOutHash:  output.Input.PrevOutHash,
					PrevOutIndex: output.Input.PrevOutIndex,
					PkHash:       address.GetPkHash(),
				}}, key, wallet.GetAddressFromString(senderAddress)); err != nil {
					return jerr.Get("error sending funds back", err)
				}
			}
		}
		//delete the field from the database, and restart the stream without it
		err = b.Db.Delete([]byte(searchString), nil)
		if err != nil {
			return jerr.Get("error deleting "+ searchString, err)
		}
		//get num streams from database
		numStream,err := b.Db.Get([]byte("memobot-num-streams"), nil)
		if err != nil {
			return jerr.Get("error getting memobot-num-streams", err)
		}
		numStreamUint,err := strconv.ParseUint(string(numStream), 10, 64)
		if err != nil {
			return jerr.Get("error converting memobot-num-streams to uint", err)
		}
		//update the database to have 1 less stream
		err = b.Db.Put([]byte("memobot-num-streams"), []byte(strconv.FormatUint(uint64(numStreamUint-1), 10)), nil)
		if err != nil {
			return jerr.Get("error updating memobot-num-streams", err)
		}
		err = b.UpdateStream()
		if err != nil {
			return jerr.Get("error updating stream", err)
		}
	} else {
		fmt.Printf("\n\nMessage not in correct format\n\n")
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
		}}, b.Key, wallet.GetAddressFromString(senderAddress), "Message was not in correct format"); err != nil {
			return jerr.Get("error sending money back", err)
		}
		return nil
	}
	return nil
}

func (b *Bot) UpdateStream() error {
	//api := tweetstream.FetchTweets(streamToken.AccessToken)
	//api.StopStream()
	//create an array of {twitterName, newKey} objects by searching through the linked-<senderAddress>-<twitterName> fields
	streamArray := make([]config.Stream, 0)
	iter := b.Db.NewIterator(util.BytesPrefix([]byte("linked-")), nil)
	for iter.Next() {
		//find the twitterName at the end of the linked-<senderAddress>-<twitterName> field
		twitterName := strings.Split(string(iter.Key()), "-")[2]
		newKey := string(iter.Value())
		streamArray = append(streamArray, config.Stream{Key: newKey, Name: twitterName})
	}
	for _, stream := range streamArray {
		println("streaming " + stream.Name + " to key " + stream.Key)
	}
	iter.Release()
	if err := iter.Error(); err != nil {
		return jerr.Get("error iterating through database", err)
	}
	go func() {
		if err := b.Stream.InitiateStream(streamArray); err != nil {
			b.ErrorChan <- jerr.Get("error twitter initiate stream in update", err)
		}
	}()
	return nil
}
