package database

import (
	"encoding/json"
	"fmt"
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/index/client/lib"
	"github.com/memocash/index/client/lib/graph"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"strings"
)
type Database struct {
	Db *leveldb.DB
}

func GetClient() (*lib.Client, error) {
	database, err := NewDatabase()
	if err != nil {
		return nil, jerr.Get("error getting database", err)
	}
	return lib.NewClient(database), nil
}

func NewDatabase() (*Database, error) {
	db, err := leveldb.OpenFile("tweets.db", nil)
	if err != nil {
		return nil, jerr.Get("error opening database", err)
	}
	return &Database{
		Db: db,
	}, nil
}

func (d *Database) GetAddressBalance(address *wallet.Addr) (int64, error) {
	//query := "" +
	//	"SELECT " +
	//	"   outputs.address, " +
	//	"   IFNULL(SUM(CASE WHEN inputs.hash IS NULL THEN outputs.value ELSE 0 END), 0) AS balance " +
	//	"FROM outputs " +
	//	"LEFT JOIN inputs ON (inputs.prev_hash = outputs.hash AND inputs.prev_index = outputs.`index`) " +
	//	"WHERE outputs.address = ? " +
	//	"GROUP BY outputs.address "
	//var result struct {
	//	Address string
	//	Balance int64
	//}
	//if err := d.Db.QueryRow(query, address.String()).Scan(&result.Address, &result.Balance); err != nil {
	//	return 0, jerr.Get("error getting address balance exec query", err)
	//}
	return 0, nil
}

func (d *Database) GetAddressHeight(address *wallet.Addr) (int64, error) {
	iter := d.Db.NewIterator(util.BytesPrefix([]byte("output-"+ address.String())), nil)
	var txhashes []string
	for iter.Next() {
		//get txhash from the name of the output
		txhash := strings.Split(string(iter.Key()), "-")[2]
		txhashes = append(txhashes, txhash)
	}
	iter.Release()
	err := iter.Error()
	if err != nil {
		return 0, jerr.Get("error getting address height", err)
	}
	var blockHashes []string
	for _, txhash := range txhashes {
		iter := d.Db.NewIterator(util.BytesPrefix([]byte("txblock-"+ txhash)), nil)
		for iter.Next() {
			blockHash := strings.Split(string(iter.Key()), "-")[2]
			blockHashes = append(blockHashes, blockHash)
		}
		iter.Release()
		err := iter.Error()
		if err != nil {
			return 0, jerr.Get("error getting address height", err)
		}
	}
	var heights []int64
	for _, blockHash := range blockHashes {
		block,err := d.Db.Get([]byte("block-"+ blockHash), nil)
		if err != nil {
			return 0, jerr.Get("error getting address height", err)
		}
		var blockData graph.Block
		err = json.Unmarshal(block, &blockData)
		if err != nil {
			return 0, jerr.Get("error getting address height", err)
		}
		heights = append(heights, blockData.Height)
	}
	var max int64 = 0
	for _, height := range heights {
		if height > max {
			max = height
		}
	}
	//using this list of txhashes, get all the blocks, and get the max height
	return max, nil
}

func (d *Database) GetUtxos(address *wallet.Addr) ([]graph.Output, error) {
	//iterate over all outputs, and search if an input field key exists that matches "input-hash-index"
	var utxos []graph.Output
	//create an iterator for the prefix "output-address"
	iter := d.Db.NewIterator(util.BytesPrefix([]byte("output-"+ address.String())), nil)
	for iter.Next() {
		//check if the input exists
		var output graph.Output
		err := json.Unmarshal(iter.Value(), &output)
		if err != nil {
			return nil, jerr.Get("error getting utxos", err)
		}
		//get txhash from the name of the output
		txhash := strings.Split(string(iter.Key()), "-")[2]
		//check if the input exists
		inputKey := []byte(fmt.Sprintf("input-%s-%d", txhash, output.Index))
		_, err = d.Db.Get(inputKey, nil)
		if err != nil {
			if err == leveldb.ErrNotFound {
				//if the input doesn't exist, add the output to the utxos array
				utxos = append(utxos, output)
			} else {
				return nil, jerr.Get("error getting utxos", err)
			}
		}
	}
	iter.Release()
	err := iter.Error()
	if err != nil {
		return nil, jerr.Get("error getting utxos", err)
	}
	return utxos, nil
}

func (d *Database) SaveTxs(txs []graph.Tx) error {
	for _, tx := range txs {
		//marshal the tx into a byte array
		for _, input := range tx.Inputs {
			//input-prevHash-prevIndex
			key := []byte(fmt.Sprintf("input-%s-%d", input.PrevHash, input.PrevIndex))
			err := d.Db.Put(key, nil, nil)
			if err != nil {
				return jerr.Get("error saving tx", err)
			}
		}
		for _, output := range tx.Outputs {
			//output-address-txhash-index
			key := []byte(fmt.Sprintf("output-%s-%s-%d", output.Lock.Address, tx.Hash, output.Index))
			value, err := json.MarshalIndent(output, "", "  ")
			if err != nil {
				return jerr.Get("error saving tx", err)
			}
			err = d.Db.Put(key, value, nil)
			if err != nil {
				return jerr.Get("error saving tx", err)
			}
		}
		for _, block := range tx.Blocks {
			//txblock-txhash-blockhash
			//this lets us search the database by txhash and will tell us what block it's in
			key := []byte(fmt.Sprintf("txblock-%s-%s", tx.Hash, block.Hash))
			println("saving txblock", string(key))
			err := d.Db.Put(key, nil, nil)
			if err != nil {
				return jerr.Get("error saving tx", err)
			}
			//block-blockhash
			//this lets us search the database by blockhash and will tell us the height of the block
			key2 := []byte(fmt.Sprintf("block-%s", block.Hash))
			value, err := json.MarshalIndent(block, "", "  ")
			if err != nil {
				return jerr.Get("error saving tx", err)
			}
			err = d.Db.Put(key2, value, nil)
			if err != nil {
				return jerr.Get("error saving tx", err)
			}
		}
	}
	return nil
}