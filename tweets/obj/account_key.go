package obj

import (
	"github.com/memocash/index/ref/bitcoin/wallet"
)

type AccountKey struct {
	Account string
	Key     wallet.PrivateKey
	Address wallet.Address
}

func (t AccountKey) GetAddress() string {
	return t.Address.GetEncoded()
}

func GetAccountKeyFromArgs(args []string) AccountKey {
	key, _ := wallet.ImportPrivateKey(args[0])
	address := key.GetAddress()
	account := args[1]
	return AccountKey{
		Account: account,
		Key:     key,
		Address: address,
	}
}
