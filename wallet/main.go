package wallet

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"github.com/hasura/go-graphql-client"
	"github.com/jchavannes/btcd/chaincfg/chainhash"
	"github.com/jchavannes/jgo/jerr"
	"github.com/jchavannes/jgo/jutil"
	"github.com/memocash/index/ref/bitcoin/memo"
	"github.com/memocash/index/ref/bitcoin/tx/gen"
	"github.com/memocash/index/ref/bitcoin/tx/script"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/db"
	"github.com/memocash/tweet/graph"
	"golang.org/x/crypto/scrypt"
	"io"
	"time"
)

type Address struct {
	Outputs []Output `json:"outputs"`
}

type Output struct {
	Tx     Tx      `json:"tx"`
	Hash   string  `json:"hash"`
	Index  int     `json:"index"`
	Amount int64   `json:"amount"`
	Spends []Input `json:"spends"`
}

type Input struct {
	Hash  string `json:"hash"`
	Index int    `json:"index"`
}

type Tx struct {
	Seen time.Time `json:"seen"`
}

type Profile struct {
	Name        string
	Description string
	ProfilePic  string
}

type Wallet struct {
	Address wallet.Address
	Key     wallet.PrivateKey
	Getter  gen.InputGetter
}

func NewWallet(address wallet.Address, key wallet.PrivateKey) Wallet {
	return Wallet{
		Address: address,
		Key:     key,
		Getter:  &InputGetter{Address: address},
	}
}

func MakePost(wlt Wallet, message string) (chainhash.Hash, error) {
	memoTx, err := buildTx(wlt, script.Post{Message: message})
	//check if the prefix already exists in the database
	if err != nil {
		return chainhash.Hash{}, jerr.Get("error generating memo tx make post", err)
	}
	if err := graph.Broadcast(memoTx); err != nil {
		return chainhash.Hash{}, jerr.Get("error completing transaction make post", err)
	}
	return memoTx.MsgTx.TxHash(), nil
}

func MakeReply(wallet Wallet, parentHash []byte, message string) (chainhash.Hash, error) {
	memoTx, err := buildTx(wallet, script.Reply{Message: message, TxHash: parentHash})
	if err != nil {
		return chainhash.Hash{}, jerr.Get("error generating memo tx", err)
	}
	if err := graph.Broadcast(memoTx); err != nil {
		return chainhash.Hash{}, jerr.Get("error completing transaction memo reply", err)
	}
	return memoTx.MsgTx.TxHash(), nil
}

func GetProfile(address string, date time.Time, client *graphql.Client) (*graph.Profiles, error) {
	var senderData graph.Profiles
	var variables = map[string]interface{}{"address": address}
	var startDate string
	if !jutil.IsTimeZero(date) {
		startDate = date.Format(time.RFC3339)
	} else {
		startDate = time.Date(2009, 1, 1, 0, 0, 0, 0, time.Local).Format(time.RFC3339)
	}
	variables["start"] = graph.Date(startDate)
	err := client.Query(context.Background(), &senderData, variables)
	if err != nil {
		return nil, jerr.Get("error getting sender profile", err)
	}
	return &senderData, nil
}

func FundTwitterAddress(utxo memo.UTXO, key wallet.PrivateKey, address wallet.Address, historyNum int, botExists bool) error {
	memoTx, err := gen.Tx(gen.TxRequest{
		InputsToUse: []memo.UTXO{utxo},
		Change: wallet.Change{
			Main: key.GetAddress(),
		},
		Outputs: []*memo.Output{{
			Amount: memo.GetMaxSendFromCount(utxo.Input.Value, 1),
			Script: script.P2pkh{PkHash: address.GetPkHash()},
		}},
		KeyRing: wallet.KeyRing{
			Keys: []wallet.PrivateKey{key},
		},
	})
	if err != nil {
		return jerr.Get("error generating memo tx fund twitter address", err)
	}
	if err := db.Save([]db.ObjectI{&db.SubBotCommand{
		TxHash:     memoTx.MsgTx.TxHash(),
		HistoryNum: historyNum,
		BotExists:  botExists}}); err != nil {
		return jerr.Get("error saving sub bot command", err)
	}
	if err := graph.Broadcast(memoTx); err != nil {
		return jerr.Get("error completing transaction fund twitter address", err)
	}
	return nil
}

func WithdrawAmount(utxos []memo.UTXO, key wallet.PrivateKey, address wallet.Address, amount int64) error {
	memoTx, err := gen.Tx(gen.TxRequest{
		InputsToUse: utxos,
		Change: wallet.Change{
			Main: key.GetAddress(),
		},
		Outputs: []*memo.Output{{
			Amount: amount,
			Script: script.P2pkh{PkHash: address.GetPkHash()},
		}},
		KeyRing: wallet.KeyRing{
			Keys: []wallet.PrivateKey{key},
		},
	})
	if err != nil {
		return jerr.Get("error generating memo tx", err)
	}
	if err := graph.Broadcast(memoTx); err != nil {
		return jerr.Get("error completing transaction withdraw amount", err)
	}
	return nil
}

