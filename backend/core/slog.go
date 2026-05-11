package core

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"backend/global"
)

// 颜色常量
const (
	colorRed    = "\x1b[31m"
	colorYellow = "\x1b[33m"
	colorBlue   = "\x1b[36m"
	colorGray   = "\x1b[37m"
	colorReset  = "\x1b[0m"
)

// ColorHandler 自定义彩色日志处理器
type ColorHandler struct {
	handler slog.Handler
	w       io.Writer
	level   slog.Level
	prefix  string
}

// NewColorHandler 创建新的彩色处理器
func NewColorHandler(w io.Writer, opts *slog.HandlerOptions) *ColorHandler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	return &ColorHandler{
		handler: slog.NewTextHandler(w, opts),
		w:       w,
		level:   opts.Level.Level(),
		prefix:  global.Config.Logger.Prefix,
	}
}

func (h *ColorHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *ColorHandler) Handle(ctx context.Context, r slog.Record) error {
	// 获取调用者信息
	var fileVal string
	if global.Config.Logger.ShowLine {
		// 跳过4层调用栈获取真实的调用位置
		_, file, line, ok := runtime.Caller(4)
		if ok {
			fileVal = fmt.Sprintf(" %s:%d", filepath.Base(file), line)
		}
	}

	// 获取时间戳
	timestamp := r.Time.Format("2006-01-02 15:04:05")

	// 根据日志级别选择颜色
	var levelColor string
	switch r.Level {
	case slog.LevelDebug:
		levelColor = colorGray
	case slog.LevelWarn:
		levelColor = colorYellow
	case slog.LevelError:
		levelColor = colorRed
	default:
		levelColor = colorBlue
	}

	// 格式化输出
	fmt.Fprintf(h.w, "%s[%s] %s[%s]%s%s %s\n",
		h.prefix,
		timestamp,
		levelColor,
		r.Level.String(),
		colorReset,
		fileVal,
		r.Message)

	return nil
}

func (h *ColorHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *ColorHandler) WithGroup(name string) slog.Handler {
	return h
}

// InitLogger 初始化slog日志器
func InitLogger() *slog.Logger {
	// 解析日志级别
	var level slog.Level
	switch global.Config.Logger.Level {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// 创建处理器选项
	opts := &slog.HandlerOptions{
		Level: level,
	}

	// 创建彩色处理器
	handler := NewColorHandler(os.Stdout, opts)

	// 创建日志器
	logger := slog.New(handler)

	// 设置为默认日志器
	slog.SetDefault(logger)

	return logger
}
