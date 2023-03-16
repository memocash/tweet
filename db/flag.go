package db

import (
	"encoding/json"
	"fmt"
	"github.com/jchavannes/jgo/jutil"
	"strings"
)

type Flags struct {
	Link    bool `json:"link"`
	Date    bool `json:"date"`
	CatchUp bool `json:"catch_up"`
}

func GetDefaultFlags() Flags {
	return Flags{
		Link:    true,
		Date:    false,
		CatchUp: true,
	}
}

type Flag struct {
	Address string
	UserID  int64
	Flags   Flags
}

func (f *Flag) GetPrefix() string {
	return PrefixFlag
}

func (f *Flag) GetUid() []byte {
	return []byte(fmt.Sprintf("%s-%d", f.Address, f.UserID))
}

func (f *Flag) SetUid(b []byte) {
	parts := strings.Split(string(b), "-")
	if len(parts) != 2 {
		return
	}
	f.Address = parts[0]
	f.UserID = jutil.GetInt64FromString(strings.TrimLeft(parts[1], "0"))
}

func (f *Flag) Serialize() []byte {
	flagsBytes, _ := json.Marshal(f.Flags)
	return flagsBytes
}

func (f *Flag) Deserialize(d []byte) {
	json.Unmarshal(d, &f.Flags)
}

func GetFlag(address string, userId int64) (*Flag, error) {
	var flag = &Flag{
		Address: address,
		UserID:  userId,
	}
	if err := GetSpecificItem(flag); err != nil {
		return nil, fmt.Errorf("error getting flag from db; %w", err)
	}
	return flag, nil
}
