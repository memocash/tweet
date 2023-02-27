package bot

import (
	"encoding/hex"
	"github.com/jchavannes/jgo/jerr"
	"github.com/jchavannes/jgo/jlog"
	"github.com/memocash/index/ref/bitcoin/memo"
	"github.com/memocash/index/ref/bitcoin/tx/hs"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/graph"
	tweetWallet "github.com/memocash/tweet/wallet"
)

func refund(tx graph.Tx, b *Bot, coinIndex uint32, senderAddress string, errMsg string) error {
	_, err := b.SafeUpdate()
	if err != nil {
		return jerr.Get("error updating stream", err)
	}
	jlog.Logf("Sending refund error message to %s: %s\n", senderAddress, errMsg)
	sentToMainBot := false
	//check all the outputs to see if any of them match the bot's address, if not, return nil, if so, continue with the function
	for _, output := range tx.Outputs {
		if output.Lock.Address == b.Addresses[0] {
			sentToMainBot = true
			break
		}
	}
	if !sentToMainBot {
		return nil
	}
	//handle sending back money
	//not enough to send back
	if memo.GetMaxSendFromCount(tx.Outputs[coinIndex].Amount, 1) <= 0 {
		if b.Verbose {
			jlog.Log("Not enough funds to refund")
		}
		return nil
	}
	//create a transaction with the sender address and the amount of the transaction
	pkScript, err := hex.DecodeString(tx.Outputs[coinIndex].Script)
	if err != nil {
		return jerr.Get("error decoding script pk script for refund", err)
	}
	if err := tweetWallet.SendToTwitterAddress(memo.UTXO{Input: memo.TxInput{
		Value:        tx.Outputs[coinIndex].Amount,
		PrevOutHash:  hs.GetTxHash(tx.Hash),
		PrevOutIndex: coinIndex,
		PkHash:       wallet.GetAddressFromString(b.Addresses[0]).GetPkHash(),
		PkScript:     pkScript,
	}}, b.Key, wallet.GetAddressFromString(senderAddress), errMsg); err != nil {
		return jerr.Get("error sending money back", err)
	}
	return nil
}
