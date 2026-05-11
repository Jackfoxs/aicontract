package compliance

import (
	"context"
	"strings"

	"backend/models"

	"github.com/bytedance/sonic"
	"xorm.io/xorm"
)

// HighlightPair 记录单个规则对应的响应/比选高亮ID
type HighlightPair struct {
	ResponseID uint64
	TenderID   uint64
}

// HighlightBuilder 负责将匹配结果转换为高亮信息并写入数据库
type HighlightBuilder struct {
	db *xorm.Engine
}

// NewHighlightBuilder 创建高亮构建器
func NewHighlightBuilder(db *xorm.Engine) *HighlightBuilder {
	return &HighlightBuilder{db: db}
}

// CreateHighlights 根据 LLM 结果生成高亮记录
func (b *HighlightBuilder) CreateHighlights(ctx context.Context, jobID uint64, issues []IssueResult, tenderText, responseText string) (map[uint64]HighlightPair, error) {
	pairs := make(map[uint64]HighlightPair)
	if len(issues) == 0 {
		return pairs, nil
	}

	for _, issue := range issues {
		if issue.Rule == nil {
			continue
		}
		pair := HighlightPair{}
		responseTextTrim := strings.TrimSpace(issue.ResponseExcerpt)
		if responseTextTrim != "" {
			start, end := locateFragment(responseText, responseTextTrim)
			highlight := &models.ComplianceHighlight{
				JobID:       jobID,
				FileRole:    models.ComplianceFileRoleResponse,
				Page:        0,
				OffsetStart: start,
				OffsetEnd:   end,
				Text:        responseTextTrim,
			}
			if err := highlight.SetBBoxes(nil); err != nil {
				return nil, err
			}
			if _, err := b.db.Context(ctx).Insert(highlight); err != nil {
				return nil, err
			}
			pair.ResponseID = highlight.ID
		} else if issue.Status != "matched" {
			missingText := strings.TrimSpace(issue.Rule.Content)
			if missingText == "" {
				missingText = strings.TrimSpace(issue.Remark)
			}
			highlight := &models.ComplianceHighlight{
				JobID:       jobID,
				FileRole:    models.ComplianceFileRoleResponse,
				Page:        0,
				OffsetStart: -1,
				OffsetEnd:   -1,
				Text:        formatMissingText(missingText),
			}
			if err := highlight.SetBBoxes(nil); err != nil {
				return nil, err
			}
			if _, err := b.db.Context(ctx).Insert(highlight); err != nil {
				return nil, err
			}
			pair.ResponseID = highlight.ID
		}

		tenderExcerpt := strings.TrimSpace(issue.TenderExcerpt)
		if tenderExcerpt == "" {
			tenderExcerpt = strings.TrimSpace(issue.Rule.Content)
		}
		if tenderExcerpt != "" {
			start, end := locateFragment(tenderText, tenderExcerpt)
			highlight := &models.ComplianceHighlight{
				JobID:       jobID,
				FileRole:    models.ComplianceFileRoleTender,
				Page:        0,
				OffsetStart: start,
				OffsetEnd:   end,
				Text:        tenderExcerpt,
			}
			if err := highlight.SetBBoxes(nil); err != nil {
				return nil, err
			}
			if _, err := b.db.Context(ctx).Insert(highlight); err != nil {
				return nil, err
			}
			pair.TenderID = highlight.ID
		}
		if pair.ResponseID != 0 || pair.TenderID != 0 {
			pairs[issue.Rule.ID] = pair
		}
	}
	return pairs, nil
}

func locateFragment(text, fragment string) (int, int) {
	if text == "" || fragment == "" {
		return -1, -1
	}
	lowerText := strings.ToLower(text)
	lowerFrag := strings.ToLower(strings.TrimSpace(fragment))
	idx := strings.Index(lowerText, lowerFrag)
	if idx < 0 {
		return -1, -1
	}
	return idx, idx + len(lowerFrag)
}

func extractExcerpt(fullText string, start, end int, ruleContent, fallback string) string {
	if start >= 0 && end >= 0 && end > start && end <= len(fullText) {
		return strings.TrimSpace(fullText[start:end])
	}
	if fallback != "" {
		return strings.TrimSpace(fallback)
	}
	return strings.TrimSpace(ruleContent)
}

func formatMissingText(text string) string {
	content := strings.TrimSpace(text)
	if content == "" {
		return "未匹配到响应内容"
	}
	const limit = 200
	runes := []rune(content)
	if len(runes) > limit {
		content = string(runes[:limit]) + "..."
	}
	return "未匹配到响应内容：" + content
}

// EncodeHighlightRef 将高亮ID对编码为 JSON 字符串
func EncodeHighlightRef(pair HighlightPair) (string, error) {
	if pair.ResponseID == 0 && pair.TenderID == 0 {
		return "", nil
	}
	payload := map[string]uint64{}
	if pair.ResponseID != 0 {
		payload["response_id"] = pair.ResponseID
	}
	if pair.TenderID != 0 {
		payload["tender_id"] = pair.TenderID
	}
	data, err := sonic.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
