package compliance

import (
	"backend/global"
	"backend/llm"
	"backend/models"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/sonic"
)

// ChatClient 定义与 LLM 交互所需的接口，便于测试
type ChatClient interface {
	ChatWithConfig(ctx context.Context, systemPrompt, userPrompt string, temperature float64, maxTokens int) (string, float64, error)
}

// LLMAnalyzer 负责调用大模型对响应文件做合规判定
type LLMAnalyzer struct {
	client      ChatClient
	model       string
	batchSize   int
	concurrency int
}

// NewLLMAnalyzer 创建分析器
func NewLLMAnalyzer(client ChatClient, model string) *LLMAnalyzer {
	return &LLMAnalyzer{
		client:      client,
		model:       model,
		batchSize:   5,
		concurrency: 3,
	}
}

// SetBatchConfig 设置批量调用参数
func (a *LLMAnalyzer) SetBatchConfig(batchSize, concurrency int) {
	if batchSize > 0 {
		a.batchSize = batchSize
	}
	if concurrency > 0 {
		a.concurrency = concurrency
	}
}

// batchResult 单个批次的调用结果
type batchResult struct {
	batchIdx int
	results  []IssueResult
	cost     float64
	err      error
}

// Analyze 批量合并 + 并发调用 LLM 审核
func (a *LLMAnalyzer) Analyze(ctx context.Context, jobID uint64, matches []RuleMatch, tenderText, responseText string) ([]IssueResult, float64, error) {
	if a == nil || a.client == nil {
		return nil, 0, fmt.Errorf("llm analyzer not initialized")
	}
	if len(matches) == 0 {
		return []IssueResult{}, 0, nil
	}

	// 分批
	batches := splitMatchBatches(matches, a.batchSize)
	resultCh := make(chan batchResult, len(batches))

	// 并发控制
	sem := make(chan struct{}, a.concurrency)
	var wg sync.WaitGroup

	for i, batch := range batches {
		wg.Add(1)
		go func(idx int, batchMatches []RuleMatch) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			results, cost, err := a.analyzeBatch(ctx, jobID, idx, batchMatches)
			resultCh <- batchResult{batchIdx: idx, results: results, cost: cost, err: err}
		}(i, batch)
	}

	wg.Wait()
	close(resultCh)

	// 按批次顺序收集结果
	batchResults := make([]batchResult, len(batches))
	for br := range resultCh {
		batchResults[br.batchIdx] = br
	}

	allResults := make([]IssueResult, 0, len(matches))
	var totalCost float64
	for _, br := range batchResults {
		if br.err != nil {
			return nil, totalCost, br.err
		}
		totalCost += br.cost
		allResults = append(allResults, br.results...)
	}

	return allResults, totalCost, nil
}

// analyzeBatch 处理单个批次的 LLM 调用
func (a *LLMAnalyzer) analyzeBatch(ctx context.Context, jobID uint64, batchIdx int, matches []RuleMatch) ([]IssueResult, float64, error) {
	// 单条走原逻辑，避免批量 prompt 解析开销
	if len(matches) == 1 {
		return a.analyzeSingle(ctx, jobID, matches[0])
	}

	systemPrompt := buildBatchSystemPrompt()
	userPrompt := buildBatchUserPrompt(matches)

	start := time.Now()
	if global.Log != nil {
		global.Log.Info("合规LLM批量调用开始",
			"job_id", jobID,
			"batch_idx", batchIdx,
			"batch_size", len(matches),
			"prompt_len", len([]rune(userPrompt)),
		)
	}

	// 批量 prompt 需要更大的 maxTokens
	maxTokens := 2048 * len(matches)
	if maxTokens > 8192 {
		maxTokens = 8192
	}

	raw, cost, err := a.client.ChatWithConfig(ctx, systemPrompt, userPrompt, 0.2, maxTokens)
	if err != nil {
		if global.Log != nil {
			global.Log.Error("合规LLM批量调用失败",
				"job_id", jobID,
				"batch_idx", batchIdx,
				"error", err,
				"duration_ms", time.Since(start).Milliseconds(),
			)
		}
		return nil, 0, fmt.Errorf("LLM 批量调用失败(batch %d): %w", batchIdx, err)
	}

	if global.Log != nil {
		global.Log.Info("合规LLM批量调用完成",
			"job_id", jobID,
			"batch_idx", batchIdx,
			"duration_ms", time.Since(start).Milliseconds(),
			"cost_cny", cost,
			"response_len", len(raw),
		)
	}

	parsedList, err := parseBatchLLMResponse(raw, len(matches))
	if err != nil {
		// 批量解析失败，降级为逐条调用
		if global.Log != nil {
			global.Log.Warn("批量解析失败，降级逐条调用",
				"job_id", jobID,
				"batch_idx", batchIdx,
				"error", err,
			)
		}
		return a.analyzeFallback(ctx, jobID, matches)
	}

	results := make([]IssueResult, 0, len(matches))
	for i, match := range matches {
		parsed := parsedList[i]
		result := buildIssueResult(match, parsed, raw, a.model)
		results = append(results, result)
	}

	return results, cost, nil
}

