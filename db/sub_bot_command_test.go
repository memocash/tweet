package db_test

import (
	"github.com/jchavannes/btcd/chaincfg/chainhash"
	"github.com/memocash/tweet/db"
	"log"
	"testing"
)

func TestSubBotCommand(t *testing.T) {
	historyNum := 10
	botExists := true
	txHash, err := chainhash.NewHashFromStr("951ffc67b4f77cae554a680113be201ede06e2fe2d7f3632d04c6aa71cc7edfa")
	if err != nil {
		t.Error(err)
		return
	}
	subBotCommand := &db.SubBotCommand{
		HistoryNum: historyNum,
		BotExists:  botExists,
		TxHash:     *txHash,
	}
	uid := subBotCommand.GetUid()
	log.Printf("uid: %x\n", uid)
	var newSubBotCommand = new(db.SubBotCommand)
	newSubBotCommand.SetUid(uid)
	newSubBotCommand.Deserialize(subBotCommand.Serialize())
	if newSubBotCommand.HistoryNum != historyNum {
		t.Errorf("HistoryNum mismatch, got: %d, expected: %d", newSubBotCommand.HistoryNum, historyNum)
		return
	}
	if newSubBotCommand.BotExists != botExists {
		t.Errorf("botExists mismatch, got: %t, expected: %t",
			newSubBotCommand.BotExists, botExists)
		return
	}
}
