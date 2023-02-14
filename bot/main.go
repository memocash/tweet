package bot

import (
	"bytes"
	"encoding/hex"
	"github.com/jchavannes/btcd/txscript"
	"github.com/memocash/index/ref/bitcoin/memo"
	"time"
)

type UpdateQuery struct {
	Address struct {
		Txs []struct {
			Hash string `graphql:"hash"`
		} `graphql:"txs(start: $start)"`
	} `graphql:"address(address: $address)"`
}

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
