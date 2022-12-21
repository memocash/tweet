package util

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/dghubble/go-twitter/twitter"
	twitterstream "github.com/fallenstedt/twitter-stream"
	"github.com/hasura/go-graphql-client"
	"github.com/hasura/go-graphql-client/pkg/jsonutil"
	"github.com/jchavannes/btcd/txscript"
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/index/ref/bitcoin/memo"
	"github.com/memocash/index/ref/bitcoin/tx/hs"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/config"
	"github.com/memocash/tweet/database"
	"github.com/memocash/tweet/tweets"
	"github.com/memocash/tweet/tweetstream"
	"github.com/syndtr/goleveldb/leveldb"
	util3 "github.com/syndtr/goleveldb/leveldb/util"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func TransferTweets(accountKey tweets.AccountKey, db *leveldb.DB, appendLink bool, appendDate bool) (int, error) {
	var tweetList []tweets.TweetTx
	//find the greatest ID of all the already saved tweets
	prefix := fmt.Sprintf("saved-%s-%s", accountKey.Address, accountKey.Account)
	iter := db.NewIterator(util3.BytesPrefix([]byte(prefix)), nil)
	var startID int64 = 0
	for iter.Next() {
		key := iter.Key()
		tweetID, _ := strconv.ParseInt(string(key[len(prefix)+1:]), 10, 64)
		if tweetID > startID || startID == 0 {
			startID = tweetID
		}
	}
	iter.Release()
	//get up to 20 tweets from the tweets-twittername-tweetID prefix with the smallest IDs greater than the startID
	prefix = fmt.Sprintf("tweets-%s", accountKey.Account)
	println(prefix)
	iter = db.NewIterator(util3.BytesPrefix([]byte(prefix)), nil)
	for iter.Next() {
		key := iter.Key()
		tweetID, _ := strconv.ParseInt(string(key[len(prefix)+1:]), 10, 64)
		println("%d", tweetID)
		if tweetID > startID {
			var tweetTx tweets.TweetTx
			err := json.Unmarshal(iter.Value(), &tweetTx)
			if err != nil {
				return 0, jerr.Get("error unmarshaling tweetTx", err)
			}
			tweetList = append(tweetList, tweetTx)
			println(tweetTx.Tweet.Text)
			if len(tweetList) == 20 {
				break
			}
		}
	}
	numTransferred := 0
	for _, tweet := range tweetList {
		match, _ := regexp.MatchString("https://t.co/[a-zA-Z0-9]*$", tweet.Tweet.Text)
		if match {
			//remove the https://t.co from the tweet text
			tweet.Tweet.Text = regexp.MustCompile("https://t.co/[a-zA-Z0-9]*$").ReplaceAllString(tweet.Tweet.Text, "")
		}
		//marshal the tweet.Tweet object into a json and print it
		if len(tweet.Tweet.Entities.Media) > 0 {
			//append the url to the tweet text on a new line
			for _, media := range tweet.Tweet.ExtendedEntities.Media {
				tweet.Tweet.Text += fmt.Sprintf("\n%s", media.MediaURL)
			}
		}
		err := database.StreamTweet(accountKey, tweet, db, appendLink, appendDate)
		if err != nil {
			return numTransferred, jerr.Get("error streaming tweet", err)
		}
		numTransferred++
	}
	return numTransferred, nil
}

