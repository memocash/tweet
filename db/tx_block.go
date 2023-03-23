package db

import (
	"github.com/jchavannes/jgo/jutil"
)

type TxBlock struct {
	TxHash    [32]byte
	BlockHash [32]byte
}

func (tb *TxBlock) GetPrefix() string {
	return PrefixTxBlock
}

func (tb *TxBlock) GetUid() []byte {
	return jutil.CombineBytes(
		tb.TxHash[:],
		tb.BlockHash[:],
	)
}

func (tb *TxBlock) SetUid(b []byte) {
	if len(b) != 64 {
		return
	}
	copy(tb.TxHash[:], b[:32])
	copy(tb.BlockHash[:], b[32:])
}

func (tb *TxBlock) Serialize() []byte {
	return nil
}

func (tb *TxBlock) Deserialize([]byte) {
}
