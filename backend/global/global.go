package global

import (
	"backend/config"
	"log/slog"

	"github.com/redis/go-redis/v9"
	"xorm.io/xorm"
)

var (
	Config *config.Config
	DB     *xorm.Engine
	Log    *slog.Logger
	Redis  *redis.Client
)
