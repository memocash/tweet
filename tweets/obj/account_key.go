package obj

import (
	"github.com/memocash/index/ref/bitcoin/wallet"
	"strconv"
)

type AccountKey struct {
	UserID  int64
	Key     wallet.PrivateKey
	Address wallet.Address
}

func (t AccountKey) GetAddress() string {
	return t.Address.GetEncoded()
}

func GetAccountKeyFromArgs(args []string) AccountKey {
	key, _ := wallet.ImportPrivateKey(args[0])
	address := key.GetAddress()
	userId, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		panic(err)
	}
	return AccountKey{
		UserID:  userId,
		Key:     key,
		Address: address,
	}
}
