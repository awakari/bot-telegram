package config

import (
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Api struct {
		Host  string `envconfig:"API_HOST" default:"demo.awakari.cloud" required:"true"`
		Path  string `envconfig:"API_PATH" default:"/telegram/v1" required:"true"`
		Port  uint16 `envconfig:"API_PORT" default:"8080" required:"true"`
		Token string `envconfig:"API_TOKEN" required:"true"`
	}
	Log struct {
		Level int `envconfig:"LOG_LEVEL" default:"-4" required:"true"`
	}
}

func NewConfigFromEnv() (cfg Config, err error) {
	err = envconfig.Process("", &cfg)
	return
}
