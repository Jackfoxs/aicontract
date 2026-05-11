package compliance

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"backend/config"
	"backend/document/parser"
	"backend/global"
	"backend/llm"
	"backend/models"

	"github.com/bytedance/sonic"
	"github.com/redis/go-redis/v9"
	"xorm.io/xorm"
)

// Service 提供合规任务的核心能力
type Service struct {
	db            *xorm.Engine
	redis         redis.Cmdable
	cfg           *config.Compliance
	parserFactory *parser.ParserFactory
	matcher       *RuleMatcher
	highlights    *HighlightBuilder
	reports       *ReportBuilder
	analyzer      *LLMAnalyzer
	reqExtractor  *TenderRequirementExtractor
	comparator    *ResponseComparator
}

// RuleOption 提供前端选择的规范条目
type RuleOption struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	ContentPreview string `json:"content_preview"`
	RuleID         string `json:"rule_id"`
	SectionPath    string `json:"section_path"`
	Content        string `json:"content,omitempty"`
}

// NewService 构造合规服务
func NewService(db *xorm.Engine, redisClient redis.Cmdable, cfg *config.Compliance) (*Service, error) {
	if db == nil {
		return nil, errors.New("compliance service requires db engine")
	}
	if cfg == nil {
		cfg = &config.Compliance{}
		cfg.ApplyDefaults()
	}
	sanitizedRedis := normalizeRedisClient(redisClient)
	if sanitizedRedis == nil && global.Log != nil {
		global.Log.Warn("合规服务缺少 Redis 客户端，相关功能将降级")
	}
	llmClient := llm.NewDeepSeekClient()
	analyzer := NewLLMAnalyzer(llmClient, global.Config.ChatConfig.ModelType)
	analyzer.SetBatchConfig(cfg.LLMBatchSize, cfg.LLMConcurrency)
	comparator := NewResponseComparator(llmClient)
	comparator.SetBatchConfig(cfg.LLMBatchSize, cfg.LLMConcurrency)
	svc := &Service{
		db:            db,
		redis:         sanitizedRedis,
		cfg:           cfg,
		parserFactory: parser.NewParserFactory(),
		analyzer:      analyzer,
		reqExtractor:  NewTenderRequirementExtractor(llmClient),
		comparator:    comparator,
	}
	svc.matcher = NewRuleMatcher(db)
	svc.highlights = NewHighlightBuilder(db)
	svc.reports = NewReportBuilder(cfg)
	return svc, nil
}

func (s *Service) queueKey() string {
	if s.cfg != nil && s.cfg.QueueKey != "" {
		return s.cfg.QueueKey
	}
	return "compliance:jobs"
}

func normalizeRedisClient(client redis.Cmdable) redis.Cmdable {
	if client == nil {
		return nil
	}
	v := reflect.ValueOf(client)
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			return nil
		}
	}
	return client
}

// SubmitJob 创建任务并推入队列
func (s *Service) SubmitJob(ctx context.Context, input *SubmitJobInput) (*models.ComplianceJob, error) {
	if input == nil {
		return nil, errors.New("input cannot be nil")
	}
	threshold := input.AIThreshold
	if threshold <= 0 {
		threshold = s.cfg.AIThresholdDefault
		if threshold <= 0 {
			threshold = 0.75
		}
	}

	job := &models.ComplianceJob{
		TenderFileID:   input.TenderFileID,
		ResponseFileID: input.ResponseFileID,
		NormSetID:      input.NormSetID,
		Status:         models.ComplianceJobStatusPending,
		Progress:       0,
		AIThreshold:    threshold,
		CreatedBy:      input.CreatedBy,
	}

	if err := job.SetSelectedRuleIDs(input.SelectedRuleIDs); err != nil {
		return nil, fmt.Errorf("store selected rules failed: %w", err)
	}

	if _, err := s.db.Context(ctx).Insert(job); err != nil {
		return nil, fmt.Errorf("insert compliance job failed: %w", err)
	}

	if err := s.enqueueJob(ctx, job.ID); err != nil {
		return nil, err
	}

	return job, nil
}

