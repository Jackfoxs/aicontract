package core

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"backend/config"
	"backend/global"
)

func setupTestLogger() *slog.Logger {
	handler := slog.NewTextHandler(io.Discard, nil)
	return slog.New(handler)
}

func TestInitRedisReturnsNilWhenConfigMissing(t *testing.T) {
	originalConfig := global.Config
	originalLog := global.Log
	t.Cleanup(func() {
		global.Config = originalConfig
		global.Log = originalLog
	})

	global.Config = &config.Config{}
	global.Log = setupTestLogger()

	if client := InitRedis(); client != nil {
		t.Fatalf("expected nil client when redis config missing, got %#v", client)
	}
}

func TestInitRedisReturnsNilWhenAddrEmpty(t *testing.T) {
	originalConfig := global.Config
	originalLog := global.Log
	t.Cleanup(func() {
		global.Config = originalConfig
		global.Log = originalLog
	})

	global.Config = &config.Config{Redis: &config.Redis{Enable: true}}
	global.Log = setupTestLogger()

	if client := InitRedis(); client != nil {
		t.Fatalf("expected nil client when redis addr missing, got %#v", client)
	}
}

func TestInitRedisAllowsCustomTimeouts(t *testing.T) {
	originalConfig := global.Config
	originalLog := global.Log
	t.Cleanup(func() {
		global.Config = originalConfig
		global.Log = originalLog
	})

	cfg := &config.Redis{
		Enable:       true,
		Addr:         "127.0.0.1:0",
		DialTimeout:  2 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 4 * time.Second,
	}

	global.Config = &config.Config{Redis: cfg}
	global.Log = setupTestLogger()

	if client := InitRedis(); client != nil {
		client.Close()
		t.Fatalf("expected nil client when addr invalid, got %#v", client)
	}
}