func MemoListen(mnemonic *wallet.Mnemonic, addresses []string, botKey wallet.PrivateKey, tweetClient *twitter.Client, db *leveldb.DB) error {
	println("Listening to address: " + addresses[0])
	println("Listening to key: " + botKey.GetBase58Compressed())
	client := graphql.NewSubscriptionClient("ws://127.0.0.1:26770/graphql")
	streamToken,err := tweetstream.GetStreamingToken()
	if err != nil {
		return jerr.Get("error getting streaming token", err)
	}
	api := tweetstream.FetchTweets(streamToken.AccessToken)
	streamArray := updateStreamArray(db,make([]config.Stream, 0))
	err = updateStream(db, api, streamArray)
	if err != nil {
		return jerr.Get("error updating stream", err)
	}
	defer client.Close()
	type Subscription struct {
		Addresses struct {
			Hash   string
			Seen   time.Time
			Raw    string
			Inputs []struct {
				Index     uint32
				PrevHash  string `graphql:"prev_hash"`
				PrevIndex uint32 `graphql:"prev_index"`
				Output    struct {
					Lock struct {
						Address string
					}
				}
			}
			Outputs []struct {
				Script string
				Amount int64
				Lock   struct {
					Address string
				}
			}
			Blocks []struct {
				Hash      string
				Timestamp time.Time
				Height    int
			}
		} `graphql:"addresses(addresses: $addresses)"`
	}
	var subscription = new(Subscription)
	var errorchan = make(chan error)
	num_running := 0
	_, err = client.Subscribe(&subscription, map[string]interface{}{"addresses": addresses}, func(dataValue []byte, errValue error) error {
		num_running += 1
		println("Running: " + strconv.Itoa(num_running))
		if num_running > 1{
			num_running -= 1
			return nil
		}
		if errValue != nil {
			errorchan <- jerr.Get("error in subscription", errValue)
			num_running -= 1
			return nil
		}
		data := Subscription{}
		err := jsonutil.UnmarshalGraphQL(dataValue, &data)
		if err != nil {
			errorchan <- jerr.Get("error marshaling subscription", err)
			num_running -= 1
			return nil
		}
		for _, input := range data.Addresses.Inputs {
			if input.Output.Lock.Address == addresses[0] {
				num_running -= 1
				return nil
			}
		}
		scriptArray := []string{}
		for _, output := range data.Addresses.Outputs {
			scriptArray = append(scriptArray, output.Script)
		}
		message := grabMessage(scriptArray)
		senderAddress := ""
		for _, input := range data.Addresses.Inputs {
			if input.Output.Lock.Address != addresses[0] {
				senderAddress = input.Output.Lock.Address
			}
		}
		coinIndex := uint32(0)
		for i, output := range data.Addresses.Outputs {
			if output.Lock.Address == addresses[0] {
				coinIndex = uint32(i)
			}
		}
		//use regex library to check if message matches the format "CREATE TWITTER {twittername}" tweet names are a maximum of 15 characters
		match, _ := regexp.MatchString("^CREATE TWITTER \\{[a-zA-Z0-9_]{1,15}}$", message)
		if match {
			twitterName := regexp.MustCompile("^CREATE TWITTER \\{([a-zA-Z0-9_]{1,15})}$").FindStringSubmatch(message)[1]
			//check if the value of the transaction is less than 5,000 or this address already has a bot for this account in the database
			botExists := false
			iter := db.NewIterator(util3.BytesPrefix([]byte("linked-"+senderAddress+"-"+twitterName)), nil)
			for iter.Next() {
				botExists = true
			}
			if botExists || data.Addresses.Outputs[coinIndex].Amount < 5000 {
				if data.Addresses.Outputs[coinIndex].Amount < 546 {
					num_running -= 1
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
					PkHash:       wallet.GetAddressFromString(addresses[0]).GetPkHash(),
				}}, botKey, wallet.GetAddressFromString(senderAddress), errMsg); err != nil {
					errorchan <- jerr.Get("error sending money back", err)
					num_running -= 1
					return nil
				}
				num_running -= 1
				return nil
			}
			fmt.Printf("\n\n%s\n\n", message)
			println(addresses[0])
			numStreamBytes, err := db.Get([]byte("memobot-num-streams"), nil)
			if err != nil {
				errorchan <- jerr.Get("error getting num-streams", err)
				num_running -= 1
				return nil
			}
			numStream, err := strconv.ParseUint(string(numStreamBytes), 10, 64)
			if err != nil {
				errorchan <- jerr.Get("error parsing num-streams", err)
				num_running -= 1
				return nil
			}
			//convert numStream to a uint
			numStreamUint := uint(numStream)
			path := wallet.GetBip44Path(wallet.Bip44CoinTypeBTC, numStreamUint+1, false)
			newKey, err := mnemonic.GetPath(path)
			if err != nil {
				errorchan <- jerr.Get("error getting path", err)
				num_running -= 1
				return nil
			}
			newAddr := newKey.GetAddress()
			println("New Key: " + newKey.GetBase58Compressed())
			println("New Address: " + newAddr.GetEncoded())

			if err := database.FundTwitterAddress(memo.UTXO{Input: memo.TxInput{
				Value:        data.Addresses.Outputs[coinIndex].Amount,
				PrevOutHash:  hs.GetTxHash(data.Addresses.Hash),
				PrevOutIndex: coinIndex,
				PkHash:       wallet.GetAddressFromString(addresses[0]).GetPkHash(),
			}}, botKey, newAddr); err != nil {
				errorchan <- jerr.Get("error funding twitter address", err)
				num_running -= 1
				return nil
			}
			newWallet := database.NewWallet(newAddr, *newKey)
			profile, err := tweets.GetProfile(twitterName, tweetClient)
			if err != nil {
				errorchan <- jerr.Get("fatal error getting profile", err)
				num_running -= 1
				return nil
			}
			fmt.Printf("Name: %s\nDesc: %s\nProfile Pic Link: %s\n",
				profile.Name, profile.Description, profile.ProfilePic)
			err = database.UpdateName(newWallet, profile.Name)
			if err != nil {
				errorchan <- jerr.Get("error updating name", err)
				num_running -= 1
				return nil
			} else {
				println("updated name")
			}
			err = database.UpdateProfileText(newWallet, profile.Description)
			if err != nil {
				errorchan <- jerr.Get("error updating profile text", err)
				num_running -= 1
				return nil
			} else {
				println("updated profile text")
			}
			err = database.UpdateProfilePic(newWallet, profile.ProfilePic)
			if err != nil {
				errorchan <- jerr.Get("error updating profile pic", err)
				num_running -= 1
				return nil
			} else {
				println("updated profile pic")
			}
			println("New Key: " + newKey.GetBase58Compressed())
			println("New Address: " + newAddr.GetEncoded())
			err = db.Put([]byte("memobot-num-streams"), []byte(strconv.FormatUint(uint64(numStreamUint+1), 10)), nil)
			if err != nil {
				errorchan <- jerr.Get("error updating memobot-num-streams", err)
				num_running -= 1
				return nil
			}
			//add a field to the database that links the sending address and twitter name to the new key
			err = db.Put([]byte("linked-"+senderAddress+"-"+twitterName), []byte(newKey.GetBase58Compressed()), nil)
			if err != nil {
				errorchan <- jerr.Get("error updating linked-"+senderAddress+"-"+twitterName, err)
				num_running -= 1
				return nil
			}
			streamArray = updateStreamArray(db, streamArray)
			err = updateStream(db,api, streamArray)
			if err != nil {
				errorchan <- jerr.Get("error updating stream", err)
				num_running -= 1
				return nil
			}
			println("done")
		} else if regexp.MustCompile("^WITHDRAW TWITTER \\{([a-zA-Z0-9_]{1,15})}$").MatchString(message) {
			//check the database for each field that matches linked-<senderAddress>-<twitterName>
			//if there is a match, print out the address and key
			//if there is no match, print out an error message
			twitterName := regexp.MustCompile("^WITHDRAW TWITTER \\{([a-zA-Z0-9_]{1,15})}$").FindStringSubmatch(message)[1]
			searchString := "linked-" + senderAddress + "-" + twitterName
			iter := db.NewIterator(util3.BytesPrefix([]byte(searchString)), nil)
			for iter.Next() {
				key := iter.Key()
				value := iter.Value()
				println("Field name: " + string(key))
				println("Private Key: " + string(value))
			}
			iter.Release()
			err := iter.Error()
			if err != nil {
				errorchan <- jerr.Get("error iterating through database", err)
				num_running -= 1
				return nil
			}
		} else {
			fmt.Printf("\n\nMessage not in correct format\n\n")
			//handle sending back money
			//not enough to send back
			if data.Addresses.Outputs[coinIndex].Amount < 546 {
				num_running -= 1
				return nil
			}
			//create a transaction with the sender address and the amount of the transaction
			if err := database.SendToTwitterAddress(memo.UTXO{Input: memo.TxInput{
				Value:        data.Addresses.Outputs[coinIndex].Amount,
				PrevOutHash:  hs.GetTxHash(data.Addresses.Hash),
				PrevOutIndex: coinIndex,
				PkHash:       wallet.GetAddressFromString(addresses[0]).GetPkHash(),
			}}, botKey, wallet.GetAddressFromString(senderAddress), "Message was not in correct format"); err != nil {
				errorchan <- jerr.Get("error sending money back", err)
				num_running -= 1
				return nil
			}
			num_running -= 1
			return nil
		}
		num_running -= 1
		return nil
	})
	if err != nil {
		return jerr.Get("error subscribing to graphql", err)
	}
	fmt.Println("Listening for memos...")
	//client.WithLog(log.Println)
	go func() {
		err = client.Run()
		if err != nil {
			errorchan <- jerr.Get("error running graphql client", err)
		}
	}()
	select {
	case err := <-errorchan:
		return jerr.Get("error in listen", err)
	}
}
func updateStream(db *leveldb.DB, api *twitterstream.TwitterApi, streamArray []config.Stream) error{
	//check if there are already running streams on this token
	//if there are, stop them
	//api.Stream.StopStream()
	//create an array of {twitterName, newKey} objects by searching through the linked-<senderAddress>-<twitterName> fields
	if len(streamArray) == 0 {
		return nil
	}
	go func() {
		tweetstream.ResetRules(api)
		tweetstream.FilterAccount(api, streamArray)
		tweetstream.InitiateStream(api, streamArray, db)
		tweetstream.ResetRules(api)
	}()
	return nil
}

func updateStreamArray(db *leveldb.DB, streamArray []config.Stream) []config.Stream{
	iter := db.NewIterator(util3.BytesPrefix([]byte("linked-")), nil)
	for iter.Next() {
		//find the twitterName at the end of the linked-<senderAddress>-<twitterName> field
		twitterName := strings.Split(string(iter.Key()), "-")[2]
		newKey := string(iter.Value())
		//only add it to the stream array if this key : name pair isn't already in it
		found := false
		for _, stream := range streamArray {
			if stream.Key == newKey && stream.Name == twitterName {
				found = true
				break
			}
		}
		if !found {
			streamArray = append(streamArray, config.Stream{Key: newKey, Name: twitterName})
		}
	}
	return streamArray
}

func grabMessage(outputScripts []string) string {
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
