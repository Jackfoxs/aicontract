package compliance

import "strings"

// SemanticEvaluator 使用简单的文本启发式进行语义判定
type SemanticEvaluator struct{}

// NewSemanticEvaluator 创建语义评估器
func NewSemanticEvaluator() *SemanticEvaluator {
	return &SemanticEvaluator{}
}

// Evaluate 根据匹配结果和阈值输出最终状态
func (e *SemanticEvaluator) Evaluate(matches []RuleMatch, responseText string, threshold float64) []IssueResult {
	results := make([]IssueResult, 0, len(matches))
	lowerResponse := strings.ToLower(responseText)

	for _, match := range matches {
		content := strings.ToLower(strings.TrimSpace(match.Rule.Content))
		score := 0.0
		status := "missing"
		remark := "未在响应文件中找到匹配内容"
		excerpt := match.ResponseExcerpt

		if content != "" && strings.Contains(lowerResponse, content) {
			score = 1.0
			status = "matched"
			remark = "响应文件包含规范内容"
			if excerpt == "" {
				// fallback excerpt
				idx := strings.Index(lowerResponse, content)
				excerpt = buildExcerpt(responseText, idx, len(content))
			}
		} else if match.TenderHit {
			status = "inconsistent"
			remark = "比选文件含有规范要求，但响应文件未覆盖"
		}

		if score < threshold {
			// 低于阈值按缺失处理
			if status == "matched" {
				status = "inconsistent"
				remark = "匹配置信度低于阈值"
			}
		}

		results = append(results, IssueResult{
			Rule:            match.Rule,
			Status:          status,
			Score:           score,
			Remark:          remark,
			ResponseExcerpt: excerpt,
			TenderExcerpt:   match.TenderExcerpt,
		})
	}
	return results
}
