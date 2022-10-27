package graph

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/index/ref/bitcoin/memo"
	"github.com/memocash/index/ref/bitcoin/tx/gen"
	"github.com/memocash/index/ref/bitcoin/tx/hs"
	"github.com/memocash/index/ref/bitcoin/tx/script"
	"github.com/memocash/index/ref/bitcoin/util/testing/test_tx"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"io/ioutil"
	"net/http"
	"time"
)

const (
	AddressString = test_tx.Address3String
	PrivateKey    = test_tx.Key3String
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
}

func (g *InputGetter) SetPkHashesToUse([][]byte) {
}

func (g *InputGetter) GetUTXOs(*memo.UTXORequest) ([]memo.UTXO, error) {
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
	return utxos, nil
}

func (g *InputGetter) MarkUTXOsUsed([]memo.UTXO) {
}

func (g *InputGetter) AddChangeUTXO(memo.UTXO) {
}

func (g *InputGetter) NewTx() {
}

func BasicQuery() error {
	address := test_tx.Address3
	getter := &InputGetter{Address: address}
	memoTx, err := gen.Tx(gen.TxRequest{
		Getter: getter,
		Outputs: []*memo.Output{{
			Script: &script.Post{Message: "test"},
		}},
		Change: wallet.Change{Main: address},
		KeyRing: wallet.KeyRing{
			Keys: []wallet.PrivateKey{test_tx.Address3key},
		},
	})
	if err != nil {
		return jerr.Get("error generating memo tx", err)
	}
	fmt.Printf("MsgTx: %#v\n", memoTx.MsgTx)
	return nil
}
