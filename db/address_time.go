package db

import (
	"fmt"
	"github.com/jchavannes/jgo/jutil"
	"time"
)

type AddressWalletTime struct {
	Address string
	Time    time.Time
}

func (t *AddressWalletTime) GetPrefix() string {
	return PrefixAddressTime
}

func (t *AddressWalletTime) GetUid() []byte {
	return []byte(t.Address)
}

func (t *AddressWalletTime) SetUid(b []byte) {
	t.Address = string(b)
}

func (t *AddressWalletTime) Serialize() []byte {
	return jutil.GetTimeByte(t.Time)
}

func (t *AddressWalletTime) Deserialize(d []byte) {
	t.Time = jutil.GetByteTime(d)
}

func GetAddressTime(address string) (*AddressWalletTime, error) {
	var addressTime = &AddressWalletTime{Address: address}
	if err := GetSpecificItem(addressTime); err != nil {
		return nil, fmt.Errorf("error getting address wallet time; %w", err)
	}
	return addressTime, nil
}
