package db_test

import (
	"github.com/memocash/index/ref/bitcoin/util/testing/test_tx"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/db"
	"log"
	"testing"
)

func TestAddressKey(t *testing.T) {
	address, err := wallet.GetAddrFromString(test_tx.Address1String)
	if err != nil {
		t.Error(err)
		return
	}
	userID := int64(1)
	var addressKey = &db.AddressLinkedKey{
		Address: *address,
		UserID:  userID,
	}
	uid := addressKey.GetUid()
	log.Printf("uid: %x\n", uid)
	var newAddressLinkedKey = new(db.AddressLinkedKey)
	newAddressLinkedKey.SetUid(uid)
	if newAddressLinkedKey.Address != *address {
		t.Errorf("address mismatch, got: %s, expected: %s", wallet.Addr(newAddressLinkedKey.Address), address)
		return
	}
	if newAddressLinkedKey.UserID != userID {
		t.Errorf("userID mismatch, got: %d, expected: %d",
			newAddressLinkedKey.UserID, userID)
		return
	}
}
