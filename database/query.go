package database

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/index/ref/bitcoin/memo"
	"github.com/memocash/index/ref/bitcoin/tx/gen"
	"github.com/memocash/index/ref/bitcoin/tx/hs"
	"github.com/memocash/index/ref/bitcoin/tx/parse"
	"github.com/memocash/index/ref/bitcoin/tx/script"
	"github.com/memocash/index/ref/bitcoin/util/testing/test_tx"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/cmd/util"
	"html"
	"io/ioutil"
	"net/http"
	"time"
)

type Address struct {
	Utxos []Utxo `json:"utxos"`
}

type Utxo struct {
	Tx     Tx     `json:"tx"`
	Hash   string `json:"hash"`
	Index  int    `json:"index"`
	Amount int64  `json:"amount"`
}

type Tx struct {
	Seen time.Time `json:"seen"`
}

type InputGetter struct {
	Address wallet.Address
	UTXOs   []memo.UTXO
}

func (g *InputGetter) SetPkHashesToUse([][]byte) {
}

func (g *InputGetter) GetUTXOs(*memo.UTXORequest) ([]memo.UTXO, error) {
	if len(g.UTXOs) != 0{
		return g.UTXOs, nil
	}
	jsonData := map[string]string{
		"query": `
            {
                address (address: "` + test_tx.Address3String + `") {
                    utxos {
						tx {
							seen
						}
						hash
						index
						amount
					}
                }
            }
        `,
	}
	jsonValue, _ := json.Marshal(jsonData)
	request, err := http.NewRequest("POST", "http://localhost:26770/graphql", bytes.NewBuffer(jsonValue))
	request.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: time.Second * 10}
	response, err := client.Do(request)
	defer response.Body.Close()
	if err != nil {
		return nil, jerr.Get("The HTTP request failed with error %s\n", err)
	}
	data, _ := ioutil.ReadAll(response.Body)
	var dataStruct = struct {
		Data struct {
			Address Address `json:"address"`
		} `json:"data"`
	}{}
	if err := json.Unmarshal(data, &dataStruct); err != nil {
		return nil, jerr.Get("error unmarshalling json", err)
	}
	pkHash := g.Address.GetPkHash()
	pkScript, err := script.P2pkh{PkHash: pkHash}.Get()
	if err != nil {
		return nil, jerr.Get("error getting pk script", err)
	}
	var utxos = make([]memo.UTXO, len(dataStruct.Data.Address.Utxos))
	for i, utxo := range dataStruct.Data.Address.Utxos {
		utxos[i] = memo.UTXO{
			Input: memo.TxInput{
				PkScript:     pkScript,
				PkHash:       pkHash,
				Value:        utxo.Amount,
				PrevOutHash:  hs.GetTxHash(utxo.Hash),
				PrevOutIndex: uint32(utxo.Index),
			},
		}
	}
	g.UTXOs = utxos
	return utxos, nil
}

func (g *InputGetter) MarkUTXOsUsed(used []memo.UTXO) {
	for i := 0; i < len(g.UTXOs); i++ {
		for j := 0; j < len(used); j++ {
			if g.UTXOs[i].IsEqual(used[j]) {
				//remove g.UTXOs[i] from the list
				g.UTXOs = append(g.UTXOs[:i], g.UTXOs[i+1:]...)
				//decrement i so we don't go out of bounds
				i--
			}
		}
	}
}

func (g *InputGetter) AddChangeUTXO(new memo.UTXO) {
	g.UTXOs = append(g.UTXOs, new)
}

func (g *InputGetter) NewTx() {
}

