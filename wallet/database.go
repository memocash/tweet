package wallet

import (
	"encoding/json"
	"errors"
	"github.com/jchavannes/btcd/chaincfg/chainhash"
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/index/client/lib/graph"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/db"
	"github.com/syndtr/goleveldb/leveldb"
	"time"
)

type Database struct{}

func (d *Database) GetAddressBalance(address wallet.Addr) (int64, error) {
	utxos, err := d.GetUtxos(address)
	if err != nil {
		return 0, jerr.Get("error getting address balance", err)
	}
	var balance int64 = 0
	for _, utxo := range utxos {
		balance += utxo.Amount
	}
	return balance, nil
}

func (d *Database) SetAddressLastUpdate(address wallet.Addr, updateTime time.Time) error {
	if err := db.Save([]db.ObjectI{&db.AddressWalletTime{
		Address: address,
		Time:    updateTime,
	}}); err != nil {
		return jerr.Get("error saving address wallet last update time to db", err)
	}
	return nil
}

func (d *Database) GetAddressLastUpdate(address wallet.Addr) (time.Time, error) {
	addressTime, err := db.GetAddressTime(address)
	if err != nil && !errors.Is(err, leveldb.ErrNotFound) {
		return time.Time{}, jerr.Get("error getting address wallet last update from db", err)
	}
	if addressTime != nil {
		return addressTime.Time, nil
	}
	return time.Time{}, nil
}

func (d *Database) GetUtxos(address wallet.Addr) ([]graph.Output, error) {
	var utxos []graph.Output
	dbTxOutputs, err := db.GetTxOutputs(address)
	if err != nil {
		return nil, jerr.Get("error getting tx outputs from db for get utxos", err)
	}
	for _, dbTxOutput := range dbTxOutputs {
		var output graph.Output
		if err := json.Unmarshal(dbTxOutput.Output, &output); err != nil {
			return nil, jerr.Get("error getting utxos", err)
		}
		if _, err := db.GetTxInput(dbTxOutput.TxHash, dbTxOutput.Index); err != nil {
			if !errors.Is(err, leveldb.ErrNotFound) {
				return nil, jerr.Get("error getting tx inputs from db for get utxos", err)
			}
			utxos = append(utxos, output)
		}
	}
	return utxos, nil
}

func (d *Database) SaveTxs(txs []graph.Tx) error {
	var objectsToSave []db.ObjectI
	for _, tx := range txs {
		for _, input := range tx.Inputs {
			byteHash, err := chainhash.NewHashFromStr(input.PrevHash)
			if err != nil {
				return jerr.Get("error getting hash from string", err)
			}
			objectsToSave = append(objectsToSave, &db.TxInput{
				PrevHash:  *byteHash,
				PrevIndex: input.PrevIndex,
			})
		}
		for _, output := range tx.Outputs {
			output.Tx.Hash = tx.Hash
			outputJson, err := json.MarshalIndent(output, "", "  ")
			if err != nil {
				return jerr.Get("error marshalling tx output for tx save to db", err)
			}
			_, err = wallet.GetAddrFromString(output.Lock.Address)
			if err != nil {
				continue
			}
			byteHash, err := chainhash.NewHashFromStr(tx.Hash)
			if err != nil {
				return jerr.Get("error getting hash from string", err)
			}
			objectsToSave = append(objectsToSave, &db.TxOutput{
				Address: wallet.GetAddressFromString(output.Lock.Address).GetAddr(),
				TxHash:  *byteHash,
				Index:   output.Index,
				Output:  outputJson,
			})
		}
		for _, block := range tx.Blocks {
			blockJson, err := json.MarshalIndent(block, "", "  ")
			if err != nil {
				return jerr.Get("error saving tx", err)
			}
			txByteHash, err := chainhash.NewHashFromStr(tx.Hash)
			if err != nil {
				return jerr.Get("error saving tx", err)
			}
			blockByteHash, err := chainhash.NewHashFromStr(block.Hash)
			if err != nil {
				return jerr.Get("error saving tx", err)
			}
			objectsToSave = append(objectsToSave, &db.TxBlock{
				TxHash:    *txByteHash,
				BlockHash: *blockByteHash,
			}, &db.Block{
				BlockHash: *blockByteHash,
				Block:     blockJson,
			})
		}
	}
	if err := db.Save(objectsToSave); err != nil {
		return jerr.Get("error saving wallet tx objects to db", err)
	}
	return nil
}
