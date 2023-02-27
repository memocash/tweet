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
	"github.com/memocash/tweet/tweets/obj"
	tweetWallet "github.com/memocash/tweet/wallet"
	"github.com/syndtr/goleveldb/leveldb"
	"strconv"
)

func getBotStreams(cryptKey []byte) ([]config.Stream, error) {
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
		if balance > 800 {
			wlt := tweetWallet.NewWallet(walletKey.GetAddress(), walletKey)
			botStreams = append(botStreams, config.Stream{
				Key:    decryptedKey,
				Name:   addressKey.TwitterName,
				Sender: addressKey.Address,
				Wallet: wlt,
			})
		}
	}
	return botStreams, nil
}

func createBotStream(b *Bot, twitterName string, senderAddress string, tx graph.Tx, coinIndex uint32) (*obj.AccountKey, *tweetWallet.Wallet, error) {
	//check if the value of the transaction is less than 5,000 or this address already has a bot for this account in the database
	botExists := false
	_, err := db.GetAddressKey(senderAddress, twitterName)
	if err != nil && !errors.Is(err, leveldb.ErrNotFound) {
		return nil, nil, jerr.Get("error getting bot from database", err)
	} else if err == nil {
		botExists = true
	}
	//check if this twitter account actually exists
	twitterExists := false
	if _, _, err := b.TweetClient.Users.Show(&twitter.UserShowParams{ScreenName: twitterName}); err == nil {
		twitterExists = true
	}
	if !twitterExists || tx.Outputs[coinIndex].Amount < 5000 {
		if tx.Outputs[coinIndex].Amount < 546 {
			return nil, nil, nil
		}
		errMsg := ""
		if !twitterExists {
			errMsg = fmt.Sprintf("Twitter account @%s does not exist", twitterName)
		} else {
			errMsg = fmt.Sprintf("You need to send at least 5,000 satoshis to create a bot for the account @%s", twitterName)
		}
		err = refund(tx, b, coinIndex, senderAddress, errMsg)
		if err != nil {
			return nil, nil, jerr.Get("error refunding", err)
		}
		return nil, nil, nil
	}
	println(b.Addresses[0])
	var newKey wallet.PrivateKey
	var newAddr wallet.Address
	numStreamBytes, err := b.Db.Get([]byte("memobot-num-streams"), nil)
	if err != nil {
		return nil, nil, jerr.Get("error getting num-streams", err)
	}
	numStream, err := strconv.ParseUint(string(numStreamBytes), 10, 64)
	if err != nil {
		return nil, nil, jerr.Get("error parsing num-streams", err)
	}
	//convert numStream to a uint
	numStreamUint := uint(numStream)
	if botExists {
		//get the key from the database
		//decrypt
		addressKey, err := db.GetAddressKey(senderAddress, twitterName)
		if err != nil {
			return nil, nil, jerr.Get("error getting key from database", err)
		}
		decryptedKey, err := tweetWallet.Decrypt(addressKey.Key, b.Crypt)
		if err != nil {
			return nil, nil, jerr.Get("error decrypting key", err)
		}
		newKey, err = wallet.ImportPrivateKey(string(decryptedKey))
		if err != nil {
			return nil, nil, jerr.Get("error importing private key", err)
		}
		newAddr = newKey.GetAddress()
	} else {
		path := wallet.GetBip44Path(wallet.Bip44CoinTypeBTC, numStreamUint+1, false)
		keyPointer, err := b.Mnemonic.GetPath(path)
		newKey = *keyPointer
		if err != nil {
			return nil, nil, jerr.Get("error getting path", err)
		}
		newAddr = newKey.GetAddress()
	}
	pkScript, err := hex.DecodeString(tx.Outputs[coinIndex].Script)
	if err != nil {
		return nil, nil, jerr.Get("error decoding script pk script for create bot", err)
	}
	if err := tweetWallet.FundTwitterAddress(memo.UTXO{Input: memo.TxInput{
		Value:        tx.Outputs[coinIndex].Amount,
		PrevOutHash:  hs.GetTxHash(tx.Hash),
		PrevOutIndex: coinIndex,
		PkHash:       b.Key.GetAddress().GetPkHash(),
		PkScript:     pkScript,
	}}, b.Key, newAddr); err != nil {
		return nil, nil, jerr.Get("error funding twitter address", err)
	}
	newWallet := tweetWallet.NewWallet(newAddr, newKey)
	if !botExists {
		err = updateProfile(b, newWallet, twitterName, senderAddress)
		if err != nil {
			return nil, nil, jerr.Get("error updating profile", err)
		}
	}
	if b.Verbose {
		jlog.Logf("Create bot stream Address: " + newAddr.GetEncoded())
	}
	if !botExists {
		err = b.Db.Put([]byte("memobot-num-streams"), []byte(strconv.FormatUint(uint64(numStreamUint+1), 10)), nil)
		if err != nil {
			return nil, nil, jerr.Get("error putting num-streams", err)
		}
		//add a field to the database that links the sending address and twitter name to the new key
		//encrypt
		encryptedKey, err := tweetWallet.Encrypt([]byte(newKey.GetBase58Compressed()), b.Crypt)
		if err != nil {
			return nil, nil, jerr.Get("error encrypting key", err)
		}
		if err := db.Save([]db.ObjectI{&db.AddressLinkedKey{
			Address:     senderAddress,
			TwitterName: twitterName,
			Key:         encryptedKey,
		}}); err != nil {
			return nil, nil, jerr.Get("error updating linked-"+senderAddress+"-"+twitterName, err)
		}
	}
	accountKey := obj.GetAccountKeyFromArgs([]string{newKey.GetBase58Compressed(), twitterName})
	return &accountKey, &newWallet, nil
}
