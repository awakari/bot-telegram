package config

import (
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Api struct {
		Telegram struct {
			Webhook struct {
				Host string `envconfig:"API_TELEGRAM_WEBHOOK_HOST" default:"demo.awakari.cloud" required:"true"`
				Path string `envconfig:"API_TELEGRAM_WEBHOOK_PATH" default:"/" required:"true"`
				Port uint16 `envconfig:"API_TELEGRAM_WEBHOOK_PORT" default:"8080" required:"true"`
			}
			Token string `envconfig:"API_TELEGRAM_TOKEN" required:"true"`
		}
		GroupId string `envconfig:"API_GROUP_ID" default:"com.github.awakari.bot-telegram" required:"true"`
		Uri     string `envconfig:"API_URI" default:"api:50051" required:"true"`
	}
	Chats ChatsConfig
	Log   struct {
		Level int `envconfig:"LOG_LEVEL" default:"-4" required:"true"`
	}
}

type ChatsConfig struct {
	Db ChatsDbConfig
}

type ChatsDbConfig struct {
	Uri      string `envconfig:"CHATS_DB_URI" default:"mongodb://localhost:27017/?retryWrites=true&w=majority" required:"true"`
	Name     string `envconfig:"CHATS_DB_NAME" default:"bot-telegram" required:"true"`
	UserName string `envconfig:"CHATS_DB_USERNAME" default:""`
	Password string `envconfig:"CHATS_DB_PASSWORD" default:""`
	Table    struct {
		Name string `envconfig:"CHATS_DB_TABLE_NAME" default:"chats" required:"true"`
	}
	Tls struct {
		Enabled  bool `envconfig:"CHATS_DB_TLS_ENABLED" default:"false" required:"true"`
		Insecure bool `envconfig:"CHATS_DB_TLS_INSECURE" default:"false" required:"true"`
	}
}

func NewConfigFromEnv() (cfg Config, err error) {
	err = envconfig.Process("", &cfg)
	return
}
