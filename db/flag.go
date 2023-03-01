package db

import (
	"encoding/json"
	"fmt"
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
	Address     string
	TwitterName string
	Flags       Flags
}

func (f *Flag) GetPrefix() string {
	return PrefixFlag
}

func (f *Flag) GetUid() []byte {
	return []byte(fmt.Sprintf("%s-%s", f.Address, f.TwitterName))
}

func (f *Flag) SetUid(b []byte) {
	parts := strings.Split(string(b), "-")
	if len(parts) != 2 {
		return
	}
	f.Address = parts[0]
	f.TwitterName = parts[1]
}

func (f *Flag) Serialize() []byte {
	flagsBytes, _ := json.Marshal(f.Flags)
	return flagsBytes
}

func (f *Flag) Deserialize(d []byte) {
	json.Unmarshal(d, &f.Flags)
}

func GetFlag(address, twitterName string) (*Flag, error) {
	var flag = &Flag{
		Address:     address,
		TwitterName: twitterName,
	}
	if err := GetSpecificItem(flag); err != nil {
		return nil, fmt.Errorf("error getting flag from db; %w", err)
	}
	return flag, nil
}
