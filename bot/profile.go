package bot

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jchavannes/jgo/jerr"
	"github.com/jchavannes/jgo/jlog"
	"github.com/memocash/tweet/config"
	"github.com/memocash/tweet/db"
	"github.com/memocash/tweet/tweets"
	tweetWallet "github.com/memocash/tweet/wallet"
	"github.com/syndtr/goleveldb/leveldb"
	"time"
)

func checkAndUpdateProfiles(botStreams []Stream, b *Bot) error {
	for _, stream := range botStreams {
		if err := checkAndUpdateProfile(b, stream); err != nil {
			return fmt.Errorf("error updating bot profile for stream; %w", err)
		}
		time.Sleep(config.GetScrapeSleepTime())
	}
	return nil
}

func checkAndUpdateProfile(b *Bot, stream Stream) error {
	profile, err := tweets.GetProfile(stream.UserID, b.TweetScraper)
	if err != nil {
		return jerr.Get("fatal error getting profile", err)
	}
	existingDbProfile, err := db.GetProfile(stream.Owner, stream.UserID)
	if err != nil && !errors.Is(err, leveldb.ErrNotFound) {
		return jerr.Get("error getting profile from database", err)
	}
	if existingDbProfile == nil {
		if err = tweetWallet.UpdateName(stream.Wallet, profile.Name); err != nil {
			return jerr.Get("error updating name", err)
		}
		if err = tweetWallet.UpdateProfileText(stream.Wallet, profile.Description); err != nil {
			return jerr.Get("error updating profile text", err)
		}
		if err = tweetWallet.UpdateProfilePic(stream.Wallet, profile.ProfilePic); err != nil {
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
			if err = tweetWallet.UpdateName(stream.Wallet, profile.Name); err != nil {
				return jerr.Get("error updating name", err)
			} else if b.Verbose {
				jlog.Logf("updated profile name for %s: %s to %s\n", profile.ID, dbProfile.Name, profile.Name)
			}
		}
		if dbProfile.Description != profile.Description {
			if err = tweetWallet.UpdateProfileText(stream.Wallet, profile.Description); err != nil {
				return jerr.Get("error updating profile text", err)
			} else if b.Verbose {
				jlog.Logf("updated profile text for %s: %s to %s\n", profile.ID, dbProfile.Description, profile.Description)
			}
		}
		if dbProfile.ProfilePic != profile.ProfilePic {
			if err = tweetWallet.UpdateProfilePic(stream.Wallet, profile.ProfilePic); err != nil {
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
		Owner:   stream.Owner,
		UserID:  stream.UserID,
		Profile: profileBytes,
	}}); err != nil {
		return jerr.Get("error saving profile to database", err)
	}
	if b.Verbose {
		jlog.Logf("checked for profile updates: %s (%s)", profile.Name, stream.Owner)
	}
	return nil
}
