package bot

import (
	"errors"
	"github.com/jchavannes/jgo/jerr"
	"github.com/jchavannes/jgo/jlog"
	"github.com/memocash/index/ref/bitcoin/tx/gen"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/config"
	"github.com/memocash/tweet/db"
	"github.com/memocash/tweet/graph"
	"github.com/memocash/tweet/tweets"
	"github.com/memocash/tweet/tweets/obj"
	twitterscraper "github.com/n0madic/twitter-scraper"
	"github.com/syndtr/goleveldb/leveldb"
	"log"
	"sync"
	"time"
)

type Bot struct {
	Mnemonic     *wallet.Mnemonic
	Addresses    []string
	Addr         wallet.Addr
	Key          wallet.PrivateKey
	TweetScraper *twitterscraper.Scraper
	ErrorChan    chan error
	TxMutex      sync.Mutex
	UpdateMutex  sync.Mutex
	Crypt        []byte
	Timer        *time.Timer
	Verbose      bool
	Down         bool
}

func NewBot(mnemonic *wallet.Mnemonic, scraper *twitterscraper.Scraper, addresses []string, key wallet.PrivateKey, verbose bool, down bool) (*Bot, error) {
	if len(addresses) == 0 {
		return nil, jerr.New("error new bot, no addresses")
	}
	addr, err := wallet.GetAddrFromString(addresses[0])
	if err != nil {
		return nil, jerr.Get("error getting address from string for new bot", err)
	}
	if err != nil {
		return nil, jerr.Get("error getting new tweet stream", err)
	}
	return &Bot{
		Mnemonic:     mnemonic,
		Addresses:    addresses,
		Addr:         *addr,
		Key:          key,
		TweetScraper: scraper,
		ErrorChan:    make(chan error),
		Verbose:      verbose,
		Down:         down,
	}, nil
}

func (b *Bot) ProcessMissedTxs() error {
	recentAddressSeenTx, err := db.GetRecentAddressSeenTx(b.Addr)
	if err != nil && !errors.Is(err, leveldb.ErrNotFound) {
		return jerr.Get("error getting recent address seen tx for addr", err)
	}
	var start time.Time
	if recentAddressSeenTx != nil {
		start = recentAddressSeenTx.Seen
	}
	if b.Verbose {
		jlog.Logf("Processing missed txs using start: %s\n", start.Format(time.RFC3339))
	}
	txs, err := graph.GetAddressUpdates(b.Addr.String(), start)
	if err != nil {
		return jerr.Get("error getting address update txs", err)
	} else if b.Verbose {
		jlog.Logf("Found %d missed txs\n", len(txs))
	}
	for _, tx := range txs {
		if err := b.SaveTx(tx); err != nil {
			return jerr.Get("error saving missed process tx", err)
		} else if b.Verbose {
			jlog.Logf("Found missed process tx: %s, seen: %s\n", tx.Hash, tx.Seen.GetTime().Format(time.RFC3339))
		}
	}
	return nil
}
func (b *Bot) MaintenanceListen() error {
	jlog.Logf("Bot listening to address: %s\n", b.Addr.String())
	if err := graph.AddressListen([]string{b.Addr.String()}, b.SaveTx, b.ErrorChan); err != nil {
		return jerr.Get("error listening to address on graphql", err)
	}
	return jerr.Get("error in listen", <-b.ErrorChan)
}
func (b *Bot) Listen() error {
	jlog.Logf("Bot listening to address: %s\n", b.Addr.String())
	err := b.SetAddresses()
	if err != nil {
		return jerr.Get("error setting addresses", err)
	}
	if err = graph.AddressListen(b.Addresses, b.SaveTx, b.ErrorChan); err != nil {
		return jerr.Get("error listening to address on graphql", err)
	}
	updateInterval := config.GetConfig().UpdateInterval
	if updateInterval == 0 {
		updateInterval = 180
	}
	go func() {
		for {
			t := time.NewTimer(time.Duration(updateInterval) * time.Minute)
			select {
			case <-t.C:
			}
			botStreams, err := getBotStreams(b.Crypt)
			if err != nil {
				b.ErrorChan <- jerr.Get("error making stream array bot listen", err)
			}
			if err = updateProfiles(botStreams, b); err != nil {
				b.ErrorChan <- jerr.Get("error updating profiles for bot listen", err)
			}
			if err = b.CheckForNewTweets(); err != nil {
				b.ErrorChan <- jerr.Get("error checking for new tweets for bot listen", err)
			}
		}
	}()
	botStreams, err := getBotStreams(b.Crypt)
	if err != nil {
		return jerr.Get("error getting bot streams for listen skipped", err)
	}
	for _, stream := range botStreams {
		flag, err := db.GetFlag(wallet.GetAddressFromString(stream.Sender).GetAddr(), stream.UserID)
		if err != nil {
			return jerr.Get("error getting flag for listen skipped", err)
		}
		if flag.Flags.CatchUp {
			err = tweets.GetSkippedTweets(obj.AccountKey{
				UserID:  stream.UserID,
				Key:     stream.Wallet.Key,
				Address: stream.Wallet.Address,
			}, &stream.Wallet, b.TweetScraper, flag.Flags, 100, false)
			if err != nil && !jerr.HasErrorPart(err, gen.NotEnoughValueErrorText) {
				return jerr.Get("error getting skipped tweets on bot listen", err)
			}
		}
	}
	if err = b.CheckForNewTweets(); err != nil {
		return jerr.Get("error updating stream 2nd time", err)
	}
	return jerr.Get("error in listen", <-b.ErrorChan)
}

