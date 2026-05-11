//go:build integration
// +build integration

package compliance

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"backend/config"
	"backend/global"
	"backend/models"
	"backend/utils"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"xorm.io/xorm"

	_ "github.com/mattn/go-sqlite3"
)

func setupLogger() {
	handler := slog.NewTextHandler(io.Discard, nil)
	global.Log = slog.New(handler)
}

func setupTestService(t *testing.T) (*Service, *xorm.Engine, *redis.Client, func()) {
	t.Helper()
	setupLogger()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("run miniredis: %v", err)
	}
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	db, err := xorm.NewEngine("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("create sqlite engine: %v", err)
	}
	if err := db.Sync2(models.GetModels()...); err != nil {
		t.Fatalf("sync models: %v", err)
	}

	cfg := &config.Compliance{QueueKey: "test:compliance"}
	cfg.ApplyDefaults()
	cfg.ReportOutputDir = t.TempDir()

	svc, err := NewService(db, redisClient, cfg)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	cleanup := func() {
		redisClient.Close()
		mr.Close()
		db.Close()
	}
	return svc, db, redisClient, cleanup
}

func TestSubmitAndProcessJob(t *testing.T) {
	svc, db, redisClient, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// 准备测试文件
	tempDir := t.TempDir()
	tenderPath := filepath.Join(tempDir, "tender.txt")
	responsePath := filepath.Join(tempDir, "response.txt")

	tenderContent := "本项目要求供应商提供医疗器械质保承诺"
	responseContent := "供应商承诺提供完整的医疗器械质保承诺"

	if err := os.WriteFile(tenderPath, []byte(tenderContent), 0o644); err != nil {
		t.Fatalf("write tender file: %v", err)
	}
	if err := os.WriteFile(responsePath, []byte(responseContent), 0o644); err != nil {
		t.Fatalf("write response file: %v", err)
	}

	tenderLog := &models.DocumentParseLog{
		FilePath:  tenderPath,
		FileType:  "txt",
		Success:   true,
		CreatedAt: time.Now(),
	}
	responseLog := &models.DocumentParseLog{
		FilePath:  responsePath,
		FileType:  "txt",
		Success:   true,
		CreatedAt: time.Now(),
	}

	if _, err := db.Insert(tenderLog); err != nil {
		t.Fatalf("insert tender log: %v", err)
	}
	if _, err := db.Insert(responseLog); err != nil {
		t.Fatalf("insert response log: %v", err)
	}

	ruleID := utils.GenerateID()
	rule := &models.DocumentChunk{
		ID:      ruleID,
		Title:   "质保要求",
		Content: "医疗器械质保承诺",
	}
	if _, err := db.Insert(rule); err != nil {
		t.Fatalf("insert rule chunk: %v", err)
	}

	job, err := svc.SubmitJob(ctx, &SubmitJobInput{
		TenderFileID:    tenderLog.ID,
		ResponseFileID:  responseLog.ID,
		SelectedRuleIDs: []uint64{ruleID},
		CreatedBy:       1001,
	})
	if err != nil {
		t.Fatalf("submit job failed: %v", err)
	}

	vals, err := redisClient.LRange(ctx, svc.cfg.QueueKey, 0, -1).Result()
	if err != nil {
		t.Fatalf("check redis queue: %v", err)
	}
	if len(vals) != 1 || vals[0] != strconv.FormatUint(job.ID, 10) {
		t.Fatalf("unexpected queue values: %v", vals)
	}

	if err := svc.ProcessJob(ctx, job.ID); err != nil {
		t.Fatalf("process job failed: %v", err)
	}

	var stored models.ComplianceJob
	has, err := db.ID(job.ID).Get(&stored)
	if err != nil || !has {
		t.Fatalf("load stored job: %v has=%v", err, has)
	}
	if stored.Status != models.ComplianceJobStatusSuccess {
		t.Fatalf("unexpected job status: %s", stored.Status)
	}

	var issues []models.ComplianceIssue
	if err := db.Where("job_id = ?", job.ID).Find(&issues); err != nil {
		t.Fatalf("load issues: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	if issues[0].Status != "matched" {
		t.Fatalf("unexpected issue status: %s", issues[0].Status)
	}
}
