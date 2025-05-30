package config

import (
	"github.com/kelseyhightower/envconfig"
	"time"
)

type Config struct {
	Api struct {
		GroupId  string `envconfig:"API_GROUP_ID" default:"default" required:"true"`
		Messages MessagesConfig
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
			SupportChatId               int64  `envconfig:"API_TELEGRAM_SUPPORT_CHAT_ID" required:"true"`
			Token                       string `envconfig:"API_TELEGRAM_TOKEN" required:"true"`
			PublicInterestChannelPrefix string `envconfig:"API_TELEGRAM_PUBLIC_INTEREST_CHANNEL_PREFIX" default:"awk_" required:"true"`
		}
		Reader ReaderConfig
		Queue  QueueConfig
		Writer struct {
			Uri string `envconfig:"API_WRITER_URI" default:"http://pub:8080/v1/batch" required:"true"`
		}
		Interests struct {
			Uri string `envconfig:"API_INTERESTS_URI" default:"http://interests-api:8080/v1" required:"true"`
		}
		Token struct {
			Internal string `envconfig:"API_TOKEN_INTERNAL" required:"true"`
		}
		Usage struct {
			Uri        string `envconfig:"API_USAGE_URI" default:"usage:50051" required:"true"`
			Connection struct {
				Count struct {
					Init uint32 `envconfig:"API_USAGE_CONN_COUNT_INIT" default:"1" required:"true"`
					Max  uint32 `envconfig:"API_USAGE_CONN_COUNT_MAX" default:"10" required:"true"`
				}
				IdleTimeout time.Duration `envconfig:"API_USAGE_CONN_IDLE_TIMEOUT" default:"15m" required:"true"`
			}
			Limits struct {
				Interests       map[int64]int64 `envconfig:"API_USAGE_LIMITS_INTERESTS" required:"true"`
				InterestsPublic map[int64]int64 `envconfig:"API_USAGE_LIMITS_INTERESTS_PUBLIC" required:"true"`
				Subscriptions   map[int64]int64 `envconfig:"API_USAGE_LIMITS_SUBSCRIPTIONS" required:"true"`
			}
		}
	}
	Log struct {
		Level int `envconfig:"LOG_LEVEL" default:"-4" required:"true"`
	}
}

type ReaderConfig struct {
	Uri          string `envconfig:"API_READER_URI" default:"http://reader:8080" required:"true"`
	UriEventBase string `envconfig:"API_READER_URI_EVT_BASE" default:"https://awakari.com/pub-msg.html?id=" required:"true"`
	CallBack     struct {
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

type MessagesConfig struct {
	Type string `envconfig:"API_MESSAGES_TYPE" default:"com_awakari_bot_telegram_v1" required:"true"`
}

func NewConfigFromEnv() (cfg Config, err error) {
	err = envconfig.Process("", &cfg)
	return
}
