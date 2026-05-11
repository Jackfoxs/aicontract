package compliance

import (
	"context"
	"fmt"
	"strings"

	"backend/global"

	"github.com/bytedance/sonic"
)

// TenderRequirementExtractor 使用 LLM 从比选文件中抽取要求
type TenderRequirementExtractor struct {
	client ChatClient
}

func NewTenderRequirementExtractor(client ChatClient) *TenderRequirementExtractor {
	return &TenderRequirementExtractor{client: client}
}

func (e *TenderRequirementExtractor) Extract(ctx context.Context, content string) ([]TenderRequirement, float64, error) {
	if e == nil || e.client == nil {
		return nil, 0, fmt.Errorf("requirement extractor not initialized")
	}
	prompt := buildRequirementPrompt(content)
	if global.Log != nil {
		global.Log.Info("比选要求抽取开始", "prompt_len", len([]rune(prompt)))
	}
	resp, cost, err := e.client.ChatWithConfig(ctx, requirementSystemPrompt(), prompt, 0.1, 4096)
	if err != nil {
		return nil, cost, fmt.Errorf("抽取比选要求失败: %w", err)
	}
	reqs, err := parseRequirementResponse(resp)
	if err != nil {
		return nil, cost, fmt.Errorf("解析比选要求失败: %w", err)
	}
	return reqs, cost, nil
}

func requirementSystemPrompt() string {
	return "你是一名政府采购评标专家，需要从比选文件中梳理全部采购要求，并输出结构化 JSON。"
}

func buildRequirementPrompt(content string) string {
	trimmed := strings.TrimSpace(content)
	return fmt.Sprintf(`请阅读以下比选文件内容，梳理所有强制及可选要求，输出 JSON：
{
  "requirements": [
    {
      "id": "唯一ID，形如 REQ-001",
      "title": "条款或章节标题",
      "level": "mandatory 或 optional",
      "description": "完整要求内容",
      "acceptance": "验收或达标标准，如无可留空",
      "source": "引用的原文句子",
      "keywords": ["可选的关键词数组"]
    }
  ]
}
禁止遗漏任何强制条款，确保描述清晰。

比选文件内容：
%s`, trimmed)
}

func parseRequirementResponse(raw string) ([]TenderRequirement, error) {
	type response struct {
		Requirements []struct {
			ID          string   `json:"id"`
			Title       string   `json:"title"`
			Level       string   `json:"level"`
			Description string   `json:"description"`
			Acceptance  string   `json:"acceptance"`
			Source      string   `json:"source"`
			Keywords    []string `json:"keywords"`
		} `json:"requirements"`
	}
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return []TenderRequirement{}, nil
	}
	var resp response
	if err := sonic.UnmarshalString(extractJSONSegment(trimmed), &resp); err != nil {
		return nil, err
	}
	reqs := make([]TenderRequirement, 0, len(resp.Requirements))
	for idx, item := range resp.Requirements {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			id = fmt.Sprintf("REQ-%03d", idx+1)
		}
		reqs = append(reqs, TenderRequirement{
			ID:          id,
			Title:       strings.TrimSpace(item.Title),
			Level:       strings.ToLower(strings.TrimSpace(item.Level)),
			Description: strings.TrimSpace(item.Description),
			Acceptance:  strings.TrimSpace(item.Acceptance),
			Source:      strings.TrimSpace(item.Source),
			Keywords:    item.Keywords,
		})
	}
	return reqs, nil
}