func TransferTweets(client *twitter.Client, address wallet.Address, key wallet.PrivateKey,account string, archive util.TweetObject, appendLink bool, appendDate bool) error {
	var tweetList []twitter.Tweet
	if(len(archive.TweetList) - archive.Archived >= 20){
		tweetList = archive.TweetList[len(archive.TweetList)-archive.Archived-20:len(archive.TweetList)-archive.Archived]
	} else{
		tweetList = archive.TweetList
	}
	//reverse tweetList so they are posted in chronological order in memo, instead of reverse chronological
	for i := len(tweetList)/2 - 1; i >= 0; i-- {
		opp := len(tweetList) - 1 - i
		tweetList[i], tweetList[opp] = tweetList[opp], tweetList[i]
	}
	for _, tweet := range tweetList {
		if(tweet.InReplyToStatusID == 0) {
			tweetLink := fmt.Sprintf("\nhttps://twitter.com/twitter/status/%d\n", tweet.ID)
			tweetDate := fmt.Sprintf("\n%s\n", tweet.CreatedAt)
			tweetText := tweet.Text
			if appendLink {
				tweetText += tweetLink
			}
			if appendDate {
				tweetText += tweetDate
			}
			parentHash, err := makePost(address, key, html.UnescapeString(tweetText))
			if err != nil {
				return jerr.Get("error making post", err)
			}
			//this part should be recursively called for each reply found
			err = recursiveReplies(parentHash, tweet, address, key, archive, appendLink, appendDate)
			if err != nil {
				return jerr.Get("error getting replies", err)
			}
		}
	}
	return nil
}
func recursiveReplies(parentHash []byte, tweet twitter.Tweet, address wallet.Address, key wallet.PrivateKey, archive util.TweetObject, appendLink bool, appendDate bool) error {
	replies := archive.TweetList
	for _, reply := range replies {
		if reply.InReplyToStatusID != tweet.ID{
			continue
		}
		println("\n\n\n\n" + reply.Text + "\n\n\n\n")
		replyLink := fmt.Sprintf("\nhttps://twitter.com/twitter/status/%d\n",reply.ID)
		replyDate := fmt.Sprintf("\n%s\n",reply.CreatedAt)
		replyText := reply.Text
		if appendLink {
			replyText += replyLink
		}
		if appendDate {
			replyText += replyDate
		}
		parentHash,err := makeReply(parentHash,address,key, html.UnescapeString(replyText))
		err = recursiveReplies(parentHash, reply, address, key, archive, appendLink, appendDate)
		if err != nil {
			return jerr.Get("error making reply", err)
		}
	}
	return nil
}
func makePost(address wallet.Address,key wallet.PrivateKey,message string) ([]byte, error) {
	memoTx, err := buildTx(address,key, script.Post{Message: message})
	if err != nil {
		return nil,jerr.Get("error generating memo tx", err)
	}
	txInfo := parse.GetTxInfo(memoTx)
	txInfo.Print()
	completeTransaction(memoTx, err)
	return memoTx.GetHash(), nil
}
func makeReply(parentHash []byte, address wallet.Address,key wallet.PrivateKey,message string) ([]byte, error) {
	memoTx, err := buildTx(address,key, script.Reply{Message: message, TxHash: parentHash})
	if err != nil {
		return nil,jerr.Get("error generating memo tx", err)
	}
	txInfo := parse.GetTxInfo(memoTx)
	txInfo.Print()
	completeTransaction(memoTx, err)
	return memoTx.GetHash(), nil
}

func UpdateName(address wallet.Address, key wallet.PrivateKey, name string) error {
	memoTx, err := buildTx(address, key, script.SetName{Name: name})
	if err != nil {
		return jerr.Get("error generating memo tx", err)
	}
	txInfo := parse.GetTxInfo(memoTx)
	txInfo.Print()
	completeTransaction(memoTx, err)
	return nil
}

func UpdateProfileText(address wallet.Address, key wallet.PrivateKey, profile string) error {
	memoTx, err := buildTx(address, key, script.Profile{Text: profile})
	if err != nil {
		return jerr.Get("error generating memo tx", err)
	}
	txInfo := parse.GetTxInfo(memoTx)
	txInfo.Print()
	completeTransaction(memoTx, err)
	return nil
}

func UpdateProfilePic(address wallet.Address, key wallet.PrivateKey, url string) error {
	memoTx, err := buildTx(address, key, script.ProfilePic{Url: url})
	if err != nil {
		return jerr.Get("error generating memo tx", err)
	}
	txInfo := parse.GetTxInfo(memoTx)
	txInfo.Print()
	completeTransaction(memoTx, err)
	return nil
}
func buildTx(address wallet.Address, key wallet.PrivateKey, outputScript memo.Script)(*memo.Tx, error){
	getter := &InputGetter{Address: address}
	memoTx, err := gen.Tx(gen.TxRequest{
		Getter: getter,
		Outputs: []*memo.Output{{
			Script: outputScript,
		}},
		Change: wallet.Change{Main: address},
		KeyRing: wallet.KeyRing{
			Keys: []wallet.PrivateKey{key},
		},
	})
	return memoTx, err
}
func completeTransaction(memoTx *memo.Tx, err error) {
	if err != nil {
		jerr.Get("error running basic query", err).Fatal()
	}
	jsonData := map[string]interface{}{
		"query": `mutation ($raw: String!) {
					broadcast(raw: $raw)
				}`,
		"variables": map[string]string{
			"raw": hex.EncodeToString(memo.GetRaw(memoTx.MsgTx)),
		},
	}
	jsonValue, _ := json.Marshal(jsonData)
	request, err := http.NewRequest("POST", "http://localhost:26770/graphql", bytes.NewBuffer(jsonValue))
	if err != nil {
		jerr.Get("Making a new request failed\n", err).Fatal()
	}
	request.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: time.Second * 10}
	response, err := client.Do(request)
	fmt.Printf("%#v\n", response)
	if err != nil {
		jerr.Get("The HTTP request failed with error %s\n", err).Fatal()
	}
}
