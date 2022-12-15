package util

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/hasura/go-graphql-client"
	"github.com/hasura/go-graphql-client/pkg/jsonutil"
	"github.com/jchavannes/btcd/txscript"
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/index/ref/bitcoin/memo"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/cmd/util"
	"github.com/memocash/tweet/database"
	"github.com/memocash/tweet/tweets"
	"github.com/syndtr/goleveldb/leveldb"
	util3 "github.com/syndtr/goleveldb/leveldb/util"
	"html"
	"log"
	"regexp"
	"strconv"
	"time"
)
func StreamTweet(address wallet.Address, key wallet.PrivateKey, tweet util.TweetTx, db *leveldb.DB, appendLink bool, appendDate bool) error {
	if tweet.Tweet == nil {
		return jerr.New("tweet is nil")
	}
	wlt := database.NewWallet(address, key)
	tweetLink := fmt.Sprintf("\nhttps://twitter.com/twitter/status/%d\n", tweet.Tweet.ID)
	tweetDate := fmt.Sprintf("\n%s\n", tweet.Tweet.CreatedAt)
	tweetText := tweet.Tweet.Text
	if appendLink {
		tweetText += tweetLink
	}
	if appendDate {
		tweetText += tweetDate
	}
	//add the tweet to the twitter-twittername-tweetID prefix
	prefix := fmt.Sprintf("tweets-%s-%019d", tweet.Tweet.User.ScreenName, tweet.Tweet.ID)
	tweetTx,_ := json.Marshal(tweet)
	db.Put([]byte(prefix),tweetTx,nil)
	//if the tweet was a regular post, post it normally
	if tweet.Tweet.InReplyToStatusID == 0 {
		parentHash, err := database.MakePost(wlt, html.UnescapeString(tweetText))
		//find this tweet in archive and set its hash to the hash of the post that was just made
		tweet.TxHash = parentHash
		if err != nil {
			return jerr.Get("error making post", err)
		}
	} else {
		var parentHash []byte = nil
		//search the saved-address-twittername-tweetID prefix for the tweet that this tweet is a reply to
		prefix := fmt.Sprintf("saved-%s-%s", address, tweet.Tweet.User.ScreenName)
		iter := db.NewIterator(util3.BytesPrefix([]byte(prefix)), nil)
		for iter.Next() {
			key := iter.Key()
			tweetID,_ := strconv.ParseInt(string(key[len(prefix)+1:]), 10, 64)
			if tweetID == tweet.Tweet.InReplyToStatusID {
				parentHash = iter.Value()
				break
			}
		}
		//if it turns out this tweet was actually a reply to another person's tweet, post it as a regular post
		if parentHash == nil {
			parentHash, err := database.MakePost(wlt, html.UnescapeString(tweetText))
			//find this tweet in archive and set its hash to the hash of the post that was just made
			tweet.TxHash = parentHash
			if err != nil {
				return jerr.Get("error making post", err)
			}
			//otherwise, it's part of a thread, so post it as a reply to the parent tweet
		} else {
			replyHash, err := database.MakeReply(wlt, parentHash, html.UnescapeString(tweetText))
			//find this tweet in archive and set its hash to the hash of the post that was just made
			tweet.TxHash = replyHash
			if err != nil {
				return jerr.Get("error making reply", err)
			}
		}
	}
	//save the txHash to the saved-address-twittername-tweetID prefix
	prefix = fmt.Sprintf("saved-%s-%s", address, tweet.Tweet.User.ScreenName)
	dbKey := fmt.Sprintf("%s-%019d", prefix, tweet.Tweet.ID)
	err := db.Put([]byte(dbKey), tweet.TxHash, nil)
	if err != nil {
		return jerr.Get("error saving tweetTx", err)
	}
	return nil
}
func TransferTweets(address wallet.Address, key wallet.PrivateKey, screenName string, db *leveldb.DB, appendLink bool, appendDate bool) (int, error) {
	var tweetList []util.TweetTx
	//find the greatest ID of all the already saved tweets
	prefix := fmt.Sprintf("saved-%s-%s",address, screenName)
	iter := db.NewIterator(util3.BytesPrefix([]byte(prefix)), nil)
	var startID int64 = 0
	for iter.Next() {
		key := iter.Key()
		tweetID,_ := strconv.ParseInt(string(key[len(prefix)+1:]), 10, 64)
		if tweetID > startID || startID == 0 {
			startID = tweetID
		}
	}
	iter.Release()
	//get up to 20 tweets from the tweets-twittername-tweetID prefix with the smallest IDs greater than the startID
	prefix = fmt.Sprintf("tweets-%s", screenName)
	println(prefix)
	iter = db.NewIterator(util3.BytesPrefix([]byte(prefix)), nil)
	for iter.Next() {
		key := iter.Key()
		tweetID,_ := strconv.ParseInt(string(key[len(prefix)+1:]), 10, 64)
		println("%d",tweetID)
		if tweetID > startID {
			var tweetTx util.TweetTx
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
		err := StreamTweet(address, key, tweet, db, appendLink, appendDate)
		if err != nil {
			return numTransferred, jerr.Get("error streaming tweet", err)
		}
		numTransferred++
	}
	return numTransferred, nil
}
func MemoListen(mnemonic *wallet.Mnemonic, addresses []string, botKey wallet.PrivateKey,tweetClient *twitter.Client, db *leveldb.DB) error{
	println("Listening to address: "+addresses[0])
	println("Listening to key: "+botKey.GetBase58Compressed())
	client := graphql.NewSubscriptionClient("ws://127.0.0.1:26770/graphql")
	defer client.Close()
	type Subscription struct {
		Addresses struct{
			Hash string
			Seen time.Time
			Raw string
			Inputs []struct{
				Index uint32
				PrevHash string `graphql:"prev_hash"`
				PrevIndex uint32 `graphql:"prev_index"`
				Output struct{
					Lock struct{
						Address string
					}
				}
			}
			Outputs []struct{
				Script string
			}
			Blocks []struct{
				Hash string
				Timestamp time.Time
				Height int
			}
		} `graphql:"addresses(addresses: $addresses)"`
	}
	var subscription = new(Subscription)

	var errorchan = make(chan error)
	_, err := client.Subscribe(&subscription, map[string]interface{}{"addresses": addresses}, func(dataValue []byte, errValue error) error {
		if errValue != nil {
			errorchan <- jerr.Get("error in subscription", errValue)
			return nil
		}
		data := Subscription{}
		err := jsonutil.UnmarshalGraphQL(dataValue, &data)
		if err != nil {
			errorchan <- jerr.Get("error marshaling subscription", err)
			return nil
		}
		for _,input := range data.Addresses.Inputs {
			if(input.Output.Lock.Address == addresses[0]){
				return nil
			}
		}
		scriptArray := []string{}
		for _, output := range data.Addresses.Outputs {
			scriptArray = append(scriptArray, output.Script)
		}
		message := grabMessage(scriptArray)
		//use regex library to check if message matches the format "CREATE TWITTER {twittername}" tweet names are a maximum of 15 characters
		match, _ := regexp.MatchString("^CREATE TWITTER \\{[a-zA-Z0-9_]{1,15}}$", message)
		if match {
			fmt.Printf("\n\n%s\n\n",message)
			//get the twittername from the message
			twitterName := regexp.MustCompile("^CREATE TWITTER \\{([a-zA-Z0-9_]{1,15})}$").FindStringSubmatch(message)[1]
			name,desc,profilePic,_ := tweets.GetProfile(twitterName,tweetClient)
			fmt.Printf("Name: %s\nDesc: %s\nProfile Pic Link: %s\n",name,desc,profilePic)
			println(addresses[0])
			//botAddress := botKey.GetAddress()
			//read the value of memobot-num-stream from the database as an unsigned integer
			numStreamBytes, err := db.Get([]byte("memobot-num-streams"), nil)
			if err != nil {
				errorchan <- jerr.Get("error getting num-streams", err)
				return nil
			}
			numStream, err := strconv.ParseUint(string(numStreamBytes), 10, 64)
			if err != nil {
				errorchan <- jerr.Get("error parsing num-streams", err)
				return nil
			}
			//convert numStream to a uint
			numStreamUint := uint(numStream)
			path := wallet.GetBip44Path(wallet.Bip44CoinTypeBTC, numStreamUint+1, false)
			newKey, err := mnemonic.GetPath(path)
			if err != nil {
				errorchan <- jerr.Get("error getting path", err)
				return nil
			}
			newAddr := newKey.GetAddress()
			err = database.UpdateName(database.NewWallet(newAddr,*newKey),name)
			if err != nil {
				errorchan <- jerr.Get("error updating name", err)
				return nil
			} else{
				println("updated name")
			}
			err = database.UpdateProfileText(database.NewWallet(newAddr,*newKey),desc)
			if err != nil {
				errorchan <- jerr.Get("error updating profile text", err)
				return nil
			} else{
				println("updated profile text")
			}
			err = database.UpdateProfilePic(database.NewWallet(newAddr,*newKey),profilePic)
			if err != nil {
				errorchan <- jerr.Get("error updating profile pic", err)
				return nil
			} else{
				println("updated profile pic")
			}
			println("done")
			//update the database with numStreamUint+1
			println("New Key: " + newKey.GetBase58Compressed())
			println("New Address: " + newAddr.GetEncoded())
			err = db.Put([]byte("memobot-num-streams"), []byte(strconv.FormatUint(uint64(numStreamUint+1), 10)), nil)
			if err != nil {
				errorchan <- jerr.Get("error updating memobot-num-streams", err)
				return nil
			}
		} else{
			fmt.Printf("\n\nMessage not in correct format\n\n")
			//handle sending back money
		}
		return nil
	})
	if err != nil {
		return jerr.Get("error subscribing to graphql", err)
	}
	fmt.Println("Listening for memos...")
	client.WithLog(log.Println)
	go func() {
		err = client.Run()
		if err != nil {
			errorchan <- jerr.Get("error running graphql client", err)
		}
	}()

	select{
	case err := <-errorchan:
		return jerr.Get("error in listen", err)
	}
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
