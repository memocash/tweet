package bot

import (
	"fmt"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/config"
)

func GetKey(index uint) (*wallet.PrivateKey, error) {
	botSeed := config.GetBotSeed()
	mnemonic, err := wallet.GetMnemonicFromString(botSeed)
	if err != nil {
		return nil, fmt.Errorf("error getting mnemonic from string; %w", err)
	}
	path := wallet.GetBip44Path(wallet.Bip44CoinTypeBTC, index, false)
	privKey, err := mnemonic.GetPath(path)
	if err != nil {
		return nil, fmt.Errorf("error getting path; %w", err)
	}
	return privKey, nil
}
