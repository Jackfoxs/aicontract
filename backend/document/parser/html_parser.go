package parser

import (
	"html"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cloudwego/eino/schema"
)

// HTMLParser 简易HTML文本抽取器（不引入第三方依赖）
type HTMLParser struct{}

func NewHTMLParser() *HTMLParser { return &HTMLParser{} }

func (h *HTMLParser) Parse(filePath string) (string, error) {
	b, err := os.ReadFile(filePath)
	if err != nil {
		return "", &ParserError{Code: "FILE_READ_ERROR", Message: "读取HTML失败", Err: err}
	}
	raw := string(b)
	text := extractTextFromHTML(raw)
	if strings.TrimSpace(text) == "" {
		return "", &ParserError{Code: "EMPTY_CONTENT", Message: "HTML解析结果为空"}
	}
	return text, nil
}

func (h *HTMLParser) ParseToDocuments(filePath string) ([]*schema.Document, error) {
	content, err := h.Parse(filePath)
	if err != nil {
		return nil, err
	}
	return splitTextToDocuments(content, 1400, 200), nil
}

func (h *HTMLParser) ParseWithMetadata(filePath string) (*ParseResult, error) {
	fi, err := os.Stat(filePath)
	if err != nil {
		return nil, &ParserError{Code: "FILE_NOT_FOUND", Message: "文件不存在", Err: err}
	}
	content, err := h.Parse(filePath)
	if err != nil {
		return nil, err
	}
	docs := splitTextToDocuments(content, 1400, 200)
	return &ParseResult{
		Content:   content,
		Documents: docs,
		Metadata: &DocumentMetadata{
			FileName:   filepath.Base(filePath),
			FileType:   "html",
			FileSize:   fi.Size(),
			PageCount:  len(docs),
			TextLength: len(content),
			TableCount: 0,
			HasImages:  false,
		},
		QualityScore: 1.0,
		ParseMethod:  "native_html",
	}, nil
}

func (h *HTMLParser) GetSupportedFormats() []string { return []string{"html", ".html", "htm", ".htm"} }

// extractTextFromHTML 粗略抽取文本：
// - 去掉<script>/<style>内容
// - 将块级标签替换为换行
// - 清除剩余标签，并反转义实体
func extractTextFromHTML(s string) string {
	// 去 script/style
	reScript := regexp.MustCompile(`(?is)<script[\s\S]*?</script>`)
	s = reScript.ReplaceAllString(s, "")
	reStyle := regexp.MustCompile(`(?is)<style[\s\S]*?</style>`)
	s = reStyle.ReplaceAllString(s, "")

	// 换行的标签统一替换
	blockTags := []string{"p", "div", "br", "li", "tr", "table", "section", "article", "h1", "h2", "h3", "h4", "h5", "h6"}
	for _, tag := range blockTags {
		reOpen := regexp.MustCompile("(?i)<" + tag + "[^>]*>")
		reClose := regexp.MustCompile("(?i)</" + tag + ">")
		s = reOpen.ReplaceAllString(s, "\n")
		s = reClose.ReplaceAllString(s, "\n")
	}

	// 清除其余标签
	reTags := regexp.MustCompile(`(?s)<[^>]+>`) // 其余标签
	s = reTags.ReplaceAllString(s, "")

	// HTML 实体反转义
	s = html.UnescapeString(s)

	// 压缩空白与多余空行
	s = strings.ReplaceAll(s, "\r", "")
	reSpaces := regexp.MustCompile(`[\t\f\v]+`)
	s = reSpaces.ReplaceAllString(s, " ")
	reBlank := regexp.MustCompile(`\n{3,}`)
	s = reBlank.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}

