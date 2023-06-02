package config

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/spf13/viper"
	"time"
)

type Config struct {
	BotSeed        string       `mapstructure:"BOT_SEED"`
	TwitterCreds   TwitterCreds `mapstructure:"TWITTER_CREDS"`
	UpdateInterval int          `mapstructure:"UPDATE_INTERVAL"`
	InfoServerPort int          `mapstructure:"INFO_SERVER_PORT"`
	TemplateDir    string       `mapstructure:"TEMPLATE_DIR"`
	AWS            AwsConfig    `mapstructure:"AWS"`

	DbEncryptionKey string `mapstructure:"DB_ENCRYPTION_KEY"`
}

type TwitterCreds struct {
	UserName string `mapstructure:"USER_NAME"`
	Password string `mapstructure:"PASSWORD"`
	Email    string `mapstructure:"EMAIL"`
}

func (t TwitterCreds) GetStrings() []string {
	if t.UserName == "" || t.Password == "" {
		return nil
	}
	credentials := []string{t.UserName, t.Password}
	if t.Email != "" {
		credentials = append(credentials, t.Email)
	}
	return credentials
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

func GetTwitterCreds() TwitterCreds {
	return _config.TwitterCreds
}

func GetAwsSesCredentials() AwsConfig {
	return _config.AWS
}

// GetScrapeSleepTime spaces out twitter scrapes to avoid rate limiting
func GetScrapeSleepTime() time.Duration {
	return 1 * time.Second
}

func GetDbEncryptionKey() string {
	return _config.DbEncryptionKey
}
