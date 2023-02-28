package db

import (
	"fmt"
	"strconv"
)

type BotRunningCount struct {
	Count int
}

func (c *BotRunningCount) GetPrefix() string {
	return PrefixBotRunningCount
}

func (c *BotRunningCount) GetUid() []byte {
	return nil
}

func (c *BotRunningCount) SetUid([]byte) {
}

func (c *BotRunningCount) Serialize() []byte {
	return []byte(strconv.FormatUint(uint64(c.Count), 10))
}

func (c *BotRunningCount) Deserialize(d []byte) {
	c.Count, _ = strconv.Atoi(string(d))
}

func GetBotRunningCount() (*BotRunningCount, error) {
	var botRunningCount = &BotRunningCount{}
	if err := GetSpecificItem(botRunningCount); err != nil {
		return nil, fmt.Errorf("error getting bot running count from db; %w", err)
	}
	return botRunningCount, nil
}