// ProcessJob 执行任务处理流程
func (s *Service) ProcessJob(ctx context.Context, jobID uint64) error {
	var job models.ComplianceJob
	has, err := s.db.Context(ctx).ID(jobID).Get(&job)
	if err != nil {
		return fmt.Errorf("load job failed: %w", err)
	}
	if !has {
		return fmt.Errorf("job %d not found", jobID)
	}
	if job.Status == models.ComplianceJobStatusSuccess {
		return nil
	}

	if err := s.updateJob(ctx, &job, map[string]interface{}{
		"status":        models.ComplianceJobStatusRunning,
		"progress":      10,
		"error_message": "",
	}); err != nil {
		return err
	}

	tenderDoc, err := s.parseDocument(ctx, job.TenderFileID)
	if err != nil {
		return s.failJob(ctx, &job, fmt.Errorf("parse tender file failed: %w", err))
	}
	if err := s.updateJob(ctx, &job, map[string]interface{}{"progress": 30}); err != nil {
		return err
	}

	responseDoc, err := s.parseDocument(ctx, job.ResponseFileID)
	if err != nil {
		return s.failJob(ctx, &job, fmt.Errorf("parse response file failed: %w", err))
	}
	if err := s.updateJob(ctx, &job, map[string]interface{}{"progress": 50}); err != nil {
		return err
	}

	ruleIDs, err := job.SelectedRuleIDList()
	if err != nil {
		return s.failJob(ctx, &job, fmt.Errorf("decode rule ids failed: %w", err))
	}
	rules, err := s.matcher.LoadRules(ctx, ruleIDs)
	if err != nil {
		return s.failJob(ctx, &job, fmt.Errorf("load rules failed: %w", err))
	}
	matches := s.matcher.MatchRules(ctx, rules, tenderDoc.Content, responseDoc.Content)

	if err := s.updateJob(ctx, &job, map[string]interface{}{"progress": 70}); err != nil {
		return err
	}

	totalCost := 0.0
	issueResults := []IssueResult{}
	if len(matches) > 0 {
		var err error
		issueResults, totalCost, err = s.analyzer.Analyze(ctx, job.ID, matches, tenderDoc.Content, responseDoc.Content)
		if err != nil {
			return s.failJob(ctx, &job, err)
		}
	}

	// 解析比选要求并与响应文件对比
	comparatorResults := []IssueResult{}
	if s.reqExtractor != nil && s.comparator != nil {
		reqs, extractCost, err := s.reqExtractor.Extract(ctx, tenderDoc.Content)
		totalCost += extractCost
		if err != nil {
			return s.failJob(ctx, &job, err)
		}
		if len(reqs) > 0 {
			respResults, compareCost, err := s.comparator.Compare(ctx, job.ID, reqs, responseDoc.Content)
			totalCost += compareCost
			if err != nil {
				return s.failJob(ctx, &job, err)
			}
			comparatorResults = respResults
		}
	}

	allIssues := append(issueResults, comparatorResults...)

	if err := s.updateJob(ctx, &job, map[string]interface{}{
		"llm_cost":  totalCost,
		"llm_model": global.Config.ChatConfig.ModelType,
	}); err != nil && global.Log != nil {
		global.Log.Warn("更新LLM费用失败", "error", err)
	}

	if _, err := s.db.Context(ctx).Where("job_id = ?", job.ID).Delete(&models.ComplianceHighlight{}); err != nil {
		return s.failJob(ctx, &job, fmt.Errorf("cleanup highlights failed: %w", err))
	}
	highlightPairs, err := s.highlights.CreateHighlights(ctx, job.ID, allIssues, tenderDoc.Content, responseDoc.Content)
	if err != nil {
		return s.failJob(ctx, &job, fmt.Errorf("build highlights failed: %w", err))
	}

	storedIssues, err := s.persistIssues(ctx, &job, allIssues, highlightPairs)
	if err != nil {
		return s.failJob(ctx, &job, fmt.Errorf("persist issues failed: %w", err))
	}

	reports, err := s.reports.GenerateReports(ctx, &job, storedIssues, models.ComplianceJobStatusSuccess)
	if err != nil {
		return s.failJob(ctx, &job, fmt.Errorf("generate reports failed: %w", err))
	}
	if err := s.updateJob(ctx, &job, map[string]interface{}{
		"report_path_json": reports.JSON,
		"report_path_csv":  reports.CSV,
		"report_path_pdf":  reports.PDF,
	}); err != nil {
		return err
	}

	summary := summarizeIssues(storedIssues)
	if summary != "" {
		_ = s.updateJob(ctx, &job, map[string]interface{}{"analysis_summary": summary})
	}

	if err := s.updateJob(ctx, &job, map[string]interface{}{
		"status":        models.ComplianceJobStatusSuccess,
		"progress":      100,
		"error_message": "",
	}); err != nil {
		return err
	}

	return nil
}

