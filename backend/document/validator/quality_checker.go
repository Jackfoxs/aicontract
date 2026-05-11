package validator

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// QualityChecker 文档质量检测器
type QualityChecker struct{}

// NewQualityChecker 创建质量检测器
func NewQualityChecker() *QualityChecker {
	return &QualityChecker{}
}

// QualityReport 质量检测报告
type QualityReport struct {
	// Score 综合质量评分 (0.0-1.0)
	Score float64 `json:"score"`

	// Issues 质量问题列表
	Issues []QualityIssue `json:"issues"`

	// Metrics 质量指标
	Metrics QualityMetrics `json:"metrics"`

	// Passed 是否通过质量检测
	Passed bool `json:"passed"`

	// Recommendation 建议
	Recommendation string `json:"recommendation"`
}

// QualityIssue 质量问题
type QualityIssue struct {
	// Type 问题类型
	Type IssueType `json:"type"`

	// Severity 严重程度 (low, medium, high)
	Severity string `json:"severity"`

	// Description 问题描述
	Description string `json:"description"`

	// Impact 对质量分数的影响
	Impact float64 `json:"impact"`
}

// IssueType 问题类型
type IssueType string

const (
	IssueTypeEmpty           IssueType = "empty_content"     // 内容为空
	IssueTypeGarbled         IssueType = "garbled_text"      // 乱码
	IssueTypeLowDensity      IssueType = "low_text_density"  // 文本密度低
	IssueTypeStructurePoor   IssueType = "poor_structure"    // 结构缺失
	IssueTypeIncompleteness  IssueType = "incompleteness"    // 内容不完整
	IssueTypeSpecialChars    IssueType = "excessive_special" // 特殊字符过多
	IssueTypeNoPunctuation   IssueType = "no_punctuation"    // 缺少标点
	IssueTypeExcessiveSpaces IssueType = "excessive_spaces"  // 空格过多
)

// QualityMetrics 质量指标
type QualityMetrics struct {
	// TotalChars 总字符数
	TotalChars int `json:"total_chars"`

	// TotalWords 总词数
	TotalWords int `json:"total_words"`

	// TotalLines 总行数
	TotalLines int `json:"total_lines"`

	// AvgCharsPerLine 平均每行字符数
	AvgCharsPerLine float64 `json:"avg_chars_per_line"`

	// GarbledRatio 乱码比例
	GarbledRatio float64 `json:"garbled_ratio"`

	// PunctuationRatio 标点符号比例
	PunctuationRatio float64 `json:"punctuation_ratio"`

	// SpecialCharRatio 特殊字符比例
	SpecialCharRatio float64 `json:"special_char_ratio"`

	// SpaceRatio 空格比例
	SpaceRatio float64 `json:"space_ratio"`

	// ChineseRatio 中文字符比例
	ChineseRatio float64 `json:"chinese_ratio"`

	// EnglishRatio 英文字符比例
	EnglishRatio float64 `json:"english_ratio"`

	// NumberRatio 数字比例
	NumberRatio float64 `json:"number_ratio"`
}

// Check 执行质量检测
func (qc *QualityChecker) Check(content string) *QualityReport {
	// 计算指标
	metrics := qc.calculateMetrics(content)

	// 检测问题
	issues := []QualityIssue{}

	// 1. 检查内容是否为空
	if strings.TrimSpace(content) == "" {
		issues = append(issues, QualityIssue{
			Type:        IssueTypeEmpty,
			Severity:    "high",
			Description: "文档内容为空",
			Impact:      1.0,
		})
	}

	// 2. 检查乱码
	if metrics.GarbledRatio > 0.1 {
		issues = append(issues, QualityIssue{
			Type:        IssueTypeGarbled,
			Severity:    "high",
			Description: "文档包含大量乱码字符",
			Impact:      0.5,
		})
	} else if metrics.GarbledRatio > 0.05 {
		issues = append(issues, QualityIssue{
			Type:        IssueTypeGarbled,
			Severity:    "medium",
			Description: "文档包含少量乱码字符",
			Impact:      0.2,
		})
	}

	// 3. 检查文本密度
	if metrics.AvgCharsPerLine < 20 {
		issues = append(issues, QualityIssue{
			Type:        IssueTypeLowDensity,
			Severity:    "medium",
			Description: "平均每行字符数过少，可能是扫描版PDF或解析不完整",
			Impact:      0.3,
		})
	}

	// 4. 检查标点符号
	if metrics.PunctuationRatio < 0.01 && metrics.TotalChars > 100 {
		issues = append(issues, QualityIssue{
			Type:        IssueTypeNoPunctuation,
			Severity:    "medium",
			Description: "文档缺少标点符号，可能解析不完整",
			Impact:      0.2,
		})
	}

	// 5. 检查特殊字符
	if metrics.SpecialCharRatio > 0.3 {
		issues = append(issues, QualityIssue{
			Type:        IssueTypeSpecialChars,
			Severity:    "low",
			Description: "文档包含过多特殊字符",
			Impact:      0.1,
		})
	}

	// 6. 检查空格
	if metrics.SpaceRatio > 0.5 {
		issues = append(issues, QualityIssue{
			Type:        IssueTypeExcessiveSpaces,
			Severity:    "low",
			Description: "文档包含过多空格",
			Impact:      0.05,
		})
	}

	// 7. 检查结构完整性
	if !qc.hasBasicStructure(content) {
		issues = append(issues, QualityIssue{
			Type:        IssueTypeStructurePoor,
			Severity:    "medium",
			Description: "文档缺少基本结构（标题、段落等）",
			Impact:      0.15,
		})
	}

	// 8. 检查内容完整性（启发式）
	if qc.seemsIncomplete(content) {
		issues = append(issues, QualityIssue{
			Type:        IssueTypeIncompleteness,
			Severity:    "medium",
			Description: "文档内容可能不完整（结尾异常）",
			Impact:      0.15,
		})
	}

	// 计算总分
	score := 1.0
	for _, issue := range issues {
		score -= issue.Impact
	}
	if score < 0 {
		score = 0
	}

	// 生成建议
	recommendation := qc.generateRecommendation(score, issues)

	return &QualityReport{
		Score:          score,
		Issues:         issues,
		Metrics:        metrics,
		Passed:         score >= 0.7, // 阈值：0.7
		Recommendation: recommendation,
	}
}

