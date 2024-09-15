package config

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestConfig(t *testing.T) {
	os.Setenv("API_TELEGRAM_SUPPORT_CHAT_ID", "12345")
	os.Setenv("API_TELEGRAM_BOT_PORT", "45678")
	os.Setenv("LOG_LEVEL", "4")
	os.Setenv("API_TELEGRAM_WEBHOOK_PORT", "56789")
	os.Setenv("API_TELEGRAM_TOKEN", "yohoho")
	os.Setenv("PAYMENT_PROVIDER_TOKEN", "yohoho")
	os.Setenv("REPLICA_RANGE", "2")
	os.Setenv("REPLICA_NAME", "replica-0")
	os.Setenv("LOGIN_CODE_FROM_USER_IDS", "123:true,456:true")
	cfg, err := NewConfigFromEnv()
	assert.Nil(t, err)
	assert.Equal(t, uint16(56789), cfg.Api.Telegram.Webhook.Port)
	assert.Equal(t, uint16(45678), cfg.Api.Telegram.Bot.Port)
	assert.Equal(t, 4, cfg.Log.Level)
	assert.Equal(t, "yohoho", cfg.Payment.Provider.Token)
	assert.Equal(t, map[int64]bool{
		123: true,
		456: true,
	}, cfg.LoginCode.FromUserIds)
}
