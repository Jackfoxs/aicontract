package parser

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cloudwego/eino/schema"
)

// MDParser Markdown 解析器（不引入新依赖，做基础文本抽取）
type MDParser struct{}

func NewMDParser() *MDParser { return &MDParser{} }

// Parse 读取并将Markdown转换为纯文本
func (m *MDParser) Parse(filePath string) (string, error) {
	content, err := readAllText(filePath)
	if err != nil {
		return "", &ParserError{Code: "FILE_READ_ERROR", Message: "读取Markdown失败", Err: err}
	}
	text := markdownToText(content)
	if strings.TrimSpace(text) == "" {
		return "", &ParserError{Code: "EMPTY_CONTENT", Message: "Markdown解析结果为空"}
	}
	return text, nil
}

// ParseToDocuments 切分为文档块
func (m *MDParser) ParseToDocuments(filePath string) ([]*schema.Document, error) {
	content, err := m.Parse(filePath)
	if err != nil {
		return nil, err
	}
	return splitTextToDocuments(content, 1200, 200), nil
}

// ParseWithMetadata 返回解析结果和元数据
func (m *MDParser) ParseWithMetadata(filePath string) (*ParseResult, error) {
	fi, err := os.Stat(filePath)
	if err != nil {
		return nil, &ParserError{Code: "FILE_NOT_FOUND", Message: "文件不存在", Err: err}
	}
	content, err := m.Parse(filePath)
	if err != nil {
		return nil, err
	}
	docs := splitTextToDocuments(content, 1200, 200)
	return &ParseResult{
		Content:   content,
		Documents: docs,
		Metadata: &DocumentMetadata{
			FileName:   filepath.Base(filePath),
			FileType:   "md",
			FileSize:   fi.Size(),
			PageCount:  len(docs),
			TextLength: len(content),
			TableCount: 0,
			HasImages:  false,
		},
		QualityScore: 1.0,
		ParseMethod:  "native_md",
	}, nil
}

func (m *MDParser) GetSupportedFormats() []string { return []string{"md", ".md"} }

// --- helpers ---

func readAllText(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	var b strings.Builder
	s := bufio.NewScanner(f)
	buf := make([]byte, 0, 1024*1024)
	s.Buffer(buf, 1024*1024)
	for s.Scan() {
		b.WriteString(s.Text())
		b.WriteString("\n")
	}
	if err := s.Err(); err != nil {
		return "", err
	}
	return b.String(), nil
}

// markdownToText 做基础的MD->Text转换：
// - 去掉代码块/行内代码/标题标记/列表符号
// - 链接与图片保留可读文本（[text](url) -> text, ![alt](url) -> alt）
// - 保留段落换行
func markdownToText(s string) string {
	// 去代码块 ```...```
	reCodeBlock := regexp.MustCompile("(?s)```.*?```")
	s = reCodeBlock.ReplaceAllString(s, "")
	// 行内代码 `...`
	reInline := regexp.MustCompile("`.+?`")
	s = reInline.ReplaceAllString(s, "")
	// 图片 ![alt](url) -> alt
	reImg := regexp.MustCompile(`!\[(.*?)\]\([^)]*\)`) // 捕获alt
	s = reImg.ReplaceAllString(s, "$1")
	// 链接 [text](url) -> text
	reLink := regexp.MustCompile(`\[(.*?)\]\([^)]*\)`) // 捕获text
	s = reLink.ReplaceAllString(s, "$1")
	// 标题 #、##、###
	reHeading := regexp.MustCompile(`(?m)^\s{0,3}#{1,6}\s*`)
	s = reHeading.ReplaceAllString(s, "")
	// 列表符号 -, *, +, 编号1.
	reList := regexp.MustCompile(`(?m)^\s*([\-*\+]|\d+\.)\s+`)
	s = reList.ReplaceAllString(s, "")
	// 引用 >
	reQuote := regexp.MustCompile(`(?m)^\s*>\s?`)
	s = reQuote.ReplaceAllString(s, "")
	// 表格分隔 | 和 --- 粗略去掉
	reTableSep := regexp.MustCompile(`(?m)^\s*\|?\s*:?[-=]{2,}.*\|?.*$`)
	s = reTableSep.ReplaceAllString(s, "")
	// 收敛多余空行
	reNewlines := regexp.MustCompile(`\n{3,}`)
	s = reNewlines.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}

