package db

import (
	"fmt"
	"github.com/jchavannes/jgo/jutil"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type AddressLinkedKey struct {
	Address [25]byte
	UserID  int64
	Key     []byte
}

func (k *AddressLinkedKey) GetPrefix() string {
	return PrefixAddressKey
}

func (k *AddressLinkedKey) GetUid() []byte {
	return []byte(fmt.Sprintf("%s-%d", k.Address, k.UserID))
}

func (k *AddressLinkedKey) SetUid(b []byte) {
	if len(b) != 33 {
		return
	}
	copy(k.Address[:], b[:25])
	k.UserID = jutil.GetInt64Big(b[25:])
}

func (k *AddressLinkedKey) Serialize() []byte {
	return k.Key
}

func (k *AddressLinkedKey) Deserialize(d []byte) {
	k.Key = d
}

func GetAddressKey(address [25]byte, userId int64) (*AddressLinkedKey, error) {
	var addressKey = &AddressLinkedKey{
		Address: address,
		UserID:  userId,
	}
	if err := GetSpecificItem(addressKey); err != nil {
		return nil, fmt.Errorf("error getting address key from db; %w", err)
	}
	return addressKey, nil
}

func GetAllAddressKey() ([]*AddressLinkedKey, error) {
	db, err := GetDb()
	if err != nil {
		return nil, fmt.Errorf("error getting database handler for get all address keys; %w", err)
	}
	iter := db.NewIterator(util.BytesPrefix([]byte(fmt.Sprintf("%s-", PrefixAddressKey))), nil)
	defer iter.Release()
	var addressKeys []*AddressLinkedKey
	for iter.Next() {
		var addressKey = new(AddressLinkedKey)
		Set(addressKey, iter)
		addressKeys = append(addressKeys, addressKey)
	}
	if err := iter.Error(); err != nil {
		return nil, fmt.Errorf("error iterating over all address keys; %w", err)
	}
	return addressKeys, nil
}
