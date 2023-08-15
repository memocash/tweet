package bot

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jchavannes/jgo/jerr"
	"github.com/jchavannes/jgo/jlog"
	"github.com/memocash/index/ref/bitcoin/tx/gen"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/bot/info"
	"github.com/memocash/tweet/bot/strm"
	"github.com/memocash/tweet/config"
	"github.com/memocash/tweet/db"
	"github.com/memocash/tweet/graph"
	"github.com/memocash/tweet/tweets"
	"github.com/memocash/tweet/tweets/obj"
	tweetWallet "github.com/memocash/tweet/wallet"
	twitterscraper "github.com/n0madic/twitter-scraper"
	"github.com/syndtr/goleveldb/leveldb"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
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
	bot := &Bot{
		Mnemonic:     mnemonic,
		Addresses:    addresses,
		Addr:         *addr,
		Key:          key,
		TweetScraper: scraper,
		ErrorChan:    make(chan error),
		Verbose:      verbose,
		Down:         down,
	}
	if err := bot.SetExistingCookies(); err != nil {
		return nil, jerr.Get("error setting existing cookies", err)
	}
	return bot, nil
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

func (b *Bot) Run() error {
	if err := b.ProcessMissedTxs(); err != nil {
		jerr.Get("fatal error updating bot", err).Fatal()
	}
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigc
		log.Println("Tweet Bot caught SIGINT, saving cookies and stopping...")
		b.ErrorChan <- nil
	}()
	go func() {
		infoServer := info.NewServer(b.TweetScraper)
		b.ErrorChan <- fmt.Errorf("error info server listener; %w", infoServer.Listen())
	}()
	jlog.Logf("Bot listening to address: %s\n", b.Addr.String())
	if err := b.SetAddresses(); err != nil {
		return jerr.Get("error setting addresses", err)
	}
	if err := graph.AddressListen(b.Addresses, b.SaveTx, b.ErrorChan); err != nil {
		return jerr.Get("error listening to address on graphql", err)
	}
	updateInterval := config.GetConfig().UpdateInterval
	if updateInterval == 0 {
		updateInterval = 180
	}
	go func() {
		for {
			streams, err := strm.GetStreams(true)
			if err != nil {
				b.ErrorChan <- jerr.Get("error making stream array bot listen", err)
			}
			if err = checkAndUpdateProfiles(streams, b); err != nil {
				b.ErrorChan <- jerr.Get("error updating profiles for bot listen", err)
			}
			if err = b.CheckForNewTweets(streams); err != nil {
				b.ErrorChan <- jerr.Get("error checking for new tweets for bot listen", err)
			}
			time.Sleep(time.Duration(updateInterval) * time.Minute)
		}
	}()
	mainErr := <-b.ErrorChan
	if err := tweets.SaveCookies(b.TweetScraper.GetCookies()); err != nil {
		mainErr = jerr.Combine(jerr.Get("memo bot main error", mainErr), jerr.Get("error saving cookies", err))
	}
	if mainErr != nil {
		return jerr.Get("error running memo bot listen", mainErr)
	}
	return nil
}

func (b *Bot) CheckForNewTweets(streams []strm.Stream) error {
	if b.Verbose {
		log.Println("Checking for new tweets...")
	}
	for _, stream := range streams {
		flag, err := db.GetFlag(stream.Owner, stream.UserID)
		if err != nil {
			return jerr.Get("error getting flag for listen skipped", err)
		}
		err = tweets.GetSkippedTweets(obj.AccountKey{
			UserID:  stream.UserID,
			Key:     stream.Wallet.Key,
			Address: stream.Wallet.Address,
		}, &stream.Wallet, b.TweetScraper, flag.Flags, 100, false)
		if err != nil && !jerr.HasErrorPart(err, gen.NotEnoughValueErrorText) {
			return jerr.Get("error getting skipped tweets on bot listen", err)
		}
		time.Sleep(config.GetScrapeSleepTime())
	}
	if err := b.SafeUpdate(); err != nil {
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
		decryptedKey, err := tweetWallet.DecryptFromDb(addressKey.Key)
		if err != nil {
			return jerr.Get("error decrypting key", err)
		}
		key, err := wallet.ImportPrivateKey(string(decryptedKey))
		if err != nil {
			return jerr.Get("error importing private key", err)
		}
		b.Addresses = append(b.Addresses, key.GetAddress().GetEncoded())
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
	streams, err := strm.GetStreams(false)
	if err != nil {
		return jerr.Get("error making stream array update", err)
	}
	err = b.SetAddresses()
	if err != nil {
		return jerr.Get("error setting addresses", err)
	}
	for _, stream := range streams {
		if b.Verbose {
			jlog.Logf("streaming %s to address %s\n", stream.UserID, stream.Wallet.Key.GetAddr())
		}
	}
	if err := graph.AddressListen(b.Addresses, b.SaveTx, b.ErrorChan); err != nil {
		return jerr.Get("error listening to address on graphql", err)
	}
	return nil
}

func (b *Bot) SetExistingCookies() error {
	dbCookies, err := db.GetCookies()
	if err != nil && !errors.Is(err, leveldb.ErrNotFound) {
		return fmt.Errorf("error getting cookies; %w", err)
	}
	if dbCookies != nil {
		var cookieList []*http.Cookie
		err := json.Unmarshal(dbCookies.CookieData, &cookieList)
		if err != nil {
			return jerr.Get("error unmarshalling cookies", err)
		}
		b.TweetScraper.SetCookies(cookieList)
	}
	return nil
}
