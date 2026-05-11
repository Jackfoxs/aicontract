package compliance

import (
	"backend/models"
	"context"
)

// SubmitJobInput 定义提交合规任务所需的字段
type SubmitJobInput struct {
	TenderFileID    uint64
	ResponseFileID  uint64
	NormSetID       uint64
	SelectedRuleIDs []uint64
	AIThreshold     float64
	CreatedBy       uint64
}

// ParsedDocument 表示解析后的文档结果
type ParsedDocument struct {
	Content  string
	Metadata map[string]interface{}
}

// RuleMatch 描述规则匹配过程中得到的候选片段
type RuleMatch struct {
	Rule            *models.DocumentChunk
	ResponseHit     bool
	ResponseExcerpt string
	TenderHit       bool
	TenderExcerpt   string
	ResponseContext string
	TenderContext   string
}

// IssueResult 是语义评估后的结果，用于落库
type IssueResult struct {
	Rule             *models.DocumentChunk
	Status           string
	Score            float64
	Remark           string
	ResponseExcerpt  string
	TenderExcerpt    string
	Advice           string
	RawLLMResponse   string
	LLMModel         string
	SourceType       string
	RequirementID    string
	RequirementName  string
	RequirementLevel string
	RequirementText  string
	Gap              string
	ResponseRefs     []string
}

// TenderRequirement 描述从比选文件解析出的要求
type TenderRequirement struct {
	ID          string
	Title       string
	Level       string
	Description string
	Acceptance  string
	Source      string
	Keywords    []string
}

// RuleLoader 定义获取规则内容的接口，便于扩展与测试
type RuleLoader interface {
	LoadRules(ctx context.Context, ids []uint64) ([]*models.DocumentChunk, error)
}
