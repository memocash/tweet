package bot

import (
	"bytes"
	"encoding/hex"
	"github.com/jchavannes/btcd/txscript"
	"github.com/memocash/index/ref/bitcoin/memo"
)

type UpdateQuery struct {
	Address struct {
		Txs []Tx `graphql:"txs(start: $start)"`
	} `graphql:"address(address: $address)"`
}

type Subscription struct {
	Addresses Tx `graphql:"addresses(addresses: $addresses)"`
}

type Tx struct {
	Hash   string
	Seen   GraphQlDate
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
		Timestamp GraphQlDate
		Height    int
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
