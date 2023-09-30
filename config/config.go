package config

import (
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Api struct {
		PaymentProviderToken string `envconfig:"API_PAYMENT_PROVIDER_TOKEN" required:"true"`
		Telegram             struct {
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
	Log struct {
		Level int `envconfig:"LOG_LEVEL" default:"-4" required:"true"`
	}
}

func NewConfigFromEnv() (cfg Config, err error) {
	err = envconfig.Process("", &cfg)
	return
}
