package config

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestConfig(t *testing.T) {
	os.Setenv("LOG_LEVEL", "4")
	os.Setenv("API_TELEGRAM_WEBHOOK_PORT", "56789")
	os.Setenv("API_TELEGRAM_TOKEN", "yohoho")
	os.Setenv("PAYMENT_PROVIDER_TOKEN", "yohoho")
	cfg, err := NewConfigFromEnv()
	assert.Nil(t, err)
	assert.Equal(t, uint16(56789), cfg.Api.Telegram.Webhook.Port)
	assert.Equal(t, 4, cfg.Log.Level)
	assert.Equal(t, "yohoho", cfg.Payment.Provider.Token)
}