// calculateMetrics 计算质量指标
func (qc *QualityChecker) calculateMetrics(content string) QualityMetrics {
	totalChars := len(content)
	if totalChars == 0 {
		return QualityMetrics{}
	}

	lines := strings.Split(content, "\n")
	totalLines := len(lines)

	// 统计各类字符
	garbledCount := 0
	punctCount := 0
	specialCharCount := 0
	spaceCount := 0
	chineseCount := 0
	englishCount := 0
	numberCount := 0

	for _, r := range content {
		// 乱码
		if r == '�' || r == '\ufffd' {
			garbledCount++
		}

		// 标点符号（中英文）
		if qc.isPunctuation(r) {
			punctCount++
		}

		// 特殊字符
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.IsSpace(r) {
			specialCharCount++
		}

		// 空格
		if unicode.IsSpace(r) {
			spaceCount++
		}

		// 中文
		if qc.isChinese(r) {
			chineseCount++
		}

		// 英文
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			englishCount++
		}

		// 数字
		if unicode.IsDigit(r) {
			numberCount++
		}
	}

	// 计算词数（简单按空格分割）
	words := strings.Fields(content)
	totalWords := len(words)

	return QualityMetrics{
		TotalChars:       totalChars,
		TotalWords:       totalWords,
		TotalLines:       totalLines,
		AvgCharsPerLine:  float64(totalChars) / float64(totalLines),
		GarbledRatio:     float64(garbledCount) / float64(totalChars),
		PunctuationRatio: float64(punctCount) / float64(totalChars),
		SpecialCharRatio: float64(specialCharCount) / float64(totalChars),
		SpaceRatio:       float64(spaceCount) / float64(totalChars),
		ChineseRatio:     float64(chineseCount) / float64(totalChars),
		EnglishRatio:     float64(englishCount) / float64(totalChars),
		NumberRatio:      float64(numberCount) / float64(totalChars),
	}
}

// hasBasicStructure 检查是否有基本结构
func (qc *QualityChecker) hasBasicStructure(content string) bool {
	// 启发式：至少有3个段落分隔（2个连续换行）
	paragraphs := regexp.MustCompile(`\n\s*\n`).Split(content, -1)
	if len(paragraphs) < 3 {
		return false
	}

	// 至少有一些标题特征（如"第X条"、"一、"、数字编号等）
	titlePatterns := []string{
		`第[一二三四五六七八九十百千万\d]+条`,
		`第[一二三四五六七八九十百千万\d]+章`,
		`[一二三四五六七八九十]、`,
		`^\d+\.`,
		`^\d+、`,
	}

	for _, pattern := range titlePatterns {
		if matched, _ := regexp.MatchString(pattern, content); matched {
			return true
		}
	}

	return false
}

// seemsIncomplete 检查内容是否看起来不完整
func (qc *QualityChecker) seemsIncomplete(content string) bool {
	trimmed := strings.TrimSpace(content)
	if len(trimmed) == 0 {
		return true
	}

	// 检查最后100个字符
	lastPart := trimmed
	if len(trimmed) > 100 {
		lastPart = trimmed[len(trimmed)-100:]
	}

	// 启发式：结尾没有标点、突然中断
	lastChar, _ := utf8.DecodeLastRuneInString(lastPart)
	if !qc.isPunctuation(lastChar) && lastChar != '\n' {
		// 如果最后不是标点或换行，可能不完整
		return true
	}

	return false
}

// isPunctuation 判断是否是标点符号
func (qc *QualityChecker) isPunctuation(r rune) bool {
	punctuations := `。，、；：？！""''（）【】《》…—·,.;:?!"'()[]<>-`
	return strings.ContainsRune(punctuations, r)
}

// isChinese 判断是否是中文字符
func (qc *QualityChecker) isChinese(r rune) bool {
	return r >= 0x4e00 && r <= 0x9fa5
}

// generateRecommendation 生成建议
func (qc *QualityChecker) generateRecommendation(score float64, issues []QualityIssue) string {
	if score >= 0.9 {
		return "文档质量良好，可以直接使用"
	}

	if score >= 0.7 {
		return "文档质量尚可，建议检查解析结果"
	}

	if score >= 0.5 {
		// 找出最严重的问题
		var criticalIssue *QualityIssue
		for i := range issues {
			if issues[i].Severity == "high" {
				criticalIssue = &issues[i]
				break
			}
		}

		if criticalIssue != nil {
			switch criticalIssue.Type {
			case IssueTypeGarbled:
				return "文档包含大量乱码，建议使用LLM视觉解析或重新扫描"
			case IssueTypeLowDensity:
				return "文本密度过低，建议使用LLM视觉解析"
			default:
				return "文档质量较差，建议检查原文件或使用备用解析方法"
			}
		}

		return "文档质量一般，建议人工复核关键内容"
	}

	return "文档质量很差，强烈建议使用LLM视觉解析或更换文档"
}
