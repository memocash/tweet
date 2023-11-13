package bot

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/jchavannes/jgo/jerr"
	"github.com/jchavannes/jgo/jlog"
	"github.com/memocash/index/ref/bitcoin/memo"
	"github.com/memocash/index/ref/bitcoin/tx/hs"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/db"
	"github.com/memocash/tweet/graph"
	tweetWallet "github.com/memocash/tweet/wallet"
	"github.com/syndtr/goleveldb/leveldb"
)

func createStream(b *Bot, twitterAccount *twitter.User, senderAddress string, tx graph.Tx, coinIndex uint32, historyNum int) error {
	//check if the value of the transaction is less than 5,000 or this address already has a bot for this account in the database
	botExists := false
	_, err := db.GetAddressKey(wallet.GetAddressFromString(senderAddress).GetAddr(), twitterAccount.ID)
	if err != nil && !errors.Is(err, leveldb.ErrNotFound) {
		return jerr.Get("error getting bot from database", err)
	} else if err == nil {
		botExists = true
	}
	if tx.Outputs[coinIndex].Amount < 5000 {
		//edit this to also send a message even if there's less than 546 satoshis
		if tx.Outputs[coinIndex].Amount < 546 {
			return nil
		}
		errMsg := fmt.Sprintf("You need to send at least 5,000 satoshis to create a bot for the account @%s", twitterAccount.ScreenName)
		err = refund(tx, b, coinIndex, senderAddress, errMsg)
		if err != nil {
			return jerr.Get("error refunding", err)
		}
		return nil
	}
	var newKey wallet.PrivateKey
	var newAddr wallet.Address
	botStreamsCount, err := db.GetBotStreamsCount()
	if err != nil && !errors.Is(err, leveldb.ErrNotFound) {
		return jerr.Get("error getting bot streams count", err)
	}
	var numStreamUint uint
	if botStreamsCount != nil {
		numStreamUint = uint(botStreamsCount.Count)
	}
	if botExists {
		addressKey, err := db.GetAddressKey(wallet.GetAddressFromString(senderAddress).GetAddr(), twitterAccount.ID)
		if err != nil {
			return jerr.Get("error getting key from database", err)
		}
		decryptedKey, err := tweetWallet.DecryptFromDb(addressKey.Key)
		if err != nil {
			return jerr.Get("error decrypting key", err)
		}
		newKey, err = wallet.ImportPrivateKey(string(decryptedKey))
		if err != nil {
			return jerr.Get("error importing private key", err)
		}
		newAddr = newKey.GetAddress()
	} else {
		streamKey, err := GetKey(numStreamUint+1)
		if err != nil {
			return fmt.Errorf("error getting new stream key; %w", err)
		}
		newKey = *streamKey
		newAddr = newKey.GetAddress()
	}
	if b.Verbose {
		jlog.Logf("Create bot stream Address: " + newAddr.GetEncoded())
	}
	if !botExists {
		if err := db.Save([]db.ObjectI{&db.BotStreamsCount{Count: int(numStreamUint + 1)}}); err != nil {
			return jerr.Get("error saving bot streams count", err)
		}
		encryptedKey, err := tweetWallet.EncryptForDb([]byte(newKey.GetBase58Compressed()))
		if err != nil {
			return jerr.Get("error encrypting key", err)
		}
		if err := db.Save([]db.ObjectI{&db.AddressLinkedKey{
			Address: wallet.GetAddressFromString(senderAddress).GetAddr(),
			UserID:  twitterAccount.ID,
			Key:     encryptedKey,
		}}); err != nil {
			return jerr.Get("error updating linked-"+senderAddress+"-"+twitterAccount.IDStr, err)
		}
	}
	err = b.SafeUpdate()
	if err != nil {
		return jerr.Get("error updating bot", err)
	}
	pkScript, err := hex.DecodeString(tx.Outputs[coinIndex].Script)
	if err != nil {
		return jerr.Get("error decoding script pk script for create bot", err)
	}
	if err = tweetWallet.FundTwitterAddress(memo.UTXO{Input: memo.TxInput{
		Value:        tx.Outputs[coinIndex].Amount,
		PrevOutHash:  hs.GetTxHash(tx.Hash),
		PrevOutIndex: coinIndex,
		PkHash:       b.Key.GetAddress().GetPkHash(),
		PkScript:     pkScript,
	}}, b.Key, newAddr, historyNum, botExists); err != nil {
		return jerr.Get("error funding twitter address", err)
	}

	return nil
}
