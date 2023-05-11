package db

type SubBotCommand struct {
	HistoryNum int
	BotExists  bool
	TxHash     [32]byte
}

func (t *SubBotCommand) GetPrefix() string {
	return PrefixSubBotCommand
}
func (t *SubBotCommand) GetUid() []byte {
	return t.TxHash[:]
}
func (t *SubBotCommand) SetUid(b []byte) {
	copy(t.TxHash[:], b[:32])
}

func (t *SubBotCommand) Serialize() []byte {
	return nil
}

func (t *SubBotCommand) Deserialize([]byte) {

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
