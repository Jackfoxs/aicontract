package compliance

import (
	"context"
	"regexp"
	"strings"
	"unicode/utf8"

	"backend/models"

	"xorm.io/xorm"
)

// RuleMatcher 负责在解析后的文档中寻找规则候选
type RuleMatcher struct {
	db *xorm.Engine
}

// NewRuleMatcher 创建规则匹配器
func NewRuleMatcher(db *xorm.Engine) *RuleMatcher {
	return &RuleMatcher{db: db}
}

// LoadRules 根据 DocumentChunk ID 列表加载规范条目
func (m *RuleMatcher) LoadRules(ctx context.Context, ids []uint64) ([]*models.DocumentChunk, error) {
	if len(ids) == 0 {
		return []*models.DocumentChunk{}, nil
	}
	chunks := make([]*models.DocumentChunk, 0, len(ids))
	sess := m.db.Context(ctx).In("id", ids)
	if err := sess.Find(&chunks); err != nil {
		return nil, err
	}
	return chunks, nil
}

// MatchRules 在响应/比选文本中执行简单的文本匹配，返回候选结果
func (m *RuleMatcher) MatchRules(_ context.Context, rules []*models.DocumentChunk, tenderText, responseText string) []RuleMatch {
	matches := make([]RuleMatch, 0, len(rules))
	lowerTender := strings.ToLower(tenderText)
	lowerResponse := strings.ToLower(responseText)

	for _, rule := range rules {
		content := strings.TrimSpace(rule.Content)
		if content == "" {
			continue
		}
		normalized := strings.ToLower(content)

		match := RuleMatch{Rule: rule}
		if idx := strings.Index(lowerResponse, normalized); idx >= 0 {
			match.ResponseHit = true
			match.ResponseExcerpt = buildExcerpt(responseText, idx, len(content))
		}
		if idx := strings.Index(lowerTender, normalized); idx >= 0 {
			match.TenderHit = true
			match.TenderExcerpt = buildExcerpt(tenderText, idx, len(content))
		}
		if match.ResponseExcerpt == "" {
			match.ResponseExcerpt = findRelevantContext(responseText, content, rule.Title, 600)
		}
		if match.TenderExcerpt == "" {
			match.TenderExcerpt = findRelevantContext(tenderText, content, rule.Title, 600)
		}
		match.ResponseContext = match.ResponseExcerpt
		match.TenderContext = match.TenderExcerpt
		matches = append(matches, match)
	}
	return matches
}

func findRelevantContext(source, ruleContent, ruleTitle string, maxRunes int) string {
	trimmed := strings.TrimSpace(source)
	if trimmed == "" {
		return ""
	}
	paragraphs := splitParagraphs(trimmed)
	keywords := extractKeywords(ruleContent, ruleTitle)
	bestIdx := -1
	bestScore := 0
	for idx, para := range paragraphs {
		score := keywordScore(strings.ToLower(para), keywords)
		if score > bestScore {
			bestScore = score
			bestIdx = idx
		}
	}
	if bestIdx >= 0 && bestScore > 0 {
		return clipRunes(paragraphs[bestIdx], maxRunes)
	}
	// fallback: take leading part of document
	return clipRunes(paragraphs[0], maxRunes)
}

func splitParagraphs(text string) []string {
	parts := regexp.MustCompile(`\n{2,}`).Split(text, -1)
	res := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			res = append(res, trimmed)
		}
	}
	if len(res) == 0 {
		res = []string{strings.TrimSpace(text)}
	}
	return res
}

func extractKeywords(content, title string) []string {
	candidate := title + " " + content
	fields := strings.FieldsFunc(candidate, func(r rune) bool {
		switch r {
		case ' ', '\n', '\t', '，', '。', ',', '；', ';', '：', ':', '"', '\'', '（', '）', '(', ')', '《', '》', '[', ']', '{', '}', '、', '/', '\\':
			return true
		}
		return false
	})
	keywords := make([]string, 0, len(fields))
	for _, field := range fields {
		f := strings.ToLower(strings.TrimSpace(field))
		if utf8.RuneCountInString(f) >= 2 {
			keywords = append(keywords, f)
		}
	}
	if len(keywords) > 30 {
		keywords = keywords[:30]
	}
	return keywords
}

func keywordScore(paragraph string, keywords []string) int {
	if len(keywords) == 0 {
		return 0
	}
	score := 0
	for _, kw := range keywords {
		if kw == "" {
			continue
		}
		if strings.Contains(paragraph, kw) {
			score++
		}
	}
	return score
}

func clipRunes(text string, maxRunes int) string {
	if maxRunes <= 0 {
		return text
	}
	runes := []rune(text)
	if len(runes) <= maxRunes {
		return text
	}
	return string(runes[:maxRunes]) + "..."
}

func buildExcerpt(text string, startIdx, length int) string {
	if startIdx < 0 || startIdx >= len(text) {
		return ""
	}
	endIdx := startIdx + length
	if endIdx > len(text) {
		endIdx = len(text)
	}
	excerptStart := startIdx - 60
	if excerptStart < 0 {
		excerptStart = 0
	}
	excerptEnd := endIdx + 60
	if excerptEnd > len(text) {
		excerptEnd = len(text)
	}
	return strings.TrimSpace(text[excerptStart:excerptEnd])
}
