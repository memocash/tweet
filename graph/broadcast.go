package graph

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/index/ref/bitcoin/memo"
	"net/http"
	"time"
)

func Broadcast(memoTx *memo.Tx) error {
	//log.Printf("Broadcasting transaction: %s\n", memoTx.MsgTx.TxHash())
	//parse.GetTxInfo(memoTx).Print()
	raw := memo.GetRaw(memoTx.MsgTx)
	jsonData := map[string]interface{}{
		"query": `mutation ($raw: String!) {
					broadcast(raw: $raw)
				}`,
		"variables": map[string]string{
			"raw": hex.EncodeToString(raw),
		},
	}
	jsonValue, _ := json.Marshal(jsonData)
	request, err := http.NewRequest("POST", ServerUrlHttp, bytes.NewBuffer(jsonValue))
	if err != nil {
		return jerr.Get("error making a new request failed complete transaction", err)
	}
	request.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: time.Second * 10}
	resp, err := client.Do(request)
	if err != nil {
		return jerr.Get("error the HTTP request failed complete transaction", err)
	}
	if resp.StatusCode != http.StatusOK {
		return jerr.New("error the HTTP request failed complete transaction, status code: " + resp.Status)
	}
	return nil
}
