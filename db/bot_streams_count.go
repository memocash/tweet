package db

import (
	"fmt"
	"strconv"
)

type BotStreamsCount struct {
	Count int
}

func (c *BotStreamsCount) GetPrefix() string {
	return PrefixBotStreamsCount
}

func (c *BotStreamsCount) GetUid() []byte {
	return nil
}

func (c *BotStreamsCount) SetUid([]byte) {
}

func (c *BotStreamsCount) Serialize() []byte {
	return []byte(strconv.FormatUint(uint64(c.Count), 10))
}

func (c *BotStreamsCount) Deserialize(d []byte) {
	c.Count, _ = strconv.Atoi(string(d))
}

func GetBotStreamsCount() (*BotStreamsCount, error) {
	var botStreamsCount = &BotStreamsCount{}
	if err := GetSpecificItem(botStreamsCount); err != nil {
		return nil, fmt.Errorf("error getting bot streams count from db; %w", err)
	}
	return botStreamsCount, nil
}
