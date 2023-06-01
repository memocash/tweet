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
		BlockHash string `graphql:"block_hash"`
		Block     struct {
			Timestamp Date
			Height    int
		}
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

type Profiles struct {
	Profiles []Profile `graphql:"profiles(addresses: [$address])"`
}
type Profile struct {
	Lock struct {
		Address string
	}
	Address string
	Name    struct {
		Tx   Tx
		Name string
	}
	Profile struct {
		Tx   Tx
		Text string
	}
	Pic struct {
		Tx  Tx
		Pic string
	}
	Following []Follow `graphql:"following(start: $start)"`
	Followers []Follow `graphql:"followers(start: $start)"`
	Posts     []Post   `graphql:"posts()"`
}

type Follow struct {
	Tx      Tx
	Tx_hash string
	Lock    struct {
		Address string
	}
	Address     string
	Follow_Lock struct {
		Address string
	}
	Follow_Address string
	Unfollow       bool
}

type Post struct {
	Tx      Tx
	Tx_hash string
	Lock    struct {
		Address string
	}
	Address string
	Text    string
	Likes   []struct {
		Tx  Tx
		Tip int64
	}
	//Parent is another post
	Parent struct {
		Tx      Tx
		Tx_hash string
		Address string
		Text    string
	}
	Replies []struct {
		Tx      Tx
		Tx_hash string
		Address string
		Text    string
	}
}
