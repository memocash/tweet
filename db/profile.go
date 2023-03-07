package db

import (
	"fmt"
	"strconv"
	"strings"
)

type Profile struct {
	Address string
	UserID  string
	Profile []byte
}

func (o *Profile) GetPrefix() string {
	return PrefixProfile
}

func (o *Profile) GetUid() []byte {
	return []byte(fmt.Sprintf("%s-%s", o.Address, o.UserID))
}

func (o *Profile) SetUid(b []byte) {
	parts := strings.Split(string(b), "-")
	if len(parts) != 2 {
		return
	}
	o.Address = parts[0]
	o.UserID = parts[1]
}

func (o *Profile) Serialize() []byte {
	return o.Profile
}

func (o *Profile) Deserialize(d []byte) {
	o.Profile = d
}

func GetProfile(address string, userId int64) (*Profile, error) {
	var profile = &Profile{
		Address: address,
		UserID:  strconv.FormatInt(userId, 10),
	}
	if err := GetSpecificItem(profile); err != nil {
		return nil, fmt.Errorf("error getting specific profile from db; %w", err)
	}
	return profile, nil
}
