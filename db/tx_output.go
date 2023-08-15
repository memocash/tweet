package db

import (
	"fmt"
	"github.com/jchavannes/jgo/jutil"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type TxOutput struct {
	Address [25]byte
	TxHash  [32]byte
	Index   int
	Output  []byte
}

func (o *TxOutput) GetPrefix() string {
	return PrefixTxOutput
}

func (o *TxOutput) GetUid() []byte {
	return jutil.CombineBytes(
		o.Address[:],
		o.TxHash[:],
		jutil.GetIntData(o.Index),
	)
}

func (o *TxOutput) SetUid(b []byte) {
	if len(b) != 61 {
		return
	}
	copy(o.Address[:], b[:25])
	copy(o.TxHash[:], b[25:57])
	o.Index = jutil.GetInt(b[57:])
}

func (o *TxOutput) Serialize() []byte {
	return o.Output
}

func (o *TxOutput) Deserialize(d []byte) {
	o.Output = d
}

func GetTxOutputs(addresses []wallet.Addr) ([]*TxOutput, error) {
	db, err := GetDb()
	if err != nil {
		return nil, fmt.Errorf("error getting database handler for get tx outputs; %w", err)
	}
	var txOutputs []*TxOutput
	for _, address := range addresses {
		iterKey := jutil.CombineBytes([]byte(PrefixTxOutput), []byte{Spacer}, address[:])
		iter := db.NewIterator(util.BytesPrefix(iterKey), nil)
		for iter.Next() {
			var tweetTx = new(TxOutput)
			Set(tweetTx, iter)
			txOutputs = append(txOutputs, tweetTx)
		}
		iter.Release()
		if err := iter.Error(); err != nil {
			return nil, fmt.Errorf("error iterating over tx outputs for address; %w", err)
		}
	}
	return txOutputs, nil
}
