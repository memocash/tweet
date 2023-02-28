package db

import (
	"fmt"
	"github.com/jchavannes/jgo/jutil"
	"strings"
)

type TxInput struct {
	PrevHash  string
	PrevIndex int
}

func (i *TxInput) GetPrefix() string {
	return PrefixTxInput
}

func (i *TxInput) GetUid() []byte {
	return []byte(fmt.Sprintf("%s-%d", i.PrevHash, i.PrevIndex))
}

func (i *TxInput) SetUid(b []byte) {
	parts := strings.Split(string(b), "-")
	if len(parts) != 2 {
		return
	}
	i.PrevHash = parts[0]
	i.PrevIndex = jutil.GetIntFromString(parts[1])
}

func (i *TxInput) Serialize() []byte {
	return nil
}

func (i *TxInput) Deserialize([]byte) {
}

func GetTxInput(prevHash string, prevIndex int) (*TxInput, error) {
	var txInput = &TxInput{
		PrevHash:  prevHash,
		PrevIndex: prevIndex,
	}
	if err := GetSpecificItem(txInput); err != nil {
		return nil, fmt.Errorf("error getting tx input; %w", err)
	}
	return txInput, nil
}
