package core

import (
	"context"
	"time"

	"backend/global"

	"github.com/redis/go-redis/v9"
)

// InitRedis 初始化 Redis 客户端
func InitRedis() *redis.Client {
	cfg := global.Config.Redis
	if cfg == nil {
		if global.Log != nil {
			global.Log.Warn("未配置 redis，跳过初始化")
		}
		return nil
	}
	if !cfg.Enable {
		if global.Log != nil {
			global.Log.Info("redis 已禁用，跳过初始化")
		}
		return nil
	}
	if cfg.Addr == "" {
		if global.Log != nil {
			global.Log.Error("redis 地址为空，无法初始化")
		}
		return nil
	}

	opts := &redis.Options{
		Addr:     cfg.Addr,
		Username: cfg.Username,
		Password: cfg.Password,
		DB:       cfg.DB,
	}

	if cfg.PoolSize > 0 {
		opts.PoolSize = cfg.PoolSize
	}
	if cfg.MinIdleConns > 0 {
		opts.MinIdleConns = cfg.MinIdleConns
	}
	if cfg.DialTimeout > 0 {
		opts.DialTimeout = cfg.DialTimeout
	}
	if cfg.ReadTimeout > 0 {
		opts.ReadTimeout = cfg.ReadTimeout
	}
	if cfg.WriteTimeout > 0 {
		opts.WriteTimeout = cfg.WriteTimeout
	}

	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		if global.Log != nil {
			global.Log.Warn("redis 连接失败", "error", err)
		}
		_ = client.Close()
		return nil
	}

	if global.Log != nil {
		global.Log.Info("redis 初始化成功", "addr", cfg.Addr)
	}

	return client
}