func (b *Bot) CheckForNewTweets() error {
	log.Println("Checking for new tweets")
	botStreams, err := getBotStreams(b.Crypt)
	if err != nil {
		return jerr.Get("error getting bot streams for listen skipped", err)
	}
	for _, stream := range botStreams {
		flag, err := db.GetFlag(wallet.GetAddressFromString(stream.Sender).GetAddr(), stream.UserID)
		if err != nil {
			return jerr.Get("error getting flag for listen skipped", err)
		}
		if flag.Flags.CatchUp {
			err = tweets.GetSkippedTweets(obj.AccountKey{
				UserID:  stream.UserID,
				Key:     stream.Wallet.Key,
				Address: stream.Wallet.Address,
			}, &stream.Wallet, b.TweetScraper, flag.Flags, 100, false)
			if err != nil && !jerr.HasErrorPart(err, gen.NotEnoughValueErrorText) {
				return jerr.Get("error getting skipped tweets on bot listen", err)
			}
		}
	}
	err = b.SafeUpdate()
	if err != nil {
		return jerr.Get("error updating streams after getting new tweets", err)
	}
	return nil
}

func (b *Bot) SaveTx(tx graph.Tx) error {
	b.TxMutex.Lock()
	defer b.TxMutex.Unlock()
	saveTx := NewSaveTx(b)
	if err := saveTx.Save(tx); err != nil {
		return jerr.Get("error saving bot tx", err)
	}
	return nil
}

func (b *Bot) SetAddresses() error {
	b.Addresses = []string{b.Addr.String()}
	addressKeys, err := db.GetAllAddressKey()
	if err != nil {
		return jerr.Get("error getting all address keys", err)
	}
	for _, addressKey := range addressKeys {
		b.Addresses = append(b.Addresses, wallet.Addr(addressKey.Address).String())
	}
	return nil
}

func (b *Bot) SafeUpdate() error {
	if b.Down {
		return nil
	}
	b.UpdateMutex.Lock()
	defer b.UpdateMutex.Unlock()
	var waitCount = 1
	err := b.UpdateStream()
	for err != nil && waitCount < 30 {
		if !jerr.HasErrorPart(err, "DuplicateRule") {
			return jerr.Get("error updating stream", err)
		}
		jlog.Logf("Error updating stream: %s\n", err.Error())
		err = b.UpdateStream()
		time.Sleep(time.Duration(waitCount) * time.Second)
		waitCount *= 2
	}
	if err != nil {
		return jerr.Get("error updating stream", err)
	}
	return nil
}

func (b *Bot) UpdateStream() error {
	//create an array of {userId, newKey} objects by searching through the linked-<senderAddress>-<userId> fields
	botStreams, err := getBotStreams(b.Crypt)
	if err != nil {
		return jerr.Get("error making stream array update", err)
	}
	err = b.SetAddresses()
	if err != nil {
		return jerr.Get("error setting addresses", err)
	}
	for _, stream := range botStreams {
		streamKey, err := wallet.ImportPrivateKey(stream.Key)
		if err != nil {
			return jerr.Get("error importing private key", err)
		}
		streamAddress := streamKey.GetAddress()
		if b.Verbose {
			jlog.Logf("streaming %s to address %s\n", stream.UserID, streamAddress.GetEncoded())
		}
	}
	if err := db.Save([]db.ObjectI{&db.BotRunningCount{Count: len(botStreams)}}); err != nil {
		return jerr.Get("error saving bot running count", err)
	}
	if err := graph.AddressListen(b.Addresses, b.SaveTx, b.ErrorChan); err != nil {
		return jerr.Get("error listening to address on graphql", err)
	}
	return nil
}
