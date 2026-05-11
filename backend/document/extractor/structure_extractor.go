package extractor

import (
	"fmt"
	"regexp"
	"strings"

	"backend/models"
	"backend/utils"
)

// StructureExtractor 合同结构提取器
type StructureExtractor struct {
	sectionPatterns []string // 章节标题模式
	clausePatterns  []string // 条款编号模式
}

// NewStructureExtractor 创建结构提取器
func NewStructureExtractor() *StructureExtractor {
	return &StructureExtractor{
		// 章节标题模式（如：第一章、第二部分、一、1.等）
		sectionPatterns: []string{
			`^第[一二三四五六七八九十百]+章\s*(.+)`,
			`^第[一二三四五六七八九十百]+部分\s*(.+)`,
			`^第[一二三四五六七八九十百]+节\s*(.+)`,
			`^[一二三四五六七八九十]、\s*(.+)`,
			`^[（\(]?[一二三四五六七八九十][）\)]\s*(.+)`,
		},
		// 条款编号模式（如：第一条、1.1、(1)等）
		clausePatterns: []string{
			`^第[一二三四五六七八九十百千]+条\s*(.*)`,
			`^第[0-9]+条\s*(.*)`,
			`^[0-9]+\.\s*(.+)`,
			`^[0-9]+\.[0-9]+\s*(.+)`,
			`^[（\(][0-9]+[）\)]\s*(.+)`,
		},
	}
}

// ExtractStructure 从文本提取合同结构
func (e *StructureExtractor) ExtractStructure(content string) (*models.ContractStructure, error) {
	lines := strings.Split(content, "\n")

	structure := &models.ContractStructure{
		Sections: []models.ContractSection{},
		Clauses:  []models.ContractClause{},
	}

	var currentSection *models.ContractSection
	currentPos := 0

	for lineNum, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			currentPos += 1
			continue
		}

		// 尝试识别章节
		if section := e.tryExtractSection(line, lineNum, currentPos); section != nil {
			// 如果之前有章节，更新其结束位置
			if currentSection != nil {
				currentSection.EndPos = currentPos - 1
				structure.Sections = append(structure.Sections, *currentSection)
			}
			currentSection = section
			currentPos += len(line) + 1
			continue
		}

		// 尝试识别条款
		if clause := e.tryExtractClause(line); clause != nil {
			if currentSection != nil {
				clause.SectionID = currentSection.ID
			}
			structure.Clauses = append(structure.Clauses, *clause)
		}

		currentPos += len(line) + 1
	}

	// 添加最后一个章节
	if currentSection != nil {
		currentSection.EndPos = currentPos
		structure.Sections = append(structure.Sections, *currentSection)
	}

	return structure, nil
}

// tryExtractSection 尝试从行中提取章节
func (e *StructureExtractor) tryExtractSection(line string, lineNum, pos int) *models.ContractSection {
	for level, pattern := range e.sectionPatterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(line); len(matches) > 1 {
			title := line
			if matches[1] != "" {
				title = matches[1]
			}

			return &models.ContractSection{
				ID:       fmt.Sprintf("section_%d", lineNum),
				Title:    title,
				Level:    level + 1,
				Content:  "", // 后续填充
				StartPos: pos,
				EndPos:   0, // 后续更新
			}
		}
	}
	return nil
}

// tryExtractClause 尝试从行中提取条款
func (e *StructureExtractor) tryExtractClause(line string) *models.ContractClause {
	for _, pattern := range e.clausePatterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(line); len(matches) > 0 {
			// 提取条款编号
			clauseNumber := e.extractClauseNumber(line)

			// 提取条款内容
			content := line
			if len(matches) > 1 && matches[1] != "" {
				content = matches[1]
			}

			// 尝试识别条款类型
			clauseType := e.identifyClauseType(content)

			return &models.ContractClause{
				ID:        fmt.Sprintf("%d", utils.GenerateID()),
				Number:    clauseNumber,
				Title:     e.extractClauseTitle(content),
				Content:   content,
				Type:      clauseType,
				SectionID: "", // 由调用方填充
			}
		}
	}
	return nil
}

