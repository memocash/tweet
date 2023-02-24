package wallet

import (
	"bytes"
	"github.com/jchavannes/jgo/jerr"
	"github.com/jchavannes/jgo/jlog"
	"github.com/memocash/index/client/lib"
	"github.com/memocash/index/ref/bitcoin/memo"
	"github.com/memocash/index/ref/bitcoin/tx/hs"
	"github.com/memocash/index/ref/bitcoin/tx/script"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/syndtr/goleveldb/leveldb"
)

type InputGetter struct {
	Address wallet.Address
	UTXOs   []memo.UTXO
	Db      *leveldb.DB
	reset   bool
}

func (g *InputGetter) SetPkHashesToUse([][]byte) {
}

func (g *InputGetter) GetUTXOs(*memo.UTXORequest) ([]memo.UTXO, error) {
	if g.reset && len(g.UTXOs) > 0 {
		jlog.Logf("Using existing UTXOS: %d\n", len(g.UTXOs))
		g.reset = false
		return g.UTXOs, nil
	}
	database := Database{Db: g.Db}
	client := lib.NewClient("http://localhost:26770/graphql", &database)
	address := g.Address.GetAddr()
	outputs, err := client.GetUtxos(address)
	if err != nil {
		return nil, jerr.Get("error getting utxos from database for input getter", err)
	}
	balance, err := client.GetBalance(address)
	if err != nil {
		return nil, jerr.Get("error getting balance from database for input getter", err)
	}
	jlog.Logf("address balance (input getter): %s %d (outs: %d)\n", address, balance, len(outputs))
	var utxos []memo.UTXO
	pkHash := g.Address.GetPkHash()
	pkScript, err := script.P2pkh{PkHash: pkHash}.Get()
	if err != nil {
		return nil, jerr.Get("error getting pk script", err)
	}
	for _, output := range outputs {
		txHash := hs.GetTxHash(output.Tx.Hash)
		for _, utxo := range g.UTXOs {
			if bytes.Equal(utxo.Input.PrevOutHash, txHash) &&
				utxo.Input.PrevOutIndex == uint32(output.Index) {
				continue
			}
		}
		utxos = append(utxos, memo.UTXO{
			Input: memo.TxInput{
				PkScript:     pkScript,
				PkHash:       pkHash,
				Value:        output.Amount,
				PrevOutHash:  txHash,
				PrevOutIndex: uint32(output.Index),
			},
		})
	}
	g.UTXOs = utxos
	return utxos, nil
}

func (g *InputGetter) MarkUTXOsUsed(used []memo.UTXO) {
	for i := 0; i < len(g.UTXOs); i++ {
		for j := 0; j < len(used); j++ {
			if g.UTXOs[i].IsEqual(used[j]) {
				//remove g.UTXOs[i] from the list
				g.UTXOs = append(g.UTXOs[:i], g.UTXOs[i+1:]...)
				//decrement i so we don't go out of bounds
				i--
				break
			}
		}
	}
}

func (g *InputGetter) AddChangeUTXO(new memo.UTXO) {
	g.UTXOs = append(g.UTXOs, new)
}

func (g *InputGetter) NewTx() {
	g.reset = true
}
