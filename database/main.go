package database

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/index/ref/bitcoin/memo"
	"github.com/memocash/index/ref/bitcoin/tx/gen"
	"github.com/memocash/index/ref/bitcoin/tx/hs"
	"github.com/memocash/index/ref/bitcoin/tx/parse"
	"github.com/memocash/index/ref/bitcoin/tx/script"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/syndtr/goleveldb/leveldb"
	"io/ioutil"
	"net/http"
	"time"
)

func GetDb() (*leveldb.DB, error) {
	db, err := leveldb.OpenFile("tweets.db", nil)
	if err != nil {
		return nil, jerr.Get("error opening db", err)
	}
	return db, nil
}

type Address struct {
	Outputs []Output `json:"outputs"`
}

type Output struct {
	Tx     Tx      `json:"tx"`
	Hash   string  `json:"hash"`
	Index  int     `json:"index"`
	Amount int64   `json:"amount"`
	Spends []Input `json:"spends"`
}

type Input struct {
	Hash  string `json:"hash"`
	Index int    `json:"index"`
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
	if len(g.UTXOs) != 0 {
		return g.UTXOs, nil
	}
	jsonData := map[string]string{
		"query": `
            {
                address (address: "` + g.Address.GetEncoded() + `") {
                    outputs {
						tx {
							seen
						}
						hash
						index
						amount
						spends {
							hash
						    index
						}
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
	var utxos []memo.UTXO
	for _, output := range dataStruct.Data.Address.Outputs {
		if len(output.Spends) > 0 {
			continue
		}
		utxos = append(utxos, memo.UTXO{
			Input: memo.TxInput{
				PkScript:     pkScript,
				PkHash:       pkHash,
				Value:        output.Amount,
				PrevOutHash:  hs.GetTxHash(output.Hash),
				PrevOutIndex: uint32(output.Index),
			},
		})
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
				break
			}
		}
	}
}

func (g *InputGetter) AddChangeUTXO(new memo.UTXO) {
	g.UTXOs = append(g.UTXOs, new)
}

func (g *InputGetter) NewTx() {
}

type Wallet struct {
	Address wallet.Address
	Key     wallet.PrivateKey
	Getter  gen.InputGetter
}

func NewWallet(address wallet.Address, key wallet.PrivateKey) Wallet {
	return Wallet{
		Address: address,
		Key:     key,
		Getter:  &InputGetter{Address: address},
	}
}

func MakePost(wlt Wallet, message string) ([]byte, error) {
	memoTx, err := buildTx(wlt, script.Post{Message: message})
	if err != nil {
		return nil, jerr.Get("error generating memo tx", err)
	}
	txInfo := parse.GetTxInfo(memoTx)
	txInfo.Print()
	completeTransaction(memoTx, err)
	return memoTx.GetHash(), nil
}
func MakeReply(wallet Wallet, parentHash []byte, message string) ([]byte, error) {
	memoTx, err := buildTx(wallet, script.Reply{Message: message, TxHash: parentHash})
	if err != nil {
		return nil, jerr.Get("error generating memo tx", err)
	}
	txInfo := parse.GetTxInfo(memoTx)
	txInfo.Print()
	completeTransaction(memoTx, err)
	return memoTx.GetHash(), nil
}

func FundTwitterAddress(utxo memo.UTXO, key wallet.PrivateKey, address wallet.Address) error {
	memoTx, err := gen.Tx(gen.TxRequest{
		InputsToUse: []memo.UTXO{utxo},
		Outputs: []*memo.Output{{
			Amount: memo.GetMaxSendFromCount(utxo.Input.Value, 1),
			Script: script.P2pkh{PkHash: address.GetPkHash()},
		}},
		KeyRing: wallet.KeyRing{
			Keys: []wallet.PrivateKey{key},
		},
	})
	txInfo := parse.GetTxInfo(memoTx)
	txInfo.Print()
	completeTransaction(memoTx, err)
	return nil
}
func SendToTwitterAddress(utxo memo.UTXO, key wallet.PrivateKey, address wallet.Address, errorMsg string) error {
	memoTx, err := gen.Tx(gen.TxRequest{
		InputsToUse: []memo.UTXO{utxo},
		Outputs: []*memo.Output{{
			Amount: memo.GetMaxSendFromCount(utxo.Input.Value, 1) - (int64(36 + len(errorMsg))),
			Script: script.P2pkh{PkHash: address.GetPkHash()},
		}, {
			Amount: 0,
			Script: script.Send{
				Hash:    address.GetPkHash(),
				Message: errorMsg},
		}},
		KeyRing: wallet.KeyRing{
			Keys: []wallet.PrivateKey{key},
		},
	})
	txInfo := parse.GetTxInfo(memoTx)
	txInfo.Print()
	completeTransaction(memoTx, err)
	return nil
}
func UpdateName(wlt Wallet, name string) error {
	memoTx, err := buildTx(wlt, script.SetName{Name: name})
	if err != nil {
		return jerr.Get("error generating memo tx", err)
	}
	txInfo := parse.GetTxInfo(memoTx)
	txInfo.Print()
	completeTransaction(memoTx, err)
	return nil
}

func UpdateProfileText(wlt Wallet, profile string) error {
	if profile == "" {
		profile = " "
	}
	memoTx, err := buildTx(wlt, script.Profile{Text: profile})
	if err != nil {
		return jerr.Get("error generating memo tx", err)
	}
	txInfo := parse.GetTxInfo(memoTx)
	txInfo.Print()
	completeTransaction(memoTx, err)
	return nil
}

func UpdateProfilePic(wlt Wallet, url string) error {
	memoTx, err := buildTx(wlt, script.ProfilePic{Url: url})
	if err != nil {
		return jerr.Get("error generating memo tx", err)
	}
	println("tx", memoTx.GetHash())
	txInfo := parse.GetTxInfo(memoTx)
	txInfo.Print()
	completeTransaction(memoTx, err)
	return nil
}

func buildTx(wlt Wallet, outputScript memo.Script) (*memo.Tx, error) {
	memoTx, err := gen.Tx(gen.TxRequest{
		Getter: wlt.Getter,
		Outputs: []*memo.Output{{
			Script: outputScript,
		}},
		Change: wallet.Change{Main: wlt.Address},
		KeyRing: wallet.KeyRing{
			Keys: []wallet.PrivateKey{wlt.Key},
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
