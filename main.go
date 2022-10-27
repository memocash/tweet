package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/index/ref/bitcoin/memo"
	"github.com/memocash/tweet/graph"
	"github.com/memocash/tweet/tweets"
	"net/http"
	"time"
)

func main() {
	tweets := tweets.Load()
	for _,tweet := range tweets {
		memoTx, err := graph.BasicQuery(tweet.Text)
		if err != nil {
			jerr.Get("error running basic query", err).Fatal()
		}

		jsonData := map[string]string{
			"mutation": `{broadcast(raw: ` + hex.EncodeToString(memo.GetRaw(memoTx.MsgTx)) +`}`,
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
}
