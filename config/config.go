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
		GroupId  string `envconfig:"API_GROUP_ID" default:"com.github.awakari.bot-telegram" required:"true"`
		Messages struct {
			Uri string `envconfig:"API_MESSAGES_URI" default:"messages:50051" required:"true"`
		}
		Source struct {
			Feeds    FeedsConfig
			Telegram TelegramConfig
		}
		Telegram struct {
			Webhook struct {
				Host    string `envconfig:"API_TELEGRAM_WEBHOOK_HOST" default:"demo.awakari.cloud" required:"true"`
				Path    string `envconfig:"API_TELEGRAM_WEBHOOK_PATH" default:"/" required:"true"`
				Port    uint16 `envconfig:"API_TELEGRAM_WEBHOOK_PORT" default:"8080" required:"true"`
				ConnMax uint32 `envconfig:"API_TELEGRAM_WEBHOOK_CONN_MAX" default:"100"`
				Token   string `envconfig:"API_TELEGRAM_WEBHOOK_TOKEN" default:"xxxxxxxxxx"`
			}
			SupportChatId int64  `envconfig:"API_TELEGRAM_SUPPORT_CHAT_ID" required:"true"`
			Token         string `envconfig:"API_TELEGRAM_TOKEN" required:"true"`
		}
		Uri    string `envconfig:"API_URI" default:"api:50051" required:"true"`
		Writer struct {
			Uri string `envconfig:"API_WRITER_URI" default:"resolver:50051" required:"true"`
		}
	}
	Chats   ChatsConfig
	Payment PaymentConfig
	Log     struct {
		Level int `envconfig:"LOG_LEVEL" default:"-4" required:"true"`
	}
	Replica ReplicaConfig
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
	GroupId string `envconfig:"API_SOURCE_FEEDS_GROUP_ID" default:"com.github.awakari.source-feeds"`
	Uri     string `envconfig:"API_SOURCE_FEEDS_URI" default:"source-feeds:50051" required:"true"`
}

type TelegramConfig struct {
	GroupId string `envconfig:"API_SOURCE_TELEGRAM_GROUP_ID" default:"com.github.awakari.source-telegram"`
	Uri     string `envconfig:"API_SOURCE_TELEGRAM_URI" default:"source-telegram:50051" required:"true"`
}

type ReplicaConfig struct {
	Range uint32 `envconfig:"REPLICA_RANGE" required:"true"`
	Name  string `envconfig:"REPLICA_NAME" required:"true"`
}

func NewConfigFromEnv() (cfg Config, err error) {
	err = envconfig.Process("", &cfg)
	return
}
