package bot

import (
	"encoding/json"
	"errors"
	"github.com/jchavannes/jgo/jerr"
	"github.com/jchavannes/jgo/jlog"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/tweet/config"
	"github.com/memocash/tweet/db"
	"github.com/memocash/tweet/tweets"
	tweetWallet "github.com/memocash/tweet/wallet"
	"github.com/syndtr/goleveldb/leveldb"
	"log"
	"time"
)

func updateProfiles(botStreams []config.Stream, b *Bot) error {
	for _, stream := range botStreams {
		streamKey, err := wallet.ImportPrivateKey(stream.Key)
		if err != nil {
			return jerr.Get("error importing private key", err)
		}
		streamAddress := streamKey.GetAddress()
		newWallet := tweetWallet.NewWallet(streamAddress, streamKey)
		err = updateProfile(b, newWallet, stream.UserID, stream.Sender)
		time.Sleep(1 * time.Second)
	}
	return nil
}

func updateProfile(b *Bot, newWallet tweetWallet.Wallet, userId int64, senderAddress string) error {
	profile, err := tweets.GetProfile(userId, b.TweetScraper)
	if err != nil {
		return jerr.Get("fatal error getting profile", err)
	}
	log.Println("Name", profile.Name)
	log.Println("Description", profile.Description)
	log.Println("ProfilePic", profile.ProfilePic)
	existingDbProfile, err := db.GetProfile(wallet.GetAddressFromString(senderAddress).GetAddr(), userId)
	if err != nil && !errors.Is(err, leveldb.ErrNotFound) {
		return jerr.Get("error getting profile from database", err)
	}
	if existingDbProfile == nil {
		if err = tweetWallet.UpdateName(newWallet, profile.Name); err != nil {
			return jerr.Get("error updating name", err)
		}
		if err = tweetWallet.UpdateProfileText(newWallet, profile.Description); err != nil {
			return jerr.Get("error updating profile text", err)
		}
		if err = tweetWallet.UpdateProfilePic(newWallet, profile.ProfilePic); err != nil {
			return jerr.Get("error updating profile pic", err)
		}
		if b.Verbose {
			jlog.Log("updated profile info for the first time")
		}
	} else {
		dbProfile := new(tweets.Profile)
		if err = json.Unmarshal(existingDbProfile.Profile, &dbProfile); err != nil {
			return jerr.Getf(err, "error unmarshalling profile: %s", dbProfile.Name)
		}
		if dbProfile.Name != profile.Name {
			if err = tweetWallet.UpdateName(newWallet, profile.Name); err != nil {
				return jerr.Get("error updating name", err)
			} else if b.Verbose {
				jlog.Logf("updated profile name for %s: %s to %s\n", profile.ID, dbProfile.Name, profile.Name)
			}
		}
		if dbProfile.Description != profile.Description {
			if err = tweetWallet.UpdateProfileText(newWallet, profile.Description); err != nil {
				return jerr.Get("error updating profile text", err)
			} else if b.Verbose {
				jlog.Logf("updated profile text for %s: %s to %s\n", profile.ID, dbProfile.Description, profile.Description)
			}
		}
		if dbProfile.ProfilePic != profile.ProfilePic {
			if err = tweetWallet.UpdateProfilePic(newWallet, profile.ProfilePic); err != nil {
				return jerr.Get("error updating profile pic", err)
			} else if b.Verbose {
				jlog.Logf("updated profile pic for %s: %s to %s\n", profile.ID, dbProfile.ProfilePic, profile.ProfilePic)
			}
		}
	}
	profileBytes, err := json.Marshal(profile)
	if err != nil {
		return jerr.Get("error marshalling profile", err)
	}
	if err := db.Save([]db.ObjectI{&db.Profile{
		Address: wallet.GetAddressFromString(senderAddress).GetAddr(),
		UserID:  userId,
		Profile: profileBytes,
	}}); err != nil {
		return jerr.Get("error saving profile to database", err)
	}
	if b.Verbose {
		jlog.Logf("checked for profile updates: %s (%s)", profile.Name, senderAddress)
	}
	return nil
}
