package extractor

import (
	"fmt"
	"strings"

	"backend/models"
)

// TableExtractor 表格提取器
type TableExtractor struct {
	minColumns int // 最小列数
	minRows    int // 最小行数
}

// NewTableExtractor 创建表格提取器
func NewTableExtractor() *TableExtractor {
	return &TableExtractor{
		minColumns: 2,
		minRows:    2,
	}
}

// ExtractTables 从文本提取表格
// 注意：此实现为简化版本，实际应使用专门的表格识别库或PDF解析库
func (t *TableExtractor) ExtractTables(content string) ([]*models.ExtractedTable, error) {
	var tables []*models.ExtractedTable

	lines := strings.Split(content, "\n")
	var currentTable *models.ExtractedTable
	var tableLines []string
	inTable := false

	for i, line := range lines {
		line = strings.TrimSpace(line)

		// 检测表格开始（简单规则：包含多个分隔符）
		if t.looksLikeTableRow(line) {
			if !inTable {
				inTable = true
				tableLines = []string{}
				currentTable = &models.ExtractedTable{
					ID:         fmt.Sprintf("table_%d", len(tables)+1),
					PageNumber: 0, // 需要从文档解析器获取
					Headers:    []string{},
					Rows:       [][]string{},
					Type:       "unknown",
				}

				// 尝试从前一行获取表格标题
				if i > 0 {
					prevLine := strings.TrimSpace(lines[i-1])
					if t.looksLikeTableTitle(prevLine) {
						currentTable.Title = prevLine
					}
				}
			}
			tableLines = append(tableLines, line)
		} else if inTable {
			// 表格结束
			if len(tableLines) >= t.minRows {
				t.parseTableLines(currentTable, tableLines)
				if len(currentTable.Headers) >= t.minColumns {
					currentTable.Type = t.identifyTableType(currentTable)
					tables = append(tables, currentTable)
				}
			}
			inTable = false
			tableLines = []string{}
			currentTable = nil
		}
	}

	// 处理最后一个表格
	if inTable && len(tableLines) >= t.minRows && currentTable != nil {
		t.parseTableLines(currentTable, tableLines)
		if len(currentTable.Headers) >= t.minColumns {
			currentTable.Type = t.identifyTableType(currentTable)
			tables = append(tables, currentTable)
		}
	}

	return tables, nil
}

// looksLikeTableRow 判断是否像表格行
func (t *TableExtractor) looksLikeTableRow(line string) bool {
	// 包含制表符或多个空格分隔
	if strings.Count(line, "\t") >= t.minColumns-1 {
		return true
	}

	// 包含表格分隔符（|或│）
	if strings.Count(line, "|") >= t.minColumns-1 ||
		strings.Count(line, "│") >= t.minColumns-1 {
		return true
	}

	// 包含多个连续空格（可能是空格对齐的表格）
	if strings.Count(line, "  ") >= t.minColumns-1 {
		return true
	}

	return false
}

// looksLikeTableTitle 判断是否像表格标题
func (t *TableExtractor) looksLikeTableTitle(line string) bool {
	keywords := []string{"表", "清单", "明细", "参数", "配置", "规格"}
	for _, keyword := range keywords {
		if strings.Contains(line, keyword) && len(line) < 50 {
			return true
		}
	}
	return false
}

// parseTableLines 解析表格行
func (t *TableExtractor) parseTableLines(table *models.ExtractedTable, lines []string) {
	if len(lines) == 0 {
		return
	}

	// 尝试检测分隔符类型
	delimiter := t.detectDelimiter(lines[0])

	// 第一行作为表头
	table.Headers = t.splitTableRow(lines[0], delimiter)

	// 其他行作为数据
	for i := 1; i < len(lines); i++ {
		row := t.splitTableRow(lines[i], delimiter)
		if len(row) > 0 {
			table.Rows = append(table.Rows, row)
		}
	}

	// 原始文本可以后续扩展
	// table.RawText = strings.Join(lines, "\n")
}

// detectDelimiter 检测表格分隔符
func (t *TableExtractor) detectDelimiter(line string) string {
	if strings.Contains(line, "\t") {
		return "\t"
	}
	if strings.Contains(line, "|") {
		return "|"
	}
	if strings.Contains(line, "│") {
		return "│"
	}
	// 默认使用多空格分隔
	return "  "
}

// splitTableRow 分割表格行
func (t *TableExtractor) splitTableRow(line string, delimiter string) []string {
	var cells []string

	if delimiter == "  " {
		// 按多个空格分割
		parts := strings.Fields(line)
		for _, part := range parts {
			if part != "" {
				cells = append(cells, strings.TrimSpace(part))
			}
		}
	} else {
		// 按指定分隔符分割
		parts := strings.Split(line, delimiter)
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" && part != "|" && part != "│" {
				cells = append(cells, part)
			}
		}
	}

	return cells
}

// identifyTableType 识别表格类型
func (t *TableExtractor) identifyTableType(table *models.ExtractedTable) string {
	titleLower := strings.ToLower(table.Title)

	// 技术参数表
	if strings.Contains(titleLower, "参数") || strings.Contains(titleLower, "技术") ||
		strings.Contains(titleLower, "规格") || strings.Contains(titleLower, "配置") {
		return "tech_params"
	}

	// 价格明细表
	if strings.Contains(titleLower, "价格") || strings.Contains(titleLower, "金额") ||
		strings.Contains(titleLower, "费用") || strings.Contains(titleLower, "报价") {
		return "price_breakdown"
	}

	// 设备清单
	if strings.Contains(titleLower, "清单") || strings.Contains(titleLower, "列表") ||
		strings.Contains(titleLower, "设备") {
		return "device_list"
	}

	// 交付计划
	if strings.Contains(titleLower, "交付") || strings.Contains(titleLower, "进度") ||
		strings.Contains(titleLower, "计划") {
		return "delivery_schedule"
	}

	// 检查表头
	headersStr := strings.Join(table.Headers, " ")
	headersLower := strings.ToLower(headersStr)

	if strings.Contains(headersLower, "参数") || strings.Contains(headersLower, "parameter") {
		return "tech_params"
	}
	if strings.Contains(headersLower, "金额") || strings.Contains(headersLower, "价格") {
		return "price_breakdown"
	}

	return "general"
}

// ExtractTableByTitle 根据标题查找表格
func (t *TableExtractor) ExtractTableByTitle(tables []*models.ExtractedTable, titleKeyword string) *models.ExtractedTable {
	for _, table := range tables {
		if strings.Contains(table.Title, titleKeyword) {
			return table
		}
	}
	return nil
}

// ExtractTablesByType 获取指定类型的所有表格
func (t *TableExtractor) ExtractTablesByType(tables []*models.ExtractedTable, tableType string) []*models.ExtractedTable {
	var result []*models.ExtractedTable
	for _, table := range tables {
		if table.Type == tableType {
			result = append(result, table)
		}
	}
	return result
}

// ConvertTableToMap 将表格转换为键值对映射（适用于双列参数表）
func (t *TableExtractor) ConvertTableToMap(table *models.ExtractedTable) map[string]string {
	result := make(map[string]string)

	if len(table.Headers) < 2 {
		return result
	}

	for _, row := range table.Rows {
		if len(row) >= 2 {
			key := strings.TrimSpace(row[0])
			value := strings.TrimSpace(row[1])
			if key != "" {
				result[key] = value
			}
		}
	}

	return result
}
