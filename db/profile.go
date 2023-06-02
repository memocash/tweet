package db

import (
	"fmt"
	"github.com/jchavannes/jgo/jutil"
)

type Profile struct {
	Owner   [25]byte
	UserID  int64
	Profile []byte
}

func (o *Profile) GetPrefix() string {
	return PrefixProfile
}

func (o *Profile) GetUid() []byte {
	return jutil.CombineBytes(
		o.Owner[:],
		jutil.GetInt64DataBig(o.UserID),
	)
}

func (o *Profile) SetUid(b []byte) {
	if len(b) != 33 {
		return
	}
	copy(o.Owner[:], b[:25])
	o.UserID = jutil.GetInt64Big(b[25:])
}

func (o *Profile) Serialize() []byte {
	return o.Profile
}

func (o *Profile) Deserialize(d []byte) {
	o.Profile = d
}

func GetProfile(address [25]byte, userId int64) (*Profile, error) {
	var profile = &Profile{
		Owner:  address,
		UserID: userId,
	}
	if err := GetSpecificItem(profile); err != nil {
		return nil, fmt.Errorf("error getting specific profile from db; %w", err)
	}
	return profile, nil
}
