package db

import (
	"fmt"
	"github.com/syndtr/goleveldb/leveldb/util"
	"strings"
)

type AddressLinkedKey struct {
	Address     string
	TwitterName string
	Key         []byte
}

func (k *AddressLinkedKey) GetPrefix() string {
	return PrefixAddressKey
}

func (k *AddressLinkedKey) GetUid() []byte {
	return []byte(fmt.Sprintf("%s-%s", k.Address, k.TwitterName))
}

func (k *AddressLinkedKey) SetUid(b []byte) {
	parts := strings.Split(string(b), "-")
	if len(parts) != 2 {
		return
	}
	k.Address = parts[0]
	k.TwitterName = parts[1]
}

func (k *AddressLinkedKey) Serialize() []byte {
	return k.Key
}

func (k *AddressLinkedKey) Deserialize(d []byte) {
	k.Key = d
}

func GetAddressKey(address, twitterName string) (*AddressLinkedKey, error) {
	var addressKey = &AddressLinkedKey{
		Address:     address,
		TwitterName: twitterName,
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
