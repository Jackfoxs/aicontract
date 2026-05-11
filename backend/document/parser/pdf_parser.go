package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/cloudwego/eino/schema"
	pdflib "github.com/dslipak/pdf"
)

// PDFParser PDF解析器
type PDFParser struct {
	contentBuilder strings.Builder
}

// NewPDFParser 创建PDF解析器
func NewPDFParser() *PDFParser {
	return &PDFParser{}
}

// Parse 解析PDF文件并返回提取的文本内容
func (p *PDFParser) Parse(filePath string) (string, error) {
	reader, err := pdflib.Open(filePath)
	if err != nil {
		return "", &ParserError{
			Code:    "PDF_PARSE_ERROR",
			Message: "解析PDF文件失败",
			Err:     err,
		}
	}

	p.contentBuilder.Reset()
	numPages := reader.NumPage()

	for i := 1; i <= numPages; i++ {
		page := reader.Page(i)
		content := page.Content()

		// 提取文本内容
		for _, t := range content.Text {
			s := normalizeUTF8(t.S)
			p.contentBuilder.WriteString(s)
		}
		p.contentBuilder.WriteString("\n\n")
	}

	result := ensureParagraphBreaks(ensureHeaderNewlines(p.contentBuilder.String()))
	if strings.TrimSpace(result) == "" {
		return "", &ParserError{
			Code:    "EMPTY_CONTENT",
			Message: "PDF文件解析结果为空",
		}
	}

	return result, nil
}

// ParseToDocuments 解析PDF文件并返回文档对象数组（按页分块）
func (p *PDFParser) ParseToDocuments(filePath string) ([]*schema.Document, error) {
	reader, err := pdflib.Open(filePath)
	if err != nil {
		return nil, &ParserError{
			Code:    "PDF_PARSE_ERROR",
			Message: "解析PDF文件失败",
			Err:     err,
		}
	}

	docs := []*schema.Document{}
	numPages := reader.NumPage()

	for i := 1; i <= numPages; i++ {
		page := reader.Page(i)
		content := page.Content()

		// 提取文本内容
		var textContent strings.Builder
		for _, t := range content.Text {
			textContent.WriteString(normalizeUTF8(t.S))
		}

		pageText := ensureParagraphBreaks(ensureHeaderNewlines(textContent.String()))
		if strings.TrimSpace(pageText) == "" {
			continue // 跳过空页
		}

		doc := &schema.Document{
			ID:      fmt.Sprintf("page_%d", i),
			Content: pageText,
			MetaData: map[string]interface{}{
				"page":      i,
				"file_type": "pdf",
			},
		}
		docs = append(docs, doc)
	}

	if len(docs) == 0 {
		return nil, &ParserError{
			Code:    "EMPTY_CONTENT",
			Message: "PDF文件解析结果为空",
		}
	}

	return docs, nil
}

// ParseWithMetadata 解析PDF并返回内容和元数据
func (p *PDFParser) ParseWithMetadata(filePath string) (*ParseResult, error) {
	// 获取文件信息
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, &ParserError{
			Code:    "FILE_NOT_FOUND",
			Message: "文件不存在",
			Err:     err,
		}
	}

	reader, err := pdflib.Open(filePath)
	if err != nil {
		return nil, &ParserError{
			Code:    "PDF_PARSE_ERROR",
			Message: "解析PDF文件失败",
			Err:     err,
		}
	}

	// 解析文档
	docs := []*schema.Document{}
	p.contentBuilder.Reset()
	numPages := reader.NumPage()
	tableCount := 0

	for i := 1; i <= numPages; i++ {
		page := reader.Page(i)
		content := page.Content()

		// 提取文本内容
		var textContent strings.Builder
		for _, t := range content.Text {
			textContent.WriteString(normalizeUTF8(t.S))
		}

		pageText := textContent.String()
		p.contentBuilder.WriteString(pageText)
		p.contentBuilder.WriteString("\n\n")

		// 简单的表格检测（启发式：多个连续的制表符或特定格式）
		if strings.Contains(pageText, "\t\t") || strings.Contains(pageText, "│") {
			tableCount++
		}

		if strings.TrimSpace(pageText) != "" {
			doc := &schema.Document{
				ID:      fmt.Sprintf("page_%d", i),
				Content: pageText,
				MetaData: map[string]interface{}{
					"page":      i,
					"file_type": "pdf",
				},
			}
			docs = append(docs, doc)
		}
	}

	fullContent := ensureParagraphBreaks(ensureHeaderNewlines(normalizeUTF8(p.contentBuilder.String())))
	textLength := len(fullContent)

	// 计算质量评分
	qualityScore := p.calculateQualityScore(fullContent, numPages)

	result := &ParseResult{
		Content:   fullContent,
		Documents: docs,
		Metadata: &DocumentMetadata{
			FileName:   filepath.Base(filePath),
			FileType:   "pdf",
			FileSize:   fileInfo.Size(),
			PageCount:  numPages,
			TextLength: textLength,
			TableCount: tableCount,
			HasImages:  false, // dslipak/pdf库暂不支持图片检测
		},
		QualityScore: qualityScore,
		ParseMethod:  "native_pdf",
		Tables:       []*Table{}, // TODO: 实现表格提取
	}

	return result, nil
}

