package strm

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/db"
	tweetWallet "github.com/memocash/tweet/wallet"
)

type Stream struct {
	UserID int64
	Owner  wallet.Addr
	Wallet tweetWallet.Wallet
}

func GetStreams(onlyFunded bool) ([]Stream, error) {
	var streams []Stream
	addressKeys, err := db.GetAllAddressKey()
	if err != nil {
		return nil, jerr.Get("error getting address keys for database for stream configs", err)
	}
	for _, addressKey := range addressKeys {
		decryptedWifByte, err := tweetWallet.DecryptFromDb(addressKey.Key)
		if err != nil {
			return nil, jerr.Get("error decrypting", err)
		}
		wif := string(decryptedWifByte)
		walletKey, err := wallet.ImportPrivateKey(wif)
		if err != nil {
			return nil, jerr.Get("error importing private key", err)
		}
		if onlyFunded {
			inputGetter := tweetWallet.InputGetter{
				Address: walletKey.GetAddress(),
			}
			outputs, err := inputGetter.GetUTXOs(nil)
			if err != nil {
				return nil, jerr.Get("error getting utxos", err)
			}
			balance := int64(0)
			for _, output := range outputs {
				balance += output.Input.Value
			}
			if balance <= 800 {
				continue
			}
		}
		wlt := tweetWallet.NewWallet(walletKey.GetAddress(), walletKey)
		if err != nil {
			return nil, jerr.Get("error parsing user id", err)
		}
		streams = append(streams, Stream{
			UserID: addressKey.UserID,
			Owner:  addressKey.Address,
			Wallet: wlt,
		})
	}
	return streams, nil
}