// analyzeSingle 单条规则调用（批次只有1条时使用）
func (a *LLMAnalyzer) analyzeSingle(ctx context.Context, jobID uint64, match RuleMatch) ([]IssueResult, float64, error) {
	systemPrompt := buildSystemPrompt()
	userPrompt := buildUserPrompt(match)

	ruleID := uint64(0)
	ruleTitle := ""
	if match.Rule != nil {
		ruleID = match.Rule.ID
		ruleTitle = strings.TrimSpace(match.Rule.Title)
	}
	start := time.Now()
	if global.Log != nil {
		global.Log.Info("合规LLM调用开始",
			"job_id", jobID,
			"rule_id", ruleID,
			"rule_title", ruleTitle,
			"prompt_len", len([]rune(userPrompt)),
		)
	}

	raw, cost, err := a.client.ChatWithConfig(ctx, systemPrompt, userPrompt, 0.2, 2048)
	if err != nil {
		if global.Log != nil {
			global.Log.Error("合规LLM调用失败",
				"job_id", jobID,
				"rule_id", ruleID,
				"error", err,
				"duration_ms", time.Since(start).Milliseconds(),
			)
		}
		return nil, 0, fmt.Errorf("LLM 调用失败: %w", err)
	}

	if global.Log != nil {
		global.Log.Info("合规LLM调用完成",
			"job_id", jobID,
			"rule_id", ruleID,
			"duration_ms", time.Since(start).Milliseconds(),
			"cost_cny", cost,
			"response_len", len(raw),
		)
	}

	parsed, err := parseLLMResponse(raw)
	if err != nil {
		return nil, cost, fmt.Errorf("解析 LLM 响应失败: %w", err)
	}

	result := buildIssueResult(match, parsed, raw, a.model)
	return []IssueResult{result}, cost, nil
}

// analyzeFallback 批量解析失败时降级为逐条调用
func (a *LLMAnalyzer) analyzeFallback(ctx context.Context, jobID uint64, matches []RuleMatch) ([]IssueResult, float64, error) {
	results := make([]IssueResult, 0, len(matches))
	var totalCost float64
	for _, match := range matches {
		r, cost, err := a.analyzeSingle(ctx, jobID, match)
		if err != nil {
			return nil, totalCost, err
		}
		totalCost += cost
		results = append(results, r...)
	}
	return results, totalCost, nil
}

// buildIssueResult 从解析结果构建 IssueResult
func buildIssueResult(match RuleMatch, parsed *llmStructuredResponse, raw, model string) IssueResult {
	status := normalizeLLMStatus(parsed.Status)
	if status == "" {
		status = "missing"
	}
	score := clampFloat(parsed.Confidence, 0, 1)
	remark := strings.TrimSpace(parsed.Reason)
	if remark == "" {
		remark = strings.TrimSpace(parsed.Summary)
	}

	result := IssueResult{
		Rule:            match.Rule,
		Status:          status,
		Score:           score,
		Remark:          remark,
		ResponseExcerpt: strings.TrimSpace(parsed.Evidence),
		TenderExcerpt:   match.TenderExcerpt,
		Advice:          strings.TrimSpace(parsed.Advice),
		RawLLMResponse:  raw,
		LLMModel:        model,
		SourceType:      models.ComplianceIssueSourceRule,
	}
	if result.ResponseExcerpt == "" {
		result.ResponseExcerpt = match.ResponseContext
	}
	if result.TenderExcerpt == "" {
		result.TenderExcerpt = match.TenderContext
	}
	return result
}

// splitMatchBatches 将 matches 按 batchSize 分组
func splitMatchBatches(matches []RuleMatch, batchSize int) [][]RuleMatch {
	if batchSize <= 0 {
		batchSize = 5
	}
	batches := make([][]RuleMatch, 0, (len(matches)+batchSize-1)/batchSize)
	for i := 0; i < len(matches); i += batchSize {
		end := i + batchSize
		if end > len(matches) {
			end = len(matches)
		}
		batches = append(batches, matches[i:end])
	}
	return batches
}

func buildSystemPrompt() string {
	return `你是一名医疗采购合规审核专家，擅长根据规范条目和供应商响应文件进行逐条审查。
输出必须严谨，禁止编造不存在的内容。`
}

func buildBatchSystemPrompt() string {
	return `你是一名医疗采购合规审核专家，擅长根据规范条目和供应商响应文件进行批量审查。
你将收到多条规范条目，需要逐条判定并输出 JSON 数组。
输出必须严谨，禁止编造不存在的内容。`
}

