package config

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/spf13/viper"
)

type Config struct {
	BotKey  string   `mapstructure:"BOT_KEY"`
	Streams []Stream `mapstructure:"STREAMS"`
}

type Stream struct {
	Key  string `mapstructure:"KEY"`
	Name string `mapstructure:"NAME"`
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