func (s *Service) enqueueJob(ctx context.Context, jobID uint64) error {
	if s.redis == nil {
		global.Log.Warn("redis client missing, skip enqueue", "job_id", jobID)
		return nil
	}
	if err := s.redis.LPush(ctx, s.queueKey(), jobID).Err(); err != nil {
		return fmt.Errorf("enqueue job failed: %w", err)
	}
	return nil
}

func (s *Service) parseDocument(ctx context.Context, fileID uint64) (*ParsedDocument, error) {
	if fileID == 0 {
		return nil, errors.New("file id empty")
	}
	var log models.DocumentParseLog
	has, err := s.db.Context(ctx).ID(fileID).Get(&log)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, fmt.Errorf("parse log %d not found", fileID)
	}
	if strings.TrimSpace(log.FilePath) == "" {
		return nil, fmt.Errorf("parse log %d missing file path", fileID)
	}

	ext := log.FileType
	if ext == "" {
		ext = filepath.Ext(log.FilePath)
	}
	if !strings.HasPrefix(ext, ".") && ext != "" {
		ext = "." + ext
	}
	docParser, err := s.parserFactory.GetParser(strings.ToLower(ext))
	if err != nil {
		return nil, err
	}
	result, err := docParser.ParseWithMetadata(log.FilePath)
	if err != nil {
		return nil, err
	}

	metadataBytes, _ := sonic.Marshal(result.Metadata)
	parsed := &ParsedDocument{
		Content:  result.Content,
		Metadata: map[string]interface{}{},
	}
	if len(metadataBytes) > 0 {
		_ = sonic.Unmarshal(metadataBytes, &parsed.Metadata)
	}
	return parsed, nil
}

func (s *Service) persistIssues(ctx context.Context, job *models.ComplianceJob, issues []IssueResult, pairs map[uint64]HighlightPair) ([]models.ComplianceIssue, error) {
	if job == nil {
		return nil, errors.New("job is nil")
	}
	if _, err := s.db.Context(ctx).Where("job_id = ?", job.ID).Delete(&models.ComplianceIssue{}); err != nil {
		return nil, err
	}

	stored := make([]models.ComplianceIssue, 0, len(issues))
	for _, issue := range issues {
		record := &models.ComplianceIssue{
			JobID:           job.ID,
			ResponseExcerpt: issue.ResponseExcerpt,
			MatchScore:      issue.Score,
			Status:          issue.Status,
			Remark:          issue.Remark,
			LLMAdvice:       issue.Advice,
			LLMModel:        issue.LLMModel,
			LLMRaw:          issue.RawLLMResponse,
			SourceType:      issue.SourceType,
			Gap:             issue.Gap,
		}
		if record.SourceType == "" {
			record.SourceType = models.ComplianceIssueSourceRule
		}
		if len(issue.ResponseRefs) > 0 {
			refsJSON, err := sonic.Marshal(issue.ResponseRefs)
			if err != nil {
				return nil, err
			}
			record.ResponseRefs = string(refsJSON)
		}
		if record.SourceType == models.ComplianceIssueSourceTenderRequirement {
			record.RequirementID = strings.TrimSpace(issue.RequirementID)
			record.RequirementName = strings.TrimSpace(issue.RequirementName)
			record.RequirementLevel = strings.TrimSpace(issue.RequirementLevel)
			content := strings.TrimSpace(issue.RequirementText)
			if content != "" {
				record.RequiredContent = content
			}
			if record.RuleTitle == "" {
				record.RuleTitle = record.RequirementName
			}
		}
		if issue.Rule != nil {
			record.RuleID = issue.Rule.ID
			if strings.TrimSpace(record.RuleTitle) == "" {
				record.RuleTitle = issue.Rule.Title
			}
			if strings.TrimSpace(record.RequiredContent) == "" {
				record.RequiredContent = issue.Rule.Content
			}
			record.IsMandatory = issue.Rule.RuleID != ""
		}
		payload := map[string]interface{}{}
		if issue.Rule != nil {
			if pair, ok := pairs[issue.Rule.ID]; ok {
				if pair.ResponseID != 0 {
					payload["response_id"] = pair.ResponseID
				}
				if pair.TenderID != 0 {
					payload["tender_id"] = pair.TenderID
				}
			}
		}
		if issue.TenderExcerpt != "" {
			payload["tender_excerpt"] = issue.TenderExcerpt
		}
		if len(payload) > 0 {
			refJSON, err := sonic.Marshal(payload)
			if err != nil {
				return nil, err
			}
			record.HighlightRef = string(refJSON)
		}
		if _, err := s.db.Context(ctx).Insert(record); err != nil {
			return nil, err
		}
		stored = append(stored, *record)
	}
	return stored, nil
}

