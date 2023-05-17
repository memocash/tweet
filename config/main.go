package config

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/tweet/wallet"
	"github.com/spf13/viper"
)

type Config struct {
	BotSeed        string     `mapstructure:"BOT_SEED"`
	BotCrypt       string     `mapstructure:"BOT_CRYPT"`
	Streams        []Stream   `mapstructure:"STREAMS"`
	TwitterAPI     TwitterAPI `mapstructure:"TWITTER_API"`
	UpdateInterval int        `mapstructure:"UPDATE_INTERVAL"`
	InfoServerPort int        `mapstructure:"INFO_SERVER_PORT"`
	TemplateDir    string     `mapstructure:"TEMPLATE_DIR"`
	AWS            AwsConfig  `mapstructure:"AWS"`
}

type TwitterAPI struct {
	UserName string `mapstructure:"USER_NAME"`
	Password string `mapstructure:"PASSWORD"`
	Email    string `mapstructure:"EMAIL"`
}

type Stream struct {
	Key    string        `mapstructure:"KEY"`
	UserID int64         `mapstructure:"USER_ID"`
	Sender string        `mapstructure:"SENDER"`
	Wallet wallet.Wallet `mapstructure:"WALLET"`
}
type AwsCredentials struct {
	Key    string
	Secret string
	Region string
}

type AwsConfig struct {
	Key       string   `mapstructure:"SES_KEY"`
	Secret    string   `mapstructure:"SES_SECRET"`
	Region    string   `mapstructure:"SES_REGION"`
	FromEmail string   `mapstructure:"SES_FROM_EMAIL"`
	ToEmails  []string `mapstructure:"SES_TO_EMAILS"`
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
	if _config.TemplateDir == "" {
		_config.TemplateDir = "bot/info/templates"
	}
	return nil
}

func GetConfig() Config {
	return _config
}

func GetTwitterAPIConfig() TwitterAPI {
	return _config.TwitterAPI
}

func GetAwsSesCredentials() AwsCredentials {
	return AwsCredentials{
		Key:    _config.AWS.Key,
		Secret: _config.AWS.Secret,
		Region: _config.AWS.Region,
	}
}
