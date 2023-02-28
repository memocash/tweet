package graph

import (
	"time"
)

const (
	ServerUrlHttp = "http://127.0.0.1:26770/graphql"
	ServerUrlWs   = "ws://127.0.0.1:26770/graphql"
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
	Seen   Date
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
		Timestamp Date
		Height    int
	}
}

type Date string

func (d Date) GetGraphQLType() string {
	return "Date"
}

func (d Date) GetTime() time.Time {
	t, _ := time.Parse(time.RFC3339, string(d))
	return t
}
