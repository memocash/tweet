package bot

import (
	"errors"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/jchavannes/jgo/jerr"
	"github.com/jchavannes/jgo/jlog"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/config"
	"github.com/memocash/tweet/db"
	"github.com/memocash/tweet/graph"
	"github.com/memocash/tweet/tweets"
	"github.com/memocash/tweet/tweets/obj"
	"github.com/syndtr/goleveldb/leveldb"
	"sync"
	"time"
)

type Bot struct {
	Mnemonic    *wallet.Mnemonic
	Addresses   []string
	Addr        wallet.Addr
	Key         wallet.PrivateKey
	TweetClient *twitter.Client
	Stream      *tweets.Stream
	ErrorChan   chan error
	TxMutex     sync.Mutex
	UpdateMutex sync.Mutex
	Crypt       []byte
	Timer       *time.Timer
	Verbose     bool
}

func NewBot(mnemonic *wallet.Mnemonic, addresses []string, key wallet.PrivateKey, tweetClient *twitter.Client, verbose bool) (*Bot, error) {
	if len(addresses) == 0 {
		return nil, jerr.New("error new bot, no addresses")
	}
	addr, err := wallet.GetAddrFromString(addresses[0])
	if err != nil {
		return nil, jerr.Get("error getting address from string for new bot", err)
	}
	stream, err := tweets.NewStream()
	if err != nil {
		return nil, jerr.Get("error getting new tweet stream", err)
	}
	return &Bot{
		Mnemonic:    mnemonic,
		Addresses:   addresses,
		Addr:        *addr,
		Key:         key,
		Stream:      stream,
		TweetClient: tweetClient,
		ErrorChan:   make(chan error),
		Verbose:     verbose,
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
	txs, err := graph.GetAddressUpdates(b.Addresses[0], start)
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

func (b *Bot) Listen() error {
	jlog.Logf("Bot listening to address: %s\n", b.Addresses[0])
	if err := graph.AddressListen(b.Addresses, b.SaveTx, b.ErrorChan); err != nil {
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
		}
	}()
	botStreams, err := getBotStreams(b.Crypt)
	if err != nil {
		return jerr.Get("error getting bot streams for listen skipped", err)
	}
	for _, stream := range botStreams {
		if err = tweets.GetSkippedTweets(obj.AccountKey{
			Account: stream.Name,
			Key:     stream.Wallet.Key,
			Address: stream.Wallet.Address,
		}, &stream.Wallet, b.TweetClient, db.GetDefaultFlags(), 100, false); err != nil {
			return jerr.Get("error getting skipped tweets on bot listen", err)
		}
	}
	if err = b.SafeUpdate(); err != nil {
		return jerr.Get("error updating stream 2nd time", err)
	}
	return jerr.Get("error in listen", <-b.ErrorChan)
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

func (b *Bot) SafeUpdate() error {
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
	//create an array of {twitterName, newKey} objects by searching through the linked-<senderAddress>-<twitterName> fields
	botStreams, err := getBotStreams(b.Crypt)
	if err != nil {
		return jerr.Get("error making stream array update", err)
	}
	for _, stream := range botStreams {
		streamKey, err := wallet.ImportPrivateKey(stream.Key)
		if err != nil {
			return jerr.Get("error importing private key", err)
		}
		streamAddress := streamKey.GetAddress()
		if b.Verbose {
			jlog.Logf("streaming %s to address %s\n", stream.Name, streamAddress.GetEncoded())
		}
	}
	if err := db.Save([]db.ObjectI{&db.BotRunningCount{Count: len(botStreams)}}); err != nil {
		return jerr.Get("error saving bot running count", err)
	}
	go func() {
		if len(botStreams) == 0 {
			return
		}
		if err := b.Stream.ListenForNewTweets(botStreams); err != nil {
			b.ErrorChan <- jerr.Get("error twitter initiate stream in update", err)
		}
	}()
	return nil
}
