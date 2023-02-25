package db

import (
	"fmt"
	"github.com/jchavannes/jgo/jutil"
	"github.com/syndtr/goleveldb/leveldb/util"
	"time"
)

type AddressSeenTx struct {
	Address [25]byte
	Seen    time.Time
	TxHash  [32]byte
}

func (t *AddressSeenTx) GetPrefix() string {
	return PrefixAddressSeenTx
}

func (t *AddressSeenTx) GetUid() []byte {
	return jutil.CombineBytes(
		t.Address[:],
		jutil.GetTimeByteBig(t.Seen),
		t.TxHash[:],
	)
}

func (t *AddressSeenTx) SetUid(b []byte) {
	if len(b) != 65 {
		return
	}
	copy(t.Address[:], b[:25])
	t.Seen = jutil.GetByteTimeBig(b[25:33])
	copy(t.TxHash[:], b[33:65])
}

func (t *AddressSeenTx) Serialize() []byte {
	return nil
}

func (t *AddressSeenTx) Deserialize([]byte) {
}

func GetRecentAddressSeenTx(address [25]byte) (*AddressSeenTx, error) {
	var addressSeenTx = &AddressSeenTx{}
	if err := GetLastItem(addressSeenTx, address[:]); err != nil {
		return nil, fmt.Errorf("error getting last address seen tx item; %w", err)
	}
	return addressSeenTx, nil
}

func GetAllAddressSeenTx() ([]*AddressSeenTx, error) {
	db, err := GetDb()
	if err != nil {
		return nil, fmt.Errorf("error getting database handler for get all address seen txs; %w", err)
	}
	iter := db.NewIterator(util.BytesPrefix([]byte(fmt.Sprintf("%s-", PrefixAddressSeenTx))), nil)
	defer iter.Release()
	var addressSeenTxs []*AddressSeenTx
	for iter.Next() {
		var addressSeenTx = new(AddressSeenTx)
		Set(addressSeenTx, iter)
		addressSeenTxs = append(addressSeenTxs, addressSeenTx)
	}
	return addressSeenTxs, nil
}
