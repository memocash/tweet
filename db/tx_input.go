package db

import (
	"fmt"
	"github.com/jchavannes/jgo/jutil"
)

type TxInput struct {
	PrevHash  [32]byte
	PrevIndex int
}

func (i *TxInput) GetPrefix() string {
	return PrefixTxInput
}

func (i *TxInput) GetUid() []byte {
	return jutil.CombineBytes(
		i.PrevHash[:],
		jutil.GetIntData(i.PrevIndex),
	)
}

func (i *TxInput) SetUid(b []byte) {
	if len(b) != 36 {
		return
	}
	copy(i.PrevHash[:], b[:32])
	i.PrevIndex = jutil.GetInt(b[32:])
}

func (i *TxInput) Serialize() []byte {
	return nil
}

func (i *TxInput) Deserialize([]byte) {
}

func GetTxInput(prevHash [32]byte, prevIndex int) (*TxInput, error) {
	var txInput = &TxInput{
		PrevHash:  prevHash,
		PrevIndex: prevIndex,
	}
	if err := GetSpecificItem(txInput); err != nil {
		return nil, fmt.Errorf("error getting tx input; %w", err)
	}
	return txInput, nil
}
