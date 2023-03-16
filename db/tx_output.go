package db

import (
	"fmt"
	"github.com/jchavannes/jgo/jutil"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type TxOutput struct {
	Address [25]byte
	TxHash  string
	Index   int
	Output  []byte
}

func (o *TxOutput) GetPrefix() string {
	return PrefixTxOutput
}

func (o *TxOutput) GetUid() []byte {
	return []byte(fmt.Sprintf("%s-%s-%d", o.Address, o.TxHash, o.Index))
}

func (o *TxOutput) SetUid(b []byte) {
	if len(b) != 63 || b[25] != '-' || b[58] != '-' {
		return
	}
	copy(o.Address[:], b[:25])
	o.TxHash = string(b[26:58])
	o.Index = jutil.GetInt(b[59:])
}

func (o *TxOutput) Serialize() []byte {
	return o.Output
}

func (o *TxOutput) Deserialize(d []byte) {
	o.Output = d
}

func GetTxOutputs(address [25]byte) ([]*TxOutput, error) {
	db, err := GetDb()
	if err != nil {
		return nil, fmt.Errorf("error getting database handler for get tx outputs; %w", err)
	}
	iter := db.NewIterator(util.BytesPrefix([]byte(fmt.Sprintf("%s-%s-", PrefixTxOutput, address))), nil)
	defer iter.Release()
	var txOutputs []*TxOutput
	for iter.Next() {
		var tweetTx = new(TxOutput)
		Set(tweetTx, iter)
		txOutputs = append(txOutputs, tweetTx)
	}
	if err := iter.Error(); err != nil {
		return nil, fmt.Errorf("error iterating over tx outputs for address; %w", err)
	}
	return txOutputs, nil
}
