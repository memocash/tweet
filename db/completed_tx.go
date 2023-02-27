package db

import (
	"errors"
	"fmt"
	"github.com/jchavannes/jgo/jutil"
	"github.com/syndtr/goleveldb/leveldb"
)

type CompletedTx struct {
	TxHash [32]byte
}

func (t *CompletedTx) GetPrefix() string {
	return PrefixCompletedTx
}

func (t *CompletedTx) GetUid() []byte {
	return jutil.CombineBytes(
		t.TxHash[:],
	)
}

func (t *CompletedTx) SetUid(b []byte) {
	if len(b) != 32 {
		return
	}
	copy(t.TxHash[:], b)
}

func (t *CompletedTx) Serialize() []byte {
	return nil
}

func (t *CompletedTx) Deserialize([]byte) {
}

func HasCompletedTx(txHash [32]byte) (bool, error) {
	var completedTx = &CompletedTx{TxHash: txHash}
	if err := GetItem(completedTx); err != nil {
		if errors.Is(err, leveldb.ErrNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("error getting completed tx item; %w", err)
	}
	return true, nil
}
