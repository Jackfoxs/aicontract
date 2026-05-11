package parser

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cloudwego/eino/schema"
)

// TXTParser 纯文本解析器
type TXTParser struct{}

// NewTXTParser 创建TXT解析器
func NewTXTParser() *TXTParser {
	return &TXTParser{}
}

// Parse 读取txt全文
func (t *TXTParser) Parse(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", &ParserError{Code: "FILE_OPEN_ERROR", Message: "打开TXT文件失败", Err: err}
	}
	defer f.Close()

	var b strings.Builder
	s := bufio.NewScanner(f)
	// 提高Scanner缓冲，避免超长行截断
	buf := make([]byte, 0, 1024*1024)
	s.Buffer(buf, 1024*1024)
	for s.Scan() {
		b.WriteString(s.Text())
		b.WriteString("\n")
	}
	if err := s.Err(); err != nil {
		return "", &ParserError{Code: "FILE_READ_ERROR", Message: "读取TXT文件失败", Err: err}
	}

	content := b.String()
	if strings.TrimSpace(content) == "" {
		return "", &ParserError{Code: "EMPTY_CONTENT", Message: "TXT文件内容为空"}
	}
	return content, nil
}

// ParseToDocuments 将txt按段落/长度切分为Document数组
func (t *TXTParser) ParseToDocuments(filePath string) ([]*schema.Document, error) {
	content, err := t.Parse(filePath)
	if err != nil {
		return nil, err
	}

	return splitTextToDocuments(content, 1200, 200), nil
}

// ParseWithMetadata 返回解析结果与元数据
func (t *TXTParser) ParseWithMetadata(filePath string) (*ParseResult, error) {
	fi, err := os.Stat(filePath)
	if err != nil {
		return nil, &ParserError{Code: "FILE_NOT_FOUND", Message: "文件不存在", Err: err}
	}

	content, err := t.Parse(filePath)
	if err != nil {
		return nil, err
	}

	docs := splitTextToDocuments(content, 1200, 200)
	result := &ParseResult{
		Content:   content,
		Documents: docs,
		Metadata: &DocumentMetadata{
			FileName:   filepath.Base(filePath),
			FileType:   "txt",
			FileSize:   fi.Size(),
			PageCount:  len(docs),
			TextLength: len(content),
			TableCount: 0,
			HasImages:  false,
		},
		QualityScore: 1.0,
		ParseMethod:  "native_txt",
	}
	return result, nil
}

// GetSupportedFormats 返回支持的扩展名
func (t *TXTParser) GetSupportedFormats() []string { return []string{"txt", ".txt"} }

// splitTextToDocuments 根据最大长度与重叠切分
func splitTextToDocuments(text string, maxLen int, overlap int) []*schema.Document {
	paragraphs := strings.Split(text, "\n\n")
	var docs []*schema.Document
	var b strings.Builder
	idx := 0

	appendDoc := func(s string) {
		idx++
		docs = append(docs, &schema.Document{
			ID:      "chunk_" + strconv.Itoa(idx),
			Content: s,
			MetaData: map[string]interface{}{
				"file_type": "txt",
				"chunk":     idx,
			},
		})
	}

	for i := 0; i < len(paragraphs); {
		b.Reset()
		for i < len(paragraphs) && b.Len()+len(paragraphs[i]) <= maxLen {
			if b.Len() > 0 {
				b.WriteString("\n\n")
			}
			b.WriteString(strings.TrimSpace(paragraphs[i]))
			i++
		}
		chunk := strings.TrimSpace(b.String())
		if chunk != "" {
			appendDoc(chunk)
		}
		// 回退overlap段，制造重叠
		if overlap > 0 {
			back := overlap / 100 // 将字符重叠折算为段落近似数量
			if back > 0 {
				i -= back
				if i < 0 {
					i = 0
				}
			}
		}
	}
	return docs
}

// 无
