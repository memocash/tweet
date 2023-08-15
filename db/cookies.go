package db

import (
	"fmt"
)

type Cookies struct {
	CookieData []byte
}

func (c *Cookies) GetPrefix() string {
	return PrefixCookies
}

func (c *Cookies) GetUid() []byte {
	return nil
}

func (c *Cookies) SetUid(_ []byte) {
}

func (c *Cookies) Serialize() []byte {
	return c.CookieData
}

func (c *Cookies) Deserialize(d []byte) {
	c.CookieData = d
}

func GetCookies() (*Cookies, error) {
	var cookies = new(Cookies)
	if err := GetSpecificItem(cookies); err != nil {
		return nil, fmt.Errorf("error getting cookies from db; %w", err)
	}
	return cookies, nil
}
