package graph

import (
	"bytes"
	"encoding/json"
	"fmt"
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

func BasicQuery() {
	jsonData := map[string]string{
		"query": `
            {
                address (address: "1JJhfR2fD3mmxipTXxdtvehBiep1WNzM9q") {
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
		fmt.Printf("The HTTP request failed with error %s\n", err)
	}
	data, _ := ioutil.ReadAll(response.Body)
	var dataStruct = struct {
		Data struct {
			Address Address `json:"address"`
		} `json:"data"`
	}{}
	if err := json.Unmarshal(data, &dataStruct); err != nil {
		panic(err)
	}
	fmt.Printf("%#v\n", dataStruct.Data.Address)
}