func summarizeIssues(issues []models.ComplianceIssue) string {
	if len(issues) == 0 {
		return ""
	}
	summaryMap := map[string]int{}
	for _, issue := range issues {
		summaryMap[issue.Status]++
	}
	parts := make([]string, 0, len(summaryMap))
	order := []string{"matched", "partial", "inconsistent", "missing"}
	for _, key := range order {
		if count, ok := summaryMap[key]; ok {
			parts = append(parts, fmt.Sprintf("%s:%d", key, count))
		}
	}
	for status, count := range summaryMap {
		if containsStatus(order, status) {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s:%d", status, count))
	}
	return strings.Join(parts, ",")
}

func containsStatus(order []string, status string) bool {
	for _, item := range order {
		if item == status {
			return true
		}
	}
	return false
}

// GetJobByID 查询单个任务
func (s *Service) GetJobByID(ctx context.Context, jobID uint64) (*models.ComplianceJob, error) {
	var job models.ComplianceJob
	has, err := s.db.Context(ctx).ID(jobID).Get(&job)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, fmt.Errorf("job %d not found", jobID)
	}
	return &job, nil
}

// ListJobs 列出任务
func (s *Service) ListJobs(ctx context.Context, page, pageSize int) ([]models.ComplianceJob, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	total, err := s.db.Context(ctx).Count(&models.ComplianceJob{})
	if err != nil {
		return nil, 0, err
	}

	jobs := []models.ComplianceJob{}
	offset := (page - 1) * pageSize
	if err := s.db.Context(ctx).OrderBy("created_at DESC").Limit(pageSize, offset).Find(&jobs); err != nil {
		return nil, 0, err
	}
	return jobs, total, nil
}

// ListRuleOptions 返回可选的规范条目列表
func (s *Service) ListRuleOptions(ctx context.Context, keyword string, page, pageSize int, includeContent bool) ([]RuleOption, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	trimmed := strings.TrimSpace(keyword)

	countSess := s.db.Context(ctx).Where("rule_id <> ''")
	if trimmed != "" {
		like := "%" + trimmed + "%"
		countSess = countSess.And("(title LIKE ? OR content LIKE ?)", like, like)
	}
	total, err := countSess.Count(&models.DocumentChunk{})
	if err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	var chunks []models.DocumentChunk
	dataSess := s.db.Context(ctx).Where("rule_id <> ''")
	if trimmed != "" {
		like := "%" + trimmed + "%"
		dataSess = dataSess.And("(title LIKE ? OR content LIKE ?)", like, like)
	}
	if err := dataSess.OrderBy("order_index ASC").Limit(pageSize, offset).Find(&chunks); err != nil {
		return nil, 0, err
	}
	options := make([]RuleOption, 0, len(chunks))
	for _, ch := range chunks {
		option := RuleOption{
			ID:             strconv.FormatUint(ch.ID, 10),
			Title:          strings.TrimSpace(ch.Title),
			ContentPreview: buildRulePreview(ch.Content),
			RuleID:         strings.TrimSpace(ch.RuleID),
			SectionPath:    strings.TrimSpace(ch.SectionPath),
		}
		if includeContent {
			option.Content = strings.TrimSpace(ch.Content)
		}
		options = append(options, option)
	}
	return options, total, nil
}

