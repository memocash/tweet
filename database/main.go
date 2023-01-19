package database

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/jchavannes/jgo/jerr"
	"github.com/jchavannes/jgo/jlog"
	"github.com/memocash/index/client/lib"
	"github.com/memocash/index/ref/bitcoin/memo"
	"github.com/memocash/index/ref/bitcoin/tx/gen"
	"github.com/memocash/index/ref/bitcoin/tx/hs"
	"github.com/memocash/index/ref/bitcoin/tx/parse"
	"github.com/memocash/index/ref/bitcoin/tx/script"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/syndtr/goleveldb/leveldb"
	"golang.org/x/crypto/scrypt"
	"io"
	"net/http"
	"time"
)

func GetDb() (*leveldb.DB, error) {
	db, err := leveldb.OpenFile("tweets.db", nil)
	if err != nil {
		return nil, jerr.Get("error opening db", err)
	}
	return db, nil
}

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
	jlog.Logf("Getting new UTXOs from database\n")
	database := Database{Db: g.Db}
	//if err != nil {
	//	return nil, jerr.Get("error getting database", err)
	//}
	//print the contents of g.Db
	client := lib.NewClient("http://localhost:26770/graphql", &database)
	address := g.Address.GetAddr()
	outputs, err := client.GetUtxos(&address)
	balance, err := client.GetBalance(&address)
	println("balance: ", balance)
	if err != nil {
		return nil, jerr.Get("error getting utxos", err)
	}
	var utxos []memo.UTXO
	pkHash := g.Address.GetPkHash()
	pkScript, err := script.P2pkh{PkHash: pkHash}.Get()
	if err != nil {
		return nil, jerr.Get("error getting pk script", err)
	}
	for _, output := range outputs {
		utxos = append(utxos, memo.UTXO{
			Input: memo.TxInput{
				PkScript:     pkScript,
				PkHash:       pkHash,
				Value:        output.Amount,
				PrevOutHash:  hs.GetTxHash(output.Tx.Hash),
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

type Wallet struct {
	Address wallet.Address
	Key     wallet.PrivateKey
	Getter  gen.InputGetter
}

func NewWallet(address wallet.Address, key wallet.PrivateKey, db *leveldb.DB) Wallet {
	return Wallet{
		Address: address,
		Key:     key,
		Getter:  &InputGetter{Address: address, Db: db},
	}
}

func MakePost(wlt Wallet, message string) ([]byte, error) {
	memoTx, err := buildTx(wlt, script.Post{Message: message})
	if err != nil {
		return nil, jerr.Get("error generating memo tx", err)
	}
	txInfo := parse.GetTxInfo(memoTx)
	txInfo.Print()
	completeTransaction(memoTx, err)
	return memoTx.GetHash(), nil
}
func MakeReply(wallet Wallet, parentHash []byte, message string) ([]byte, error) {
	memoTx, err := buildTx(wallet, script.Reply{Message: message, TxHash: parentHash})
	if err != nil {
		return nil, jerr.Get("error generating memo tx", err)
	}
	txInfo := parse.GetTxInfo(memoTx)
	txInfo.Print()
	completeTransaction(memoTx, err)
	return memoTx.GetHash(), nil
}

func FundTwitterAddress(utxo memo.UTXO, key wallet.PrivateKey, address wallet.Address) error {
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
		return jerr.Get("error generating memo tx", err)
	}
	txInfo := parse.GetTxInfo(memoTx)
	txInfo.Print()
	completeTransaction(memoTx, err)
	time.Sleep(2 * time.Second)
	return nil
}
func PartialFund(utxo memo.UTXO, key wallet.PrivateKey, address wallet.Address, amount int64) error {
	memoTx, err := gen.Tx(gen.TxRequest{
		InputsToUse: []memo.UTXO{utxo},
		Change: wallet.Change{
			Main: key.GetAddress(),
		},
		Outputs: []*memo.Output{{
			Amount: memo.GetMaxSendFromCount(amount, 1),
			Script: script.P2pkh{PkHash: address.GetPkHash()},
		}},
		KeyRing: wallet.KeyRing{
			Keys: []wallet.PrivateKey{key},
		},
	})
	if err != nil {
		return jerr.Get("error generating memo tx", err)
	}
	txInfo := parse.GetTxInfo(memoTx)
	txInfo.Print()
	completeTransaction(memoTx, err)
	time.Sleep(1 * time.Second)
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
	txInfo := parse.GetTxInfo(memoTx)
	txInfo.Print()
	completeTransaction(memoTx, err)
	time.Sleep(1 * time.Second)
	return nil
}
func UpdateName(wlt Wallet, name string) error {
	memoTx, err := buildTx(wlt, script.SetName{Name: name})
	if err != nil {
		return jerr.Get("error generating memo tx", err)
	}
	txInfo := parse.GetTxInfo(memoTx)
	txInfo.Print()
	completeTransaction(memoTx, err)
	return nil
}

func UpdateProfileText(wlt Wallet, profile string) error {
	if profile == "" {
		profile = " "
	}
	memoTx, err := buildTx(wlt, script.Profile{Text: profile})
	if err != nil {
		return jerr.Get("error generating memo tx", err)
	}
	txInfo := parse.GetTxInfo(memoTx)
	txInfo.Print()
	completeTransaction(memoTx, err)
	return nil
}

func UpdateProfilePic(wlt Wallet, url string) error {
	memoTx, err := buildTx(wlt, script.ProfilePic{Url: url})
	if err != nil {
		return jerr.Get("error generating memo tx", err)
	}
	println("tx", memoTx.GetHash())
	txInfo := parse.GetTxInfo(memoTx)
	txInfo.Print()
	completeTransaction(memoTx, err)
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
func completeTransaction(memoTx *memo.Tx, err error) {
	if err != nil {
		jerr.Get("error running basic query", err).Fatal()
	}
	jsonData := map[string]interface{}{
		"query": `mutation ($raw: String!) {
					broadcast(raw: $raw)
				}`,
		"variables": map[string]string{
			"raw": hex.EncodeToString(memo.GetRaw(memoTx.MsgTx)),
		},
	}
	jsonValue, _ := json.Marshal(jsonData)
	request, err := http.NewRequest("POST", "http://localhost:26770/graphql", bytes.NewBuffer(jsonValue))
	if err != nil {
		jerr.Get("Making a new request failed\n", err).Fatal()
	}
	request.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: time.Second * 10}
	response, err := client.Do(request)
	fmt.Printf("%#v\n", response)
	if err != nil {
		jerr.Get("The HTTP request failed with error %s\n", err).Fatal()
	}
}

var salt = []byte{0xfe, 0xa9, 0xe9, 0x4c, 0xd9, 0x84, 0x50, 0x3d}

func SetSalt(newSalt []byte) {
	salt = newSalt
}

// https://golang.org/pkg/crypto/cipher/#example_NewCFBEncrypter
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

// https://golang.org/pkg/crypto/cipher/#example_NewCFBDecrypter
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

// https://godoc.org/golang.org/x/crypto/scrypt#example-package
func GenerateEncryptionKeyFromPassword(password string) ([]byte, error) {
	dk, err := scrypt.Key([]byte(password), salt, 1<<15, 8, 1, 32)
	if err != nil {
		return []byte{}, jerr.Get("error generating key", err)
	}
	return dk, nil
}
