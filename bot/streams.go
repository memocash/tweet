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
	"github.com/memocash/tweet/config"
	"github.com/memocash/tweet/db"
	"github.com/memocash/tweet/graph"
	tweetWallet "github.com/memocash/tweet/wallet"
	"github.com/syndtr/goleveldb/leveldb"
	"log"
)

func getBotStreams(cryptKey []byte, onlyFunded bool) ([]config.Stream, error) {
	botStreams := make([]config.Stream, 0)
	addressKeys, err := db.GetAllAddressKey()
	if err != nil {
		return nil, jerr.Get("error getting address keys for database for stream configs", err)
	}
	for _, addressKey := range addressKeys {
		decryptedKeyByte, err := tweetWallet.Decrypt(addressKey.Key, cryptKey)
		if err != nil {
			return nil, jerr.Get("error decrypting", err)
		}
		decryptedKey := string(decryptedKeyByte)
		walletKey, err := wallet.ImportPrivateKey(decryptedKey)
		if err != nil {
			return nil, jerr.Get("error importing private key", err)
		}
		//check the balance of the new key
		inputGetter := tweetWallet.InputGetter{
			Address: walletKey.GetAddress(),
		}
		outputs, err := inputGetter.GetUTXOs(nil)
		if err != nil {
			return nil, jerr.Get("error getting utxos", err)
		}
		//if the balance is greater than 800, add the twitterName and newKey to the botStreams
		balance := int64(0)
		for _, output := range outputs {
			balance += output.Input.Value
		}
		if balance > 800 || !onlyFunded {
			wlt := tweetWallet.NewWallet(walletKey.GetAddress(), walletKey)
			if err != nil {
				return nil, jerr.Get("error parsing user id", err)
			}
			botStreams = append(botStreams, config.Stream{
				Key:    decryptedKey,
				UserID: addressKey.UserID,
				Sender: wallet.Addr(addressKey.Address).String(),
				Wallet: wlt,
			})
		}
	}
	return botStreams, nil
}

func createBotStream(b *Bot, twitterAccount *twitter.User, senderAddress string, tx graph.Tx, coinIndex uint32, historyNum int) error {
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
		decryptedKey, err := tweetWallet.Decrypt(addressKey.Key, b.Crypt)
		if err != nil {
			return jerr.Get("error decrypting key", err)
		}
		newKey, err = wallet.ImportPrivateKey(string(decryptedKey))
		if err != nil {
			return jerr.Get("error importing private key", err)
		}
		newAddr = newKey.GetAddress()
	} else {
		path := wallet.GetBip44Path(wallet.Bip44CoinTypeBTC, numStreamUint+1, false)
		keyPointer, err := b.Mnemonic.GetPath(path)
		newKey = *keyPointer
		if err != nil {
			return jerr.Get("error getting path", err)
		}
		newAddr = newKey.GetAddress()
	}
	if b.Verbose {
		jlog.Logf("Create bot stream Address: " + newAddr.GetEncoded())
	}
	if !botExists {
		log.Println("saving bot stream")
		if err := db.Save([]db.ObjectI{&db.BotStreamsCount{Count: int(numStreamUint + 1)}}); err != nil {
			return jerr.Get("error saving bot streams count", err)
		}
		encryptedKey, err := tweetWallet.Encrypt([]byte(newKey.GetBase58Compressed()), b.Crypt)
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
