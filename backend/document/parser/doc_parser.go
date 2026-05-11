package parser

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/cloudwego/eino/schema"
)

// DOCParser 旧版Word(.doc)解析器（通过系统工具转换为文本）
type DOCParser struct{}

// NewDOCParser 创建 .doc 解析器
func NewDOCParser() *DOCParser {
	return &DOCParser{}
}

// Parse 解析 .doc 文件为纯文本
func (p *DOCParser) Parse(filePath string) (string, error) {
	text, err := convertDocToText(filePath)
	if err != nil {
		return "", &ParserError{Code: "DOC_PARSE_ERROR", Message: "解析DOC文件失败", Err: err}
	}
	text = normalizeUTF8Doc(text)
	if strings.TrimSpace(text) == "" {
		return "", &ParserError{Code: "EMPTY_CONTENT", Message: "DOC文件解析结果为空"}
	}
	return text, nil
}

// ParseToDocuments 将 .doc 转为段落并分块
func (p *DOCParser) ParseToDocuments(filePath string) ([]*schema.Document, error) {
	content, err := p.Parse(filePath)
	if err != nil {
		return nil, err
	}
	paras := splitToParagraphs(content)
	if len(paras) == 0 {
		return nil, &ParserError{Code: "EMPTY_CONTENT", Message: "DOC文件解析结果为空"}
	}

	chunkSize := 10
	sectionIndex := 0
	var current strings.Builder
	count := 0

	docs := []*schema.Document{}
	for _, p := range paras {
		line := strings.TrimSpace(p)
		if line == "" {
			continue
		}
		if current.Len() > 0 {
			current.WriteString("\n")
		}
		current.WriteString(line)
		count++
		if count >= chunkSize {
			sectionIndex++
			docs = append(docs, &schema.Document{
				ID:      fmt.Sprintf("section_%d", sectionIndex),
				Content: current.String(),
				MetaData: map[string]interface{}{
					"section":   sectionIndex,
					"file_type": "doc",
				},
			})
			current.Reset()
			count = 0
		}
	}
	if current.Len() > 0 {
		sectionIndex++
		docs = append(docs, &schema.Document{
			ID:      fmt.Sprintf("section_%d", sectionIndex),
			Content: current.String(),
			MetaData: map[string]interface{}{
				"section":   sectionIndex,
				"file_type": "doc",
			},
		})
	}
	return docs, nil
}

// ParseWithMetadata 解析并返回元数据
func (p *DOCParser) ParseWithMetadata(filePath string) (*ParseResult, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, &ParserError{Code: "FILE_NOT_FOUND", Message: "文件不存在", Err: err}
	}

	content, err := p.Parse(filePath)
	if err != nil {
		return nil, err
	}
	paras := splitToParagraphs(content)

	docs, _ := p.ParseToDocuments(filePath)

	result := &ParseResult{
		Content:   content,
		Documents: docs,
		Metadata: &DocumentMetadata{
			FileName:   filepath.Base(filePath),
			FileType:   "doc",
			FileSize:   info.Size(),
			PageCount:  len(paras),
			TextLength: len(content),
			TableCount: 0,
			HasImages:  false,
		},
		QualityScore: calculateQualityScoreDoc(content, len(paras)),
		ParseMethod:  "native_doc",
		Tables:       []*Table{},
	}
	return result, nil
}

// GetSupportedFormats 支持的格式
func (p *DOCParser) GetSupportedFormats() []string {
	return []string{"doc", ".doc"}
}

// convertDocToText 使用系统工具将 .doc 转为文本（按可用性优先级尝试）
func convertDocToText(filePath string) (string, error) {
	// 1) macOS: textutil
	if path, _ := exec.LookPath("textutil"); path != "" {
		cmd := exec.Command(path, "-convert", "txt", "-stdout", filePath)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		if err := cmd.Run(); err == nil {
			return out.String(), nil
		}
	}

	// 2) LibreOffice / OpenOffice: soffice
	if path, _ := exec.LookPath("soffice"); path != "" {
		tmpDir, _ := os.MkdirTemp("", "doc2txt-*")
		defer os.RemoveAll(tmpDir)
		cmd := exec.Command(path, "--headless", "--convert-to", "txt", "--outdir", tmpDir, filePath)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		if err := cmd.Run(); err == nil {
			base := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath)) + ".txt"
			txtPath := filepath.Join(tmpDir, base)
			if b, e := os.ReadFile(txtPath); e == nil {
				return string(b), nil
			}
		}
	}

	// 3) antiword
	if path, _ := exec.LookPath("antiword"); path != "" {
		cmd := exec.Command(path, filePath)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		if err := cmd.Run(); err == nil {
			return out.String(), nil
		}
	}

	// 4) catdoc
	if path, _ := exec.LookPath("catdoc"); path != "" {
		cmd := exec.Command(path, filePath)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		if err := cmd.Run(); err == nil {
			return out.String(), nil
		}
	}

	return "", fmt.Errorf("未找到可用的DOC转换工具，请安装 textutil(仅macOS)/soffice/antiword/catdoc 之一")
}

// splitToParagraphs 简单按双换行或单换行切分
func splitToParagraphs(content string) []string {
	if strings.Contains(content, "\r\n\r\n") || strings.Contains(content, "\n\n") {
		return strings.Split(content, "\n\n")
	}
	return strings.Split(content, "\n")
}

// calculateQualityScoreDoc 与 docx 逻辑一致，简化计分
func calculateQualityScoreDoc(content string, paragraphCount int) float64 {
	if content == "" {
		return 0.0
	}
	score := 1.0
	avg := float64(len(content)) / float64(maxInt(1, paragraphCount))
	if avg < 10 {
		score -= 0.2
	}
	garbled := strings.Count(content, "�")
	garbledRatio := float64(garbled) / float64(len(content))
	if garbledRatio > 0.1 {
		score -= 0.4
	} else if garbledRatio > 0.05 {
		score -= 0.2
	}
	punct := strings.Count(content, "。") + strings.Count(content, "，") + strings.Count(content, ".") + strings.Count(content, ",")
	if punct < len(content)/50 {
		score -= 0.1
	}
	if score < 0 {
		return 0
	}
	return score
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// normalizeUTF8Doc 保证UTF-8并去除非法控制字符
func normalizeUTF8Doc(s string) string {
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