// GetSupportedFormats 获取支持的文件格式
func (p *PDFParser) GetSupportedFormats() []string {
	return []string{"pdf", ".pdf"}
}

// calculateQualityScore 计算文档质量评分
// 基于：文本长度、页面覆盖率、乱码检测
func (p *PDFParser) calculateQualityScore(content string, pageCount int) float64 {
	if content == "" {
		return 0.0
	}

	score := 1.0

	// 1. 检查平均每页文本量（低于50字扣分）
	avgCharsPerPage := float64(len(content)) / float64(pageCount)
	if avgCharsPerPage < 50 {
		score -= 0.3
	} else if avgCharsPerPage < 100 {
		score -= 0.1
	}

	// 2. 检查乱码（连续的特殊字符或乱码模式）
	garbledCount := 0
	for _, char := range content {
		if char == '�' || char == '\ufffd' {
			garbledCount++
		}
	}
	garbledRatio := float64(garbledCount) / float64(len(content))
	if garbledRatio > 0.1 {
		score -= 0.4
	} else if garbledRatio > 0.05 {
		score -= 0.2
	}

	// 3. 检查文本连贯性（空格和标点占比）
	spaceCount := strings.Count(content, " ")
	punctCount := strings.Count(content, "。") + strings.Count(content, "，") +
		strings.Count(content, ".") + strings.Count(content, ",")

	if spaceCount+punctCount < len(content)/20 {
		score -= 0.1 // 文本可能缺乏结构
	}

	// 确保分数在 0.0-1.0 之间
	if score < 0.0 {
		score = 0.0
	}

	return score
}

// normalizeUTF8 将输入规范化为有效UTF-8字符串，去除非法序列
func normalizeUTF8(s string) string {
	if s == "" {
		return s
	}
	if utf8.ValidString(s) {
		// 进一步压缩不可见控制字符（保留常用换行、制表）
		s = strings.Map(func(r rune) rune {
			if r < 32 && r != '\n' && r != '\t' && r != '\r' {
				return -1
			}
			return r
		}, s)
		return s
	}
	// 将非法UTF-8替换为可见空字符，避免'�'蔓延
	cleaned := strings.ToValidUTF8(s, " ")
	cleaned = strings.Map(func(r rune) rune {
		if r < 32 && r != '\n' && r != '\t' && r != '\r' {
			return -1
		}
		return r
	}, cleaned)
	return cleaned
}

// ensureHeaderNewlines 在“第X章/第X条”前强制插入换行，便于后续按章/条切分
func ensureHeaderNewlines(s string) string {
	if s == "" {
		return s
	}
	// 清理页眉页脚中的“—4—/－4－/ - 4 -”等页码装饰
	s = cleanupPageNumberArtifacts(s)
	// 在不以换行开始的“第…章|第…条”前插入换行（使用捕获分组替换，避免使用不被RE2支持的正向预查）
	re := regexp.MustCompile(`(?m)([^\n])(第[一二三四五六七八九十百千零〇0-9]+(章|条))`)
	s = re.ReplaceAllString(s, "$1\n$2")
	// 合并多余空行
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = regexp.MustCompile(`\n{3,}`).ReplaceAllString(s, "\n\n")
	return s
}

// cleanupPageNumberArtifacts 移除常见PDF页眉页脚页码花体（—4—、－12－、- 3 - 等）
func cleanupPageNumberArtifacts(s string) string {
	if s == "" {
		return s
	}
	// 行整行是页码花体：删除整行
	linePattern := regexp.MustCompile(`(?m)^\s*[—–−‑‒―－-]+\s*\d+\s*[—–−‑‒―－-]+\s*$`)
	s = linePattern.ReplaceAllString(s, "")
	// 行内出现的页码花体：去掉并以单个空格替代
	inlinePattern := regexp.MustCompile(`[—–−‑‒―－-]+\s*\d+\s*[—–−‑‒―－-]+`)
	s = inlinePattern.ReplaceAllString(s, " ")
	return s
}

// ensureParagraphBreaks 尝试在中文段落缩进“　　”前加换行，保留自然段
func ensureParagraphBreaks(s string) string {
	if s == "" {
		return s
	}
	// 在句号/分号/问号/叹号后，遇到中文缩进“　　”时断行
	s = regexp.MustCompile(`([。；；;！？!?])\s*　　`).ReplaceAllString(s, "$1\n　　")
	// 对“第五条\s+”后如果不是换行，补一个换行（防止整条挤在同一行影响阅读与分段）
	s = regexp.MustCompile(`(?m)(第[一二三四五六七八九十百千零〇0-9]+条)\s+`).ReplaceAllString(s, "$1\n")
	// 合并3个以上空行为2个
	s = regexp.MustCompile(`\n{3,}`).ReplaceAllString(s, "\n\n")
	return s
}