func WithdrawAll(utxos []memo.UTXO, key wallet.PrivateKey, address wallet.Address) error {
	memoTx, err := gen.Tx(gen.TxRequest{
		InputsToUse: utxos,
		Change: wallet.Change{
			Main: key.GetAddress(),
		},
		Outputs: []*memo.Output{{
			Amount: memo.GetMaxSendForUTXOs(utxos),
			Script: script.P2pkh{PkHash: address.GetPkHash()},
		}},
		KeyRing: wallet.KeyRing{
			Keys: []wallet.PrivateKey{key},
		},
	})
	if err != nil {
		return jerr.Get("error generating memo tx", err)
	}
	if err := graph.Broadcast(memoTx); err != nil {
		return jerr.Get("error completing transaction withdraw all", err)
	}
	return nil
}

func SendToTwitterAddress(utxo memo.UTXO, key wallet.PrivateKey, address wallet.Address, errorMsg string) error {
	memoTx, err := gen.Tx(gen.TxRequest{
		InputsToUse: []memo.UTXO{utxo},
		Outputs: []*memo.Output{{
			Amount: memo.GetMaxSendFromCount(utxo.Input.Value, 1) - (int64(36 + len(errorMsg))),
			Script: script.P2pkh{PkHash: address.GetPkHash()},
		}, {
			Amount: 0,
			Script: script.Send{
				Hash:    address.GetPkHash(),
				Message: errorMsg},
		}},
		KeyRing: wallet.KeyRing{
			Keys: []wallet.PrivateKey{key},
		},
	})
	if err != nil {
		return jerr.Get("error generating memo tx send to twitter address", err)
	}
	if err := graph.Broadcast(memoTx); err != nil {
		return jerr.Get("error completing transaction send to twitter address", err)
	}
	return nil
}

func UpdateName(wlt Wallet, name string) error {
	memoTx, err := buildTx(wlt, script.SetName{Name: name})
	if err != nil {
		return jerr.Get("error generating memo tx update name", err)
	}
	if err := graph.Broadcast(memoTx); err != nil {
		return jerr.Get("error completing transaction update name", err)
	}
	return nil
}

func UpdateProfileText(wlt Wallet, profile string) error {
	if profile == "" {
		profile = " "
	}
	memoTx, err := buildTx(wlt, script.Profile{Text: profile})
	if err != nil {
		return jerr.Get("error generating memo tx update profile text", err)
	}
	if err := graph.Broadcast(memoTx); err != nil {
		return jerr.Get("error completing transaction update profile text", err)
	}
	return nil
}

func UpdateProfilePic(wlt Wallet, url string) error {
	memoTx, err := buildTx(wlt, script.ProfilePic{Url: url})
	if err != nil {
		return jerr.Get("error generating memo tx update profile pic", err)
	}
	if err := graph.Broadcast(memoTx); err != nil {
		return jerr.Get("error completing transaction update profile pic", err)
	}
	return nil
}

func buildTx(wlt Wallet, outputScript memo.Script) (*memo.Tx, error) {
	memoTx, err := gen.Tx(gen.TxRequest{
		Getter: wlt.Getter,
		Outputs: []*memo.Output{{
			Script: outputScript,
		}},
		Change: wallet.Change{Main: wlt.Address},
		KeyRing: wallet.KeyRing{
			Keys: []wallet.PrivateKey{wlt.Key},
		},
	})

	return memoTx, err
}

var salt = []byte{0xfe, 0xa9, 0xe9, 0x4c, 0xd9, 0x84, 0x50, 0x3d}

func SetSalt(newSalt []byte) {
	salt = newSalt
}

// Encrypt see: https://golang.org/pkg/crypto/cipher/#example_NewCFBEncrypter
func Encrypt(value []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return []byte{}, jerr.Get("error getting new cipher", err)
	}
	encryptedValue := make([]byte, aes.BlockSize+len(value))
	iv := encryptedValue[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return []byte{}, jerr.Get("error reading random data for iv", err)
	}
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(encryptedValue[aes.BlockSize:], value)
	return encryptedValue, nil
}

// Decrypt see: https://golang.org/pkg/crypto/cipher/#example_NewCFBDecrypter
func Decrypt(value []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return []byte{}, jerr.Get("error getting new cipher", err)
	}
	if len(value) < aes.BlockSize {
		return []byte{}, jerr.New("ciphertext too short")
	}
	iv := value[:aes.BlockSize]
	decryptedValue := make([]byte, len(value)-aes.BlockSize)
	copy(decryptedValue, value[aes.BlockSize:])
	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(decryptedValue, decryptedValue)
	return decryptedValue, nil
}

// GenerateEncryptionKeyFromPassword see: https://godoc.org/golang.org/x/crypto/scrypt#example-package
func GenerateEncryptionKeyFromPassword(password string) ([]byte, error) {
	dk, err := scrypt.Key([]byte(password), salt, 1<<15, 8, 1, 32)
	if err != nil {
		return []byte{}, jerr.Get("error generating key", err)
	}
	return dk, nil
}
