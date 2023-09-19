package config

import (
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Api struct {
		Host    string `envconfig:"API_HOST" default:"demo.awakari.cloud" required:"true"`
		Path    string `envconfig:"API_PATH" default:"/" required:"true"`
		Port    uint16 `envconfig:"API_PORT" default:"8080" required:"true"`
		Token   string `envconfig:"API_TOKEN" required:"true"`
		GroupId string `envconfig:"API_GROUP_ID" default:"com.github.awakari.bot-telegram" required:"true"`
	}
	Log struct {
		Level int `envconfig:"LOG_LEVEL" default:"-4" required:"true"`
	}
}

func NewConfigFromEnv() (cfg Config, err error) {
	err = envconfig.Process("", &cfg)
	return
}
