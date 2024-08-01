package config

import (
	"github.com/kelseyhightower/envconfig"
	"time"
)

type Config struct {
	Api struct {
		Admin struct {
			Uri string `envconfig:"API_ADMIN_URI" default:"api:56789" required:"true"`
		}
		GroupId  string `envconfig:"API_GROUP_ID" default:"default" required:"true"`
		Messages struct {
			Uri string `envconfig:"API_MESSAGES_URI" default:"messages:50051" required:"true"`
		}
		Telegram struct {
			Bot struct {
				Port uint16 `envconfig:"API_TELEGRAM_BOT_PORT" default:"50051" required:"true"`
			}
			Webhook struct {
				Host    string `envconfig:"API_TELEGRAM_WEBHOOK_HOST" default:"tgbot.awakari.com" required:"true"`
				Path    string `envconfig:"API_TELEGRAM_WEBHOOK_PATH" default:"/" required:"true"`
				Port    uint16 `envconfig:"API_TELEGRAM_WEBHOOK_PORT" default:"8080" required:"true"`
				ConnMax uint32 `envconfig:"API_TELEGRAM_WEBHOOK_CONN_MAX" default:"100"`
				Token   string `envconfig:"API_TELEGRAM_WEBHOOK_TOKEN" default:"xxxxxxxxxx"`
			}
			SupportChatId int64  `envconfig:"API_TELEGRAM_SUPPORT_CHAT_ID" required:"true"`
			Token         string `envconfig:"API_TELEGRAM_TOKEN" required:"true"`
		}
		Uri    string `envconfig:"API_URI" default:"api:50051" required:"true"`
		Reader ReaderConfig
		Queue  QueueConfig
	}
	Payment PaymentConfig
	Log     struct {
		Level int `envconfig:"LOG_LEVEL" default:"-4" required:"true"`
	}
}

type PaymentConfig struct {
	Backoff  BackoffConfig
	Currency struct {
		Code      string  `envconfig:"PAYMENT_CURRENCY_CODE" required:"true" default:"EUR"`
		SubFactor float64 `envconfig:"PAYMENT_CURRENCY_SUB_FACTOR" required:"true" default:"100"`
	}
	PreCheckout struct {
		Timeout time.Duration `envconfig:"PAYMENT_PRE_CHECKOUT_TIMEOUT" required:"true" default:"10s"`
	}
	Price    PriceConfig
	Provider struct {
		Token string `envconfig:"PAYMENT_PROVIDER_TOKEN" required:"true"`
	}
}

type BackoffConfig struct {
	Init       time.Duration `envconfig:"PAYMENT_BACKOFF_INIT" default:"100ms"`
	Factor     float64       `envconfig:"PAYMENT_BACKOFF_FACTOR" default:"2"`
	LimitTotal time.Duration `envconfig:"PAYMENT_BACKOFF_LIMIT_TOTAL" default:"15m"`
}

type PriceConfig struct {
	MessagePublishing struct {
		DailyLimit float64 `envconfig:"PAYMENT_PRICE_MESSAGE_PUBLISHING_DAILY_LIMIT" required:"true" default:"0.04"`
		Extra      float64 `envconfig:"PAYMENT_PRICE_MESSAGE_PUBLISHING_EXTRA" required:"true" default:"1"`
	}
	Subscription struct {
		CountLimit float64 `envconfig:"PAYMENT_PRICE_SUBSCRIPTION_COUNT_LIMIT" required:"true" default:"0.1"`
		Extension  float64 `envconfig:"PAYMENT_PRICE_SUBSCRIPTION_EXTENSION" required:"true" default:"0.1"`
	}
}

type FeedsConfig struct {
	Uri string `envconfig:"API_SOURCE_FEEDS_URI" default:"source-feeds:50051" required:"true"`
}

type SitesConfig struct {
	Uri string `envconfig:"API_SOURCE_SITES_URI" default:"source-sites:50051" required:"true"`
}

type ReaderConfig struct {
	Uri      string `envconfig:"API_READER_URI" default:"http://reader:8080/v1" required:"true"`
	CallBack struct {
		Protocol string `envconfig:"API_READER_CALLBACK_PROTOCOL" default:"http" required:"true"`
		Host     string `envconfig:"API_READER_CALLBACK_HOST" default:"bot-telegram" required:"true"`
		Port     uint16 `envconfig:"API_READER_CALLBACK_PORT" default:"8081" required:"true"`
		Path     string `envconfig:"API_READER_CALLBACK_PATH" default:"/v1/chat" required:"true"`
	}
}

type QueueConfig struct {
	BackoffError     time.Duration `envconfig:"API_QUEUE_BACKOFF_ERROR" default:"1s" required:"true"`
	Uri              string        `envconfig:"API_QUEUE_URI" default:"queue:50051" required:"true"`
	InterestsCreated struct {
		BatchSize uint32 `envconfig:"API_QUEUE_INTERESTS_CREATED_BATCH_SIZE" default:"10" required:"true"`
		Name      string `envconfig:"API_QUEUE_INTERESTS_CREATED_NAME" default:"bot-telegram" required:"true"`
		Subj      string `envconfig:"API_QUEUE_INTERESTS_CREATED_SUBJ" default:"interests-created" required:"true"`
	}
}

func NewConfigFromEnv() (cfg Config, err error) {
	err = envconfig.Process("", &cfg)
	return
}
