package db

import (
	"fmt"
	"strings"
)

type TxBlock struct {
	TxHash    string
	BlockHash string
}

func (tb *TxBlock) GetPrefix() string {
	return PrefixTxBlock
}

func (tb *TxBlock) GetUid() []byte {
	return []byte(fmt.Sprintf("%s-%s", tb.TxHash, tb.BlockHash))
}

func (tb *TxBlock) SetUid(b []byte) {
	parts := strings.Split(string(b), "-")
	if len(parts) != 2 {
		return
	}
	tb.TxHash = parts[0]
	tb.BlockHash = parts[1]
}

func (tb *TxBlock) Serialize() []byte {
	return nil
}

func (tb *TxBlock) Deserialize([]byte) {
}
