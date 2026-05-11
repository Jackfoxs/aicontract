package compliance

import (
	"backend/global"
	"backend/models"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/sonic"
)

// ResponseComparator 使用 LLM 比较响应文件是否满足比选要求
type ResponseComparator struct {
	client      ChatClient
	batchSize   int
	concurrency int
}

func NewResponseComparator(client ChatClient) *ResponseComparator {
	return &ResponseComparator{
		client:      client,
		batchSize:   5,
		concurrency: 3,
	}
}

// SetBatchConfig 设置批量调用参数
func (c *ResponseComparator) SetBatchConfig(batchSize, concurrency int) {
	if batchSize > 0 {
		c.batchSize = batchSize
	}
	if concurrency > 0 {
		c.concurrency = concurrency
	}
}

type compareBatchResult struct {
	batchIdx int
	results  []IssueResult
	cost     float64
	err      error
}

func (c *ResponseComparator) Compare(ctx context.Context, jobID uint64, requirements []TenderRequirement, responseText string) ([]IssueResult, float64, error) {
	if c == nil || c.client == nil {
		return nil, 0, fmt.Errorf("response comparator not initialized")
	}
	if len(requirements) == 0 {
		return []IssueResult{}, 0, nil
	}

	batches := splitRequirementBatches(requirements, c.batchSize)
	resultCh := make(chan compareBatchResult, len(batches))

	sem := make(chan struct{}, c.concurrency)
	var wg sync.WaitGroup

	for i, batch := range batches {
		wg.Add(1)
		go func(idx int, reqs []TenderRequirement) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			results, cost, err := c.compareBatch(ctx, jobID, idx, reqs, responseText)
			resultCh <- compareBatchResult{batchIdx: idx, results: results, cost: cost, err: err}
		}(i, batch)
	}

	wg.Wait()
	close(resultCh)

	batchResults := make([]compareBatchResult, len(batches))
	for br := range resultCh {
		batchResults[br.batchIdx] = br
	}

	allResults := make([]IssueResult, 0, len(requirements))
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

func (c *ResponseComparator) compareBatch(ctx context.Context, jobID uint64, batchIdx int, reqs []TenderRequirement, responseText string) ([]IssueResult, float64, error) {
	if len(reqs) == 1 {
		return c.compareSingle(ctx, jobID, reqs[0], responseText)
	}

	systemPrompt := responseBatchSystemPrompt()
	userPrompt := buildBatchResponsePrompt(reqs, responseText)

	start := time.Now()
	if global.Log != nil {
		global.Log.Info("响应比对批量调用开始",
			"job_id", jobID,
			"batch_idx", batchIdx,
			"batch_size", len(reqs),
			"prompt_len", len([]rune(userPrompt)),
		)
	}

	maxTokens := 2048 * len(reqs)
	if maxTokens > 8192 {
		maxTokens = 8192
	}

	raw, cost, err := c.client.ChatWithConfig(ctx, systemPrompt, userPrompt, 0.2, maxTokens)
	if err != nil {
		if global.Log != nil {
			global.Log.Error("响应比对批量调用失败",
				"job_id", jobID,
				"batch_idx", batchIdx,
				"error", err,
				"duration_ms", time.Since(start).Milliseconds(),
			)
		}
		return nil, 0, fmt.Errorf("响应比对批量调用失败(batch %d): %w", batchIdx, err)
	}

	if global.Log != nil {
		global.Log.Info("响应比对批量调用完成",
			"job_id", jobID,
			"batch_idx", batchIdx,
			"duration_ms", time.Since(start).Milliseconds(),
			"cost_cny", cost,
		)
	}

	parsedList, err := parseBatchComparisonResponse(raw, len(reqs))
	if err != nil {
		if global.Log != nil {
			global.Log.Warn("批量比对解析失败，降级逐条调用",
				"job_id", jobID,
				"batch_idx", batchIdx,
				"error", err,
			)
		}
		return c.compareFallback(ctx, jobID, reqs, responseText)
	}

	results := make([]IssueResult, 0, len(reqs))
	for i, req := range reqs {
		parsed := parsedList[i]
		results = append(results, buildCompareIssueResult(req, parsed, raw))
	}

	return results, cost, nil
}

func (c *ResponseComparator) compareSingle(ctx context.Context, jobID uint64, req TenderRequirement, responseText string) ([]IssueResult, float64, error) {
	prompt := buildResponsePrompt(req, responseText)
	if global.Log != nil {
		global.Log.Info("响应比对开始",
			"job_id", jobID,
			"requirement_id", req.ID,
			"prompt_len", len([]rune(prompt)),
		)
	}
	start := time.Now()
	raw, cost, err := c.client.ChatWithConfig(ctx, responseSystemPrompt(), prompt, 0.2, 2048)
	if err != nil {
		if global.Log != nil {
			global.Log.Error("响应比对失败",
				"job_id", jobID,
				"requirement_id", req.ID,
				"error", err,
			)
		}
		return nil, 0, fmt.Errorf("响应比对失败: %w", err)
	}
	if global.Log != nil {
		global.Log.Info("响应比对完成",
			"job_id", jobID,
			"requirement_id", req.ID,
			"status", "",
			"duration_ms", time.Since(start).Milliseconds(),
		)
	}
	parsed, err := parseComparisonResponse(raw)
	if err != nil {
		return nil, cost, fmt.Errorf("解析响应比对结果失败: %w", err)
	}
	return []IssueResult{buildCompareIssueResult(req, parsed, raw)}, cost, nil
}

