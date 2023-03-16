package db

import (
	"fmt"
	"github.com/jchavannes/jgo/jutil"
	"strings"
)

type Profile struct {
	Address string
	UserID  int64
	Profile []byte
}

func (o *Profile) GetPrefix() string {
	return PrefixProfile
}

func (o *Profile) GetUid() []byte {
	return []byte(fmt.Sprintf("%s-%d", o.Address, o.UserID))
}

func (o *Profile) SetUid(b []byte) {
	parts := strings.Split(string(b), "-")
	if len(parts) != 2 {
		return
	}
	o.Address = parts[0]
	o.UserID = jutil.GetInt64FromString(strings.TrimLeft(parts[1], "0"))
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
		UserID:  userId,
	}
	if err := GetSpecificItem(profile); err != nil {
		return nil, fmt.Errorf("error getting specific profile from db; %w", err)
	}
	return profile, nil
}
