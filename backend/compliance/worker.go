package compliance

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"backend/config"
	"backend/global"
	"backend/models"

	"github.com/redis/go-redis/v9"
)

// Worker 消费合规任务队列并触发处理
type Worker struct {
	service   *Service
	queue     string
	redis     redis.Cmdable
	cfg       *config.Compliance
	stopOnce  sync.Once
	stopCh    chan struct{}
	waitGroup sync.WaitGroup
}

// NewWorker 创建 Worker 实例
func NewWorker(service *Service) *Worker {
	queue := "compliance:jobs"
	var cfg *config.Compliance
	var redisClient redis.Cmdable
	if service != nil {
		cfg = service.cfg
		if cfg != nil && cfg.QueueKey != "" {
			queue = cfg.QueueKey
		}
		redisClient = normalizeRedisClient(service.redis)
	}
	return &Worker{
		service: service,
		queue:   queue,
		redis:   redisClient,
		cfg:     cfg,
		stopCh:  make(chan struct{}),
	}
}

// Start 启动 Worker，按配置并发消费队列
func (w *Worker) Start(ctx context.Context) error {
	if w.service == nil {
		return errors.New("compliance worker missing service")
	}
	if w.redis == nil {
		if global.Log != nil {
			global.Log.Warn("合规 Worker 未启动，缺少 Redis 客户端")
		}
		return nil
	}
	concurrency := 1
	if w.cfg != nil && w.cfg.WorkerConcurrency > 0 {
		concurrency = w.cfg.WorkerConcurrency
	}
	for i := 0; i < concurrency; i++ {
		w.waitGroup.Add(1)
		go w.loop(ctx, i+1)
	}
	if global.Log != nil {
		global.Log.Info("合规 Worker 启动", slog.Int("concurrency", concurrency), slog.String("queue", w.queue))
	}
	return nil
}

// Shutdown 停止 Worker 并等待所有协程退出
func (w *Worker) Shutdown(ctx context.Context) {
	w.stopOnce.Do(func() {
		close(w.stopCh)
	})
	done := make(chan struct{})
	go func() {
		w.waitGroup.Wait()
		close(done)
	}()
	select {
	case <-ctx.Done():
	case <-done:
	}
}

func (w *Worker) loop(ctx context.Context, workerID int) {
	defer w.waitGroup.Done()
	queueKey := w.queue
	backoff := time.Second
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		default:
		}

		result, err := w.redis.BRPop(ctx, time.Second*5, queueKey).Result()
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			if err == redis.Nil {
				continue
			}
			if global.Log != nil {
				global.Log.Warn("合规 Worker 拉取任务失败", slog.Int("worker", workerID), slog.String("error", err.Error()))
			}
			time.Sleep(backoff)
			if backoff < 10*time.Second {
				backoff *= 2
			}
			continue
		}
		backoff = time.Second
		if len(result) < 2 {
			continue
		}
		jobIDStr := result[1]
		jobID, err := strconv.ParseUint(jobIDStr, 10, 64)
		if err != nil {
			if global.Log != nil {
				global.Log.Warn("合规 Worker 获取任务ID失败", slog.String("job_id", jobIDStr), slog.String("error", err.Error()))
			}
			continue
		}

		timeout := 10 * time.Minute
		if w.cfg != nil && w.cfg.TaskTimeout > 0 {
			timeout = w.cfg.TaskTimeout
		}
		jobCtx, cancel := context.WithTimeout(ctx, timeout)
		err = w.service.ProcessJob(jobCtx, jobID)
		cancel()
		if err != nil {
			w.handleFailure(ctx, jobID, err)
		} else {
			w.cleanupRetry(ctx, jobID)
		}
	}
}

func (w *Worker) handleFailure(ctx context.Context, jobID uint64, procErr error) {
	if global.Log != nil {
		global.Log.Error("合规任务处理失败", slog.Uint64("job_id", jobID), slog.String("error", procErr.Error()))
	}
	if w.redis == nil {
		return
	}
	limit := 3
	if w.cfg != nil && w.cfg.RetryLimit > 0 {
		limit = w.cfg.RetryLimit
	}
	retryKey := fmt.Sprintf("%s:retry:%d", w.queue, jobID)
	attempts, err := w.redis.Incr(ctx, retryKey).Result()
	if err != nil {
		if global.Log != nil {
			global.Log.Warn("合规任务记录重试失败", slog.Uint64("job_id", jobID), slog.String("error", err.Error()))
		}
		return
	}
	_ = w.redis.Expire(ctx, retryKey, time.Hour)
	if attempts > int64(limit) {
		_ = w.redis.Del(ctx, retryKey).Err()
		if global.Log != nil {
			global.Log.Error("合规任务超过重试次数", slog.Uint64("job_id", jobID), slog.Int64("attempts", attempts))
		}
		return
	}
	if err := w.redis.RPush(ctx, w.queue, jobID).Err(); err != nil {
		if global.Log != nil {
			global.Log.Warn("合规任务重新入队失败", slog.Uint64("job_id", jobID), slog.String("error", err.Error()))
		}
		return
	}
	if _, err := w.service.db.Context(ctx).Table(&models.ComplianceJob{}).ID(jobID).
		Update(map[string]interface{}{
			"status":   models.ComplianceJobStatusPending,
			"progress": 0,
		}); err != nil {
		if global.Log != nil {
			global.Log.Warn("合规任务重试时更新状态失败", slog.Uint64("job_id", jobID), slog.String("error", err.Error()))
		}
	}
}

func (w *Worker) cleanupRetry(ctx context.Context, jobID uint64) {
	if w.redis == nil {
		return
	}
	retryKey := fmt.Sprintf("%s:retry:%d", w.queue, jobID)
	if err := w.redis.Del(ctx, retryKey).Err(); err != nil && err != redis.Nil {
		if global.Log != nil {
			global.Log.Warn("合规任务清理重试信息失败", slog.Uint64("job_id", jobID), slog.String("error", err.Error()))
		}
	}
}
