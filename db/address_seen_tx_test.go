package db_test

import (
	"github.com/jchavannes/btcd/chaincfg/chainhash"
	"github.com/memocash/index/ref/bitcoin/util/testing/test_tx"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/db"
	"log"
	"testing"
	"time"
)

func TestAddressSeenTx(t *testing.T) {
	address, err := wallet.GetAddrFromString(test_tx.Address1String)
	if err != nil {
		t.Error(err)
		return
	}
	seen := time.Date(2019, 1, 1, 0, 0, 0, 0, time.Local)
	txHash, err := chainhash.NewHashFromStr(test_tx.GenericTxHashString0)
	if err != nil {
		t.Error(err)
		return
	}
	var addressSeenTx = &db.AddressSeenTx{
		Address: *address,
		Seen:    seen,
		TxHash:  *txHash,
	}
	uid := addressSeenTx.GetUid()
	log.Printf("uid: %x\n", uid)
	var newAddressSeenTx = new(db.AddressSeenTx)
	newAddressSeenTx.SetUid(uid)
	if newAddressSeenTx.Address != *address {
		t.Errorf("address mismatch, got: %s, expected: %s", wallet.Addr(newAddressSeenTx.Address), address)
		return
	}
	if newAddressSeenTx.Seen != seen {
		t.Errorf("seen mismatch, got: %s, expected: %s",
			newAddressSeenTx.Seen.Format(time.RFC3339), seen.Format(time.RFC3339))
		return
	}
	if newAddressSeenTx.TxHash != *txHash {
		t.Errorf("tx hash mismatch, got: %s, expected: %s", chainhash.Hash(newAddressSeenTx.TxHash), txHash)
		return
	}
}