func (c *ResponseComparator) compareFallback(ctx context.Context, jobID uint64, reqs []TenderRequirement, responseText string) ([]IssueResult, float64, error) {
	results := make([]IssueResult, 0, len(reqs))
	var totalCost float64
	for _, req := range reqs {
		r, cost, err := c.compareSingle(ctx, jobID, req, responseText)
		if err != nil {
			return nil, totalCost, err
		}
		totalCost += cost
		results = append(results, r...)
	}
	return results, totalCost, nil
}

func buildCompareIssueResult(req TenderRequirement, parsed *comparisonResponse, raw string) IssueResult {
	issue := IssueResult{
		Status:           normalizeLLMStatus(parsed.Status),
		Score:            clampFloat(parsed.Confidence, 0, 1),
		Remark:           strings.TrimSpace(parsed.Reason),
		ResponseExcerpt:  strings.TrimSpace(parsed.Evidence),
		TenderExcerpt:    strings.TrimSpace(req.Source),
		Advice:           strings.TrimSpace(parsed.Advice),
		Gap:              strings.TrimSpace(parsed.Gap),
		RawLLMResponse:   raw,
		LLMModel:         global.Config.ChatConfig.ModelType,
		SourceType:       models.ComplianceIssueSourceTenderRequirement,
		RequirementID:    req.ID,
		RequirementName:  req.Title,
		RequirementLevel: req.Level,
		RequirementText:  req.Description,
		ResponseRefs:     parsed.ResponseRefs,
	}
	if issue.Status == "" {
		issue.Status = "missing"
	}
	if issue.Remark == "" {
		issue.Remark = issue.Gap
	}
	return issue
}

func splitRequirementBatches(reqs []TenderRequirement, batchSize int) [][]TenderRequirement {
	if batchSize <= 0 {
		batchSize = 5
	}
	batches := make([][]TenderRequirement, 0, (len(reqs)+batchSize-1)/batchSize)
	for i := 0; i < len(reqs); i += batchSize {
		end := i + batchSize
		if end > len(reqs) {
			end = len(reqs)
		}
		batches = append(batches, reqs[i:end])
	}
	return batches
}

func responseSystemPrompt() string {
	return "你是一名医疗采购评审专家，负责判断供应商响应是否满足比选要求，输出结构化 JSON。"
}

func responseBatchSystemPrompt() string {
	return "你是一名医疗采购评审专家，负责批量判断供应商响应是否满足多条比选要求，输出结构化 JSON 数组。"
}

func buildResponsePrompt(req TenderRequirement, response string) string {
	return fmt.Sprintf(`【比选要求】
ID: %s
标题: %s
级别: %s
要求描述: %s
验收标准: %s

【响应文件内容】
%s

请判断响应文件是否满足该要求，输出 JSON：
{
  "status": "满足|部分满足|不满足",
  "confidence": 0.0-1.0,
  "evidence": "响应文件中的佐证",
  "reason": "判定依据解释",
  "gap": "未满足内容或差距，没有则留空",
  "advice": "整改建议，没有则留空",
  "response_refs": ["可选的引用位置或页码"]
}
如果文中没有相关信息，请将 status 置为"不满足"，并指出缺失内容。`, req.ID, req.Title, req.Level, req.Description, req.Acceptance, strings.TrimSpace(response))
}

func buildBatchResponsePrompt(reqs []TenderRequirement, response string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("以下共 %d 条比选要求，请逐条判断供应商响应是否满足。\n\n", len(reqs)))

	for i, req := range reqs {
		sb.WriteString(fmt.Sprintf("--- 要求 %d ---\n", i+1))
		sb.WriteString(fmt.Sprintf("ID: %s\n标题: %s\n级别: %s\n", req.ID, req.Title, req.Level))
		sb.WriteString(fmt.Sprintf("要求描述: %s\n验收标准: %s\n\n", req.Description, req.Acceptance))
	}

	sb.WriteString(fmt.Sprintf("【响应文件内容】\n%s\n\n", strings.TrimSpace(response)))
	sb.WriteString(`请输出严格的 JSON 数组，每个元素对应一条要求，顺序与输入一致：
[
  {
    "status": "满足|部分满足|不满足",
    "confidence": 0.0-1.0,
    "evidence": "响应文件中的佐证",
    "reason": "判定依据解释",
    "gap": "未满足内容或差距，没有则留空",
    "advice": "整改建议，没有则留空",
    "response_refs": ["可选的引用位置或页码"]
  }
]
数组长度必须等于要求数量。如果文中没有相关信息，请将对应条目的 status 置为"不满足"。`)

	return sb.String()
}

type comparisonResponse struct {
	Status       string   `json:"status"`
	Confidence   float64  `json:"confidence"`
	Evidence     string   `json:"evidence"`
	Reason       string   `json:"reason"`
	Gap          string   `json:"gap"`
	Advice       string   `json:"advice"`
	ResponseRefs []string `json:"response_refs"`
}

func parseComparisonResponse(raw string) (*comparisonResponse, error) {
	trimmed := strings.TrimSpace(raw)
	seg := extractJSONSegment(trimmed)
	resp := &comparisonResponse{}
	if err := sonic.UnmarshalString(seg, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func parseBatchComparisonResponse(raw string, expectedCount int) ([]*comparisonResponse, error) {
	trimmed := strings.TrimSpace(raw)
	arrStr := extractJSONArraySegment(trimmed)
	if arrStr == "" {
		return nil, fmt.Errorf("未找到 JSON 数组")
	}

	var list []*comparisonResponse
	if err := sonic.UnmarshalString(arrStr, &list); err != nil {
		return nil, fmt.Errorf("解析比对 JSON 数组失败: %w", err)
	}

	if len(list) != expectedCount {
		return nil, fmt.Errorf("返回条目数 %d 与期望 %d 不一致", len(list), expectedCount)
	}

	return list, nil
}
