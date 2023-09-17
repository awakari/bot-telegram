package config

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestConfig(t *testing.T) {
	os.Setenv("LOG_LEVEL", "4")
	os.Setenv("API_PORT", "56789")
	cfg, err := NewConfigFromEnv()
	assert.Nil(t, err)
	assert.Equal(t, uint16(56789), cfg.Api.Port)
	assert.Equal(t, 4, cfg.Log.Level)
}