// extractClauseNumber 提取条款编号
func (e *StructureExtractor) extractClauseNumber(line string) string {
	// 匹配"第X条"
	if re := regexp.MustCompile(`^第[一二三四五六七八九十百千0-9]+条`); re.MatchString(line) {
		return re.FindString(line)
	}

	// 匹配"X."或"X.Y"
	if re := regexp.MustCompile(`^[0-9]+(\.[0-9]+)?`); re.MatchString(line) {
		return re.FindString(line)
	}

	// 匹配"(X)"
	if re := regexp.MustCompile(`^[（\(][0-9]+[）\)]`); re.MatchString(line) {
		return re.FindString(line)
	}

	return ""
}

// extractClauseTitle 提取条款标题（如果有）
func (e *StructureExtractor) extractClauseTitle(content string) string {
	// 尝试提取冒号或句号前的标题
	if idx := strings.Index(content, "："); idx > 0 && idx < 50 {
		return strings.TrimSpace(content[:idx])
	}
	if idx := strings.Index(content, ":"); idx > 0 && idx < 50 {
		return strings.TrimSpace(content[:idx])
	}

	// 如果内容太长，取前50个字符作为标题
	if len([]rune(content)) > 50 {
		return string([]rune(content)[:50]) + "..."
	}

	return content
}

// identifyClauseType 识别条款类型
func (e *StructureExtractor) identifyClauseType(content string) string {
	contentLower := strings.ToLower(content)

	// 甲乙方信息
	if strings.Contains(contentLower, "甲方") || strings.Contains(contentLower, "乙方") ||
		strings.Contains(contentLower, "买方") || strings.Contains(contentLower, "卖方") {
		return "parties"
	}

	// 金额相关
	if strings.Contains(contentLower, "金额") || strings.Contains(contentLower, "价格") ||
		strings.Contains(contentLower, "费用") || strings.Contains(contentLower, "元") {
		return "amount"
	}

	// 交付相关
	if strings.Contains(contentLower, "交付") || strings.Contains(contentLower, "交货") ||
		strings.Contains(contentLower, "验收") {
		return "delivery"
	}

	// 质保相关
	if strings.Contains(contentLower, "质保") || strings.Contains(contentLower, "保修") ||
		strings.Contains(contentLower, "warranty") {
		return "warranty"
	}

	// 支付相关
	if strings.Contains(contentLower, "支付") || strings.Contains(contentLower, "付款") {
		return "payment"
	}

	// 违约相关
	if strings.Contains(contentLower, "违约") || strings.Contains(contentLower, "责任") ||
		strings.Contains(contentLower, "赔偿") {
		return "liability"
	}

	// 争议解决
	if strings.Contains(contentLower, "争议") || strings.Contains(contentLower, "仲裁") ||
		strings.Contains(contentLower, "诉讼") {
		return "dispute"
	}

	return "general"
}

// ExtractSectionContent 提取章节内容
func (e *StructureExtractor) ExtractSectionContent(content string, section *models.ContractSection) string {
	if section.StartPos >= len(content) {
		return ""
	}

	endPos := section.EndPos
	if endPos > len(content) {
		endPos = len(content)
	}

	return content[section.StartPos:endPos]
}

// FindClauseByNumber 根据条款编号查找条款
func (e *StructureExtractor) FindClauseByNumber(structure *models.ContractStructure, number string) *models.ContractClause {
	for i := range structure.Clauses {
		if structure.Clauses[i].Number == number {
			return &structure.Clauses[i]
		}
	}
	return nil
}

// GetClausesByType 获取指定类型的所有条款
func (e *StructureExtractor) GetClausesByType(structure *models.ContractStructure, clauseType string) []models.ContractClause {
	var result []models.ContractClause
	for _, clause := range structure.Clauses {
		if clause.Type == clauseType {
			result = append(result, clause)
		}
	}
	return result
}
