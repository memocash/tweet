package config

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/database"
	"github.com/spf13/viper"
)

type Config struct {
	BotSeed        string     `mapstructure:"BOT_SEED"`
	BotCrypt       string     `mapstructure:"BOT_CRYPT"`
	Streams        []Stream   `mapstructure:"STREAMS"`
	TwitterAPI     TwitterAPI `mapstructure:"TWITTER_API"`
	UpdateInterval int        `mapstructure:"UPDATE_INTERVAL"`
	InfoServerPort int        `mapstructure:"INFO_SERVER_PORT"`
}

type TwitterAPI struct {
	ConsumerKey    string `mapstructure:"CONSUMER_KEY"`
	ConsumerSecret string `mapstructure:"CONSUMER_SECRET"`
}

func (t TwitterAPI) IsSet() bool {
	return t.ConsumerKey != "" && t.ConsumerSecret != ""
}

type Stream struct {
	Key    string          `mapstructure:"KEY"`
	Name   string          `mapstructure:"NAME"`
	Sender string          `mapstructure:"SENDER"`
	Wallet database.Wallet `mapstructure:"WALLET"`
}

var _config Config

func InitConfig() error {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		return jerr.Get("error reading config", err)
	}
	if err := viper.Unmarshal(&_config); err != nil {
		return jerr.Get("error unmarshalling config", err)
	}
	return nil
}

func GetConfig() Config {
	return _config
}

func GetTwitterAPIConfig() TwitterAPI {
	return _config.TwitterAPI
}
