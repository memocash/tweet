package db

import "github.com/jchavannes/jgo/jutil"

type SubBotCommand struct {
	HistoryNum int
	BotExists  bool
	TxHash     [32]byte
}

func (t *SubBotCommand) GetPrefix() string {
	return PrefixSubBotCommand
}
func (t *SubBotCommand) GetUid() []byte {
	return jutil.ByteReverse(t.TxHash[:])
}
func (t *SubBotCommand) SetUid(b []byte) {
	if len(b) != 32 {
		return
	}
	copy(t.TxHash[:], jutil.ByteReverse(b))
}

func (t *SubBotCommand) Serialize() []byte {
	var botExists byte
	if t.BotExists {
		botExists = 1
	}
	return jutil.CombineBytes(
		jutil.GetIntData(t.HistoryNum),
		[]byte{botExists})
}

func (t *SubBotCommand) Deserialize(data []byte) {
	t.HistoryNum = jutil.GetInt(data[:4])
	t.BotExists = data[4] == 1
}

func GetSubBotCommand(txHash [32]byte) (*SubBotCommand, error) {
	var subBotCommand = &SubBotCommand{
		TxHash: txHash,
	}
	if err := GetSpecificItem(subBotCommand); err != nil {
		return nil, err
	}
	return subBotCommand, nil
}
