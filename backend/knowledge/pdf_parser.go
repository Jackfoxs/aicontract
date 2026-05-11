package knowledge

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/cloudwego/eino/schema"
	pdflib "github.com/dslipak/pdf" // 重命名导入以避免冲突
)

// PDFParser PDF解析器，用于解析PDF文件并提取内容
type PDFParser struct {
	contentBuilder strings.Builder
}

// NewPDFParser 创建一个新的PDF解析器
func NewPDFParser() *PDFParser {
	return &PDFParser{}
}

// Parse 解析PDF文件并返回提取的文本内容
func (p *PDFParser) Parse(filePath string) (string, error) {
	// 打开PDF文件
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("无法打开PDF文件: %w", err)
	}
	defer file.Close()

	// 使用dslipak/pdf库解析PDF文件
	reader, err := pdflib.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("解析PDF文件失败: %w", err)
	}
	// dslipak/pdf库不需要显式关闭

	// 创建文档对象数组
	docs := []*schema.Document{}

	// 提取PDF内容
	numPages := reader.NumPage()

	// 重置内容构建器
	p.contentBuilder.Reset()

	for i := 1; i <= numPages; i++ {
		page := reader.Page(i)
		content := page.Content()

		// 将Text切片转换为字符串
		var textContent string
		for _, t := range content.Text {
			textContent += normalizeUTF8(t.S)
		}

		// 添加页面内容（插入章/条换行与段落换行）
		p.contentBuilder.WriteString(ensureParagraphBreaks(ensureHeaderNewlines(normalizeUTF8(textContent))))
		p.contentBuilder.WriteString("\n\n")

		// 创建文档对象
		doc := &schema.Document{
			ID:      fmt.Sprintf("page_%d", i),
			Content: textContent,
			MetaData: map[string]interface{}{
				"page": i,
			},
		}
		docs = append(docs, doc)
	}

	// 如果没有解析出文档，返回错误
	if len(docs) == 0 {
		return "", fmt.Errorf("PDF文件解析结果为空")
	}

	// 重置内容构建器
	p.contentBuilder.Reset()
	// 合并所有文档内容
	for _, doc := range docs {
		// 添加标题（如果有）
		if title, ok := doc.MetaData["title"].(string); ok && title != "" {
			p.contentBuilder.WriteString(fmt.Sprintf("# %s\n\n", title))
		}

		// 添加文档内容
		p.contentBuilder.WriteString(ensureParagraphBreaks(ensureHeaderNewlines(normalizeUTF8(doc.Content))))
		p.contentBuilder.WriteString("\n\n")
	}

	return p.contentBuilder.String(), nil
}

// ParseToDocuments 解析PDF文件并返回文档对象数组
func (p *PDFParser) ParseToDocuments(filePath string) ([]*schema.Document, error) {
	// 打开PDF文件
	reader, err := pdflib.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("解析PDF文件失败: %w", err)
	}
	// dslipak/pdf库不需要显式关闭

	// 创建文档对象数组
	docs := []*schema.Document{}

	// 提取PDF内容
	numPages := reader.NumPage()

	for i := 1; i <= numPages; i++ {
		page := reader.Page(i)
		content := page.Content()

		// 将Text切片转换为字符串
		var textContent string
		for _, t := range content.Text {
			textContent += normalizeUTF8(t.S)
		}

		// 预处理：在“第X章/第X条”前插入换行，并按中文缩进断段
		textContent = ensureParagraphBreaks(ensureHeaderNewlines(textContent))

		// 创建文档对象
		doc := &schema.Document{
			ID:      fmt.Sprintf("page_%d", i),
			Content: textContent,
			MetaData: map[string]interface{}{
				"page": i,
			},
		}
		docs = append(docs, doc)
	}

	return docs, nil
}

// normalizeUTF8 确保输出为有效UTF-8，去除非法控制符
func normalizeUTF8(s string) string {
	if s == "" {
		return s
	}
	if utf8.ValidString(s) {
		s = strings.Map(func(r rune) rune {
			if r < 32 && r != '\n' && r != '\t' && r != '\r' {
				return -1
			}
			return r
		}, s)
		return s
	}
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
	s = cleanupPageNumberArtifacts(s)
	re := regexp.MustCompile(`(?m)([^\n])(第[一二三四五六七八九十百千零〇0-9]+(章|条))`)
	s = re.ReplaceAllString(s, "$1\n$2")
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
	linePattern := regexp.MustCompile(`(?m)^\s*[—–−‑‒―－-]+\s*\d+\s*[—–−‑‒―－-]+\s*$`)
	s = linePattern.ReplaceAllString(s, "")
	inlinePattern := regexp.MustCompile(`[—–−‑‒―－-]+\s*\d+\s*[—–−‑‒―－-]+`)
	s = inlinePattern.ReplaceAllString(s, " ")
	return s
}

// ensureParagraphBreaks 尝试在中文段落缩进“　　”前加换行，保留自然段
func ensureParagraphBreaks(s string) string {
	if s == "" {
		return s
	}
	s = regexp.MustCompile(`([。；；;！？!?])\s*　　`).ReplaceAllString(s, "$1\n　　")
	s = regexp.MustCompile(`(?m)(第[一二三四五六七八九十百千零〇0-9]+条)\s+`).ReplaceAllString(s, "$1\n")
	s = regexp.MustCompile(`\n{3,}`).ReplaceAllString(s, "\n\n")
	return s
}