func buildUserPrompt(match RuleMatch) string {
	ruleTitle := strings.TrimSpace(match.Rule.Title)
	if ruleTitle == "" {
		ruleTitle = "未命名条目"
	}
	ruleContent := strings.TrimSpace(match.Rule.Content)
	tenderContext := strings.TrimSpace(match.TenderContext)
	if tenderContext == "" {
		tenderContext = "未提供比选文件上下文"
	}
	responseContext := strings.TrimSpace(match.ResponseContext)
	if responseContext == "" {
		responseContext = "未检索到疑似响应内容"
	}

	return fmt.Sprintf(`【规范条目】
标题：%s
要求：%s

【比选文件上下文】
%s

【供应商响应上下文】
%s

请根据以上信息判断供应商是否满足该条规范。
输出严格使用如下 JSON，键名必须一致，不得添加额外字段：
{
  "status": "满足|部分满足|不满足",
  "confidence": 0.0-1.0,
  "evidence": "引用或概括的响应内容",
  "reason": "简要说明判定依据",
  "advice": "若不满足，提示整改建议；若满足可写""",
  "summary": "如有需要可提供补充摘要"
}
如果上下文不足以判断，请返回"status":"不满足"并在 reason 中说明原因。`,
		ruleTitle, ruleContent, tenderContext, responseContext)
}

func buildBatchUserPrompt(matches []RuleMatch) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("以下共 %d 条规范条目，请逐条审查并输出 JSON 数组。\n\n", len(matches)))

	for i, match := range matches {
		ruleTitle := strings.TrimSpace(match.Rule.Title)
		if ruleTitle == "" {
			ruleTitle = "未命名条目"
		}
		ruleContent := strings.TrimSpace(match.Rule.Content)
		tenderContext := strings.TrimSpace(match.TenderContext)
		if tenderContext == "" {
			tenderContext = "未提供比选文件上下文"
		}
		responseContext := strings.TrimSpace(match.ResponseContext)
		if responseContext == "" {
			responseContext = "未检索到疑似响应内容"
		}

		sb.WriteString(fmt.Sprintf("--- 条目 %d ---\n", i+1))
		sb.WriteString(fmt.Sprintf("标题：%s\n要求：%s\n", ruleTitle, ruleContent))
		sb.WriteString(fmt.Sprintf("【比选文件上下文】\n%s\n", tenderContext))
		sb.WriteString(fmt.Sprintf("【供应商响应上下文】\n%s\n\n", responseContext))
	}

	sb.WriteString(`请输出严格的 JSON 数组，每个元素对应一条规范，顺序与输入一致：
[
  {
    "status": "满足|部分满足|不满足",
    "confidence": 0.0-1.0,
    "evidence": "引用或概括的响应内容",
    "reason": "简要说明判定依据",
    "advice": "若不满足，提示整改建议；若满足可写""",
    "summary": "如有需要可提供补充摘要"
  }
]
数组长度必须等于条目数量。如果上下文不足以判断，请将对应条目的 status 置为"不满足"。`)

	return sb.String()
}

type llmStructuredResponse struct {
	Status     string  `json:"status"`
	Confidence float64 `json:"confidence"`
	Evidence   string  `json:"evidence"`
	Reason     string  `json:"reason"`
	Advice     string  `json:"advice"`
	Summary    string  `json:"summary"`
}

func parseLLMResponse(raw string) (*llmStructuredResponse, error) {
	cleaned := extractJSONSegment(raw)
	resp := &llmStructuredResponse{}
	if err := sonic.UnmarshalString(cleaned, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// parseBatchLLMResponse 解析批量 JSON 数组响应
func parseBatchLLMResponse(raw string, expectedCount int) ([]*llmStructuredResponse, error) {
	trimmed := strings.TrimSpace(raw)
	arrStr := extractJSONArraySegment(trimmed)
	if arrStr == "" {
		return nil, fmt.Errorf("未找到 JSON 数组")
	}

	var list []*llmStructuredResponse
	if err := sonic.UnmarshalString(arrStr, &list); err != nil {
		return nil, fmt.Errorf("解析 JSON 数组失败: %w", err)
	}

	if len(list) != expectedCount {
		return nil, fmt.Errorf("返回条目数 %d 与期望 %d 不一致", len(list), expectedCount)
	}

	return list, nil
}

func extractJSONSegment(raw string) string {
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start >= 0 && end > start {
		return raw[start : end+1]
	}
	return raw
}

func extractJSONArraySegment(raw string) string {
	start := strings.Index(raw, "[")
	end := strings.LastIndex(raw, "]")
	if start >= 0 && end > start {
		return raw[start : end+1]
	}
	return ""
}

func normalizeLLMStatus(status string) string {
	trimmed := strings.TrimSpace(status)
	switch trimmed {
	case "满足", "完全满足", "满足要求":
		return "matched"
	case "部分满足", "部分符合", "部分满足要求":
		return "partial"
	case "不满足", "不符合", "缺失", "无法判断":
		return "missing"
	default:
		return ""
	}
}

func clampFloat(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// Ensure interface实现
var _ ChatClient = (*llm.DeepSeekClient)(nil)
