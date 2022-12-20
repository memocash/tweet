package tweets

import (
	"github.com/jchavannes/jgo/jlog"
	"github.com/memocash/index/ref/bitcoin/wallet"
)

type AccountKey struct {
	Account string
	Key     wallet.PrivateKey
	Address wallet.Address
}

func (t AccountKey) GetAddress() wallet.Address {
	return t.GetAddress()
}

func GetAccountKeyFromArgs(args []string) AccountKey {
	key, _ := wallet.ImportPrivateKey(args[0])
	address := key.GetAddress()
	jlog.Logf("Using address: %s\n", address.GetEncoded())
	account := args[1]
	return AccountKey{
		Account: account,
		Key:     key,
		Address: address,
	}
}