func buildRulePreview(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	const maxRunes = 160
	runes := []rune(content)
	if len(runes) <= maxRunes {
		return string(runes)
	}
	return string(runes[:maxRunes]) + "..."
}

// GetIssuesByJobID 获取任务问题
func (s *Service) GetIssuesByJobID(ctx context.Context, jobID uint64) ([]models.ComplianceIssue, error) {
	issues := []models.ComplianceIssue{}
	if err := s.db.Context(ctx).Where("job_id = ?", jobID).OrderBy("id ASC").Find(&issues); err != nil {
		return nil, err
	}
	return issues, nil
}

// GetHighlightsByJobID 获取高亮列表
func (s *Service) GetHighlightsByJobID(ctx context.Context, jobID uint64, fileRole string, page, pageSize int) ([]models.ComplianceHighlight, error) {
	sess := s.db.Context(ctx).Where("job_id = ?", jobID)
	if fileRole != "" {
		sess = sess.And("file_role = ?", fileRole)
	}
	if page > 0 && pageSize > 0 {
		offset := (page - 1) * pageSize
		sess = sess.Limit(pageSize, offset)
	}
	highlights := []models.ComplianceHighlight{}
	if err := sess.OrderBy("id ASC").Find(&highlights); err != nil {
		return nil, err
	}
	return highlights, nil
}

// RetryJob 将任务重新入队
func (s *Service) RetryJob(ctx context.Context, jobID uint64) error {
	job, err := s.GetJobByID(ctx, jobID)
	if err != nil {
		return err
	}
	if job.Status == models.ComplianceJobStatusRunning {
		return fmt.Errorf("job %d is running", jobID)
	}
	if err := s.updateJob(ctx, job, map[string]interface{}{
		"status":        models.ComplianceJobStatusPending,
		"progress":      0,
		"error_message": "",
	}); err != nil {
		return err
	}
	if err := s.enqueueJob(ctx, jobID); err != nil {
		return err
	}
	if s.redis != nil {
		retryKey := fmt.Sprintf("%s:retry:%d", s.queueKey(), jobID)
		_ = s.redis.Del(ctx, retryKey).Err()
	}
	return nil
}

func (s *Service) updateJob(ctx context.Context, job *models.ComplianceJob, fields map[string]interface{}) error {
	if job == nil {
		return errors.New("job is nil")
	}
	if len(fields) == 0 {
		return nil
	}
	_, err := s.db.Context(ctx).Table(&models.ComplianceJob{}).ID(job.ID).Update(fields)
	if err != nil {
		return err
	}
	if status, ok := fields["status"]; ok {
		job.Status = status.(string)
	}
	if progress, ok := fields["progress"]; ok {
		if p, ok2 := progress.(int); ok2 {
			job.Progress = p
		}
	}
	if errMsg, ok := fields["error_message"]; ok {
		if msg, ok2 := errMsg.(string); ok2 {
			job.ErrorMessage = msg
		}
	}
	return nil
}

func (s *Service) failJob(ctx context.Context, job *models.ComplianceJob, err error) error {
	msg := err.Error()
	_ = s.updateJob(ctx, job, map[string]interface{}{
		"status":        models.ComplianceJobStatusFailed,
		"error_message": msg,
		"progress":      job.Progress,
	})
	if global.Log != nil {
		global.Log.Error("compliance job failed", "job_id", job.ID, "error", msg)
	}
	return err
}

// EnsureReportDir 确保报告目录存在（供其他模块复用）
func (s *Service) EnsureReportDir() error {
	if s.cfg == nil {
		return errors.New("missing config")
	}
	dir := s.cfg.ReportOutputDir
	if dir == "" {
		dir = "uploads/compliance"
	}
	return os.MkdirAll(dir, 0o755)
}
