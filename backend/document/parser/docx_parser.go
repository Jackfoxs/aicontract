package parser

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/cloudwego/eino/schema"
	docx "github.com/fumiama/go-docx"
)

// DOCXParser DOCX解析器
type DOCXParser struct {
	contentBuilder strings.Builder
	lastStats      docxParseStats
}

type docxParseStats struct {
	Paragraphs     []string
	PageCount      int
	RuneCount      int
	UsedFallback   bool
	Author         string
	Title          string
	CreatedAt      string
	ParagraphCount int
}

// NewDOCXParser 创建DOCX解析器
func NewDOCXParser() *DOCXParser {
	return &DOCXParser{}
}

// Parse 解析DOCX文件并返回提取的文本内容
func (d *DOCXParser) Parse(filePath string) (string, error) {
	content, stats, err := d.parseContent(filePath)
	if err != nil {
		return "", err
	}
	d.lastStats = stats
	return content, nil
}

// ParseToDocuments 解析DOCX文件并返回文档对象数组（按节分块）
func (d *DOCXParser) ParseToDocuments(filePath string) ([]*schema.Document, error) {
	content, stats, err := d.parseContent(filePath)
	if err != nil {
		return nil, err
	}
	d.lastStats = stats
	return buildDocumentsFromParagraphs(stats.Paragraphs, content), nil
}

// ParseWithMetadata 解析DOCX并返回内容和元数据
func (d *DOCXParser) ParseWithMetadata(filePath string) (*ParseResult, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, &ParserError{
			Code:    "FILE_NOT_FOUND",
			Message: "文件不存在",
			Err:     err,
		}
	}
	fullContent, stats, err := d.parseContent(filePath)
	if err != nil {
		return nil, err
	}
	d.lastStats = stats

	textLength := stats.RuneCount
	qualityScore := d.calculateQualityScore(fullContent, stats)

	docs := buildDocumentsFromParagraphs(stats.Paragraphs, fullContent)

	result := &ParseResult{
		Content:   fullContent,
		Documents: docs,
		Metadata: &DocumentMetadata{
			FileName:   filepath.Base(filePath),
			FileType:   "docx",
			FileSize:   fileInfo.Size(),
			PageCount:  stats.PageCount,
			TextLength: textLength,
			TableCount: 0,
			HasImages:  false,
			Author:     stats.Author,
			Title:      stats.Title,
			CreatedAt:  stats.CreatedAt,
		},
		QualityScore: qualityScore,
		ParseMethod:  "native_docx",
		Tables:       []*Table{},
	}

	return result, nil
}

// GetSupportedFormats 获取支持的文件格式
func (d *DOCXParser) GetSupportedFormats() []string {
	return []string{"docx", ".docx"}
}

// calculateQualityScore 计算文档质量评分
func (d *DOCXParser) calculateQualityScore(content string, stats docxParseStats) float64 {
	if content == "" {
		return 0.0
	}

	score := 1.0

	if stats.UsedFallback {
		score -= 0.15
	}

	avgCharsPerParagraph := float64(stats.RuneCount)
	if stats.ParagraphCount > 0 {
		avgCharsPerParagraph = avgCharsPerParagraph / float64(stats.ParagraphCount)
	}
	if avgCharsPerParagraph < 30 {
		score -= 0.2
	} else if avgCharsPerParagraph < 60 {
		score -= 0.05
	}

	// 乱码占比
	garbledCount := 0
	for _, char := range content {
		if char == '�' || char == '\ufffd' {
			garbledCount++
		}
	}
	garbledRatio := 0.0
	if stats.RuneCount > 0 {
		garbledRatio = float64(garbledCount) / float64(stats.RuneCount)
	}
	if garbledRatio > 0.1 {
		score -= 0.4
	} else if garbledRatio > 0.05 {
		score -= 0.2
	}

	// 检查标点比例
	punctCount := strings.Count(content, "。") + strings.Count(content, "，") +
		strings.Count(content, ".") + strings.Count(content, ",")

	if stats.RuneCount > 0 && float64(punctCount) < float64(stats.RuneCount)/60.0 {
		score -= 0.1
	}

	if stats.PageCount == 0 && stats.ParagraphCount > 120 {
		score -= 0.1
	}

	if score > 1.0 {
		score = 1.0
	}
	if score < 0.0 {
		score = 0.0
	}

	return math.Round(score*100) / 100
}

// parseWithGoDocx 使用 github.com/fumiama/go-docx 解析（若失败会被上层回退）
func (d *DOCXParser) parseWithGoDocx(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return "", err
	}
	file, err := docx.Parse(f, fi.Size())
	if err != nil {
		return "", err
	}
	d.contentBuilder.Reset()
	for _, item := range file.Document.Body.Items {
		if s, ok := item.(interface{ String() string }); ok {
			text := strings.TrimSpace(s.String())
			if text != "" {
				d.contentBuilder.WriteString(text)
				d.contentBuilder.WriteString("\n")
			}
		}
	}
	return normalizeLineEndings(d.contentBuilder.String()), nil
}

// parseDocumentXML 直接读取 docx 的 word/document.xml 抽取 w:t 文本
func (d *DOCXParser) parseDocumentXML(filePath string) ([]string, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var docXML []byte
	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			data, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return nil, err
			}
			docXML = data
			break
		}
	}
	if len(docXML) == 0 {
		return nil, fmt.Errorf("document.xml 未找到")
	}

	dec := xml.NewDecoder(bytes.NewReader(docXML))
	paragraphs := make([]string, 0, 128)
	var current strings.Builder
	inParagraph := false
	inText := false

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			local := t.Name.Local
			if local == "p" { // 段落开始
				if inParagraph && current.Len() > 0 {
					paragraphs = append(paragraphs, current.String())
					current.Reset()
				}
				inParagraph = true
			} else if local == "t" { // 文本节点
				inText = true
			} else if local == "br" { // 换行
				if inParagraph {
					current.WriteString("\n")
				}
			} else if local == "tab" { // 制表符
				if inParagraph {
					current.WriteString("\t")
				}
			}
		case xml.EndElement:
			local := t.Name.Local
			if local == "t" {
				inText = false
			} else if local == "p" { // 段落结束
				if current.Len() > 0 {
					paragraphs = append(paragraphs, current.String())
					current.Reset()
				} else {
					// 空段落也保留为换行
					paragraphs = append(paragraphs, "")
				}
				inParagraph = false
			}
		case xml.CharData:
			if inText {
				current.WriteString(string(t))
			}
		}
	}
	if current.Len() > 0 {
		paragraphs = append(paragraphs, current.String())
	}
	return paragraphs, nil
}

// extractParagraphs 返回段落切片（用于分块）
func (d *DOCXParser) extractParagraphs(filePath string) ([]string, error) {
	paragraphs, err := d.parseDocumentXML(filePath)
	if err != nil {
		// 退回 go-docx 文本，按双换行近似段落
		if backup, e := d.parseWithGoDocx(filePath); e == nil && strings.TrimSpace(backup) != "" {
			backup = normalizeLineEndings(backup)
			paras := strings.Split(backup, "\n\n")
			return paras, nil
		}
		return nil, err
	}
	return paragraphs, nil
}

func (d *DOCXParser) parseContent(filePath string) (string, docxParseStats, error) {
	stats := docxParseStats{}

	paragraphs, err := d.extractParagraphs(filePath)
	if err != nil {
		return "", stats, &ParserError{Code: "DOCX_PARSE_ERROR", Message: "解析DOCX文件失败", Err: err}
	}
	if len(paragraphs) == 0 {
		return "", stats, &ParserError{Code: "EMPTY_CONTENT", Message: "DOCX文件解析结果为空"}
	}

	paragraphs = trimEmptyTails(paragraphs)
	stats.Paragraphs = paragraphs
	stats.ParagraphCount = len(paragraphs)

	var content string
	if res, err := d.parseWithGoDocx(filePath); err == nil && strings.TrimSpace(res) != "" {
		content = res
	} else {
		content = strings.Join(paragraphs, "\n")
		stats.UsedFallback = true
	}
	content = strings.TrimSpace(content)
	stats.RuneCount = utf8.RuneCountInString(content)

	props, err := readDocxProperties(filePath)
	if err == nil {
		stats.PageCount = props.Pages
		stats.Author = props.Creator
		stats.Title = props.Title
		stats.CreatedAt = props.Created
	}
	if stats.PageCount == 0 {
		stats.PageCount = estimatePageByParagraph(stats.ParagraphCount)
	}

	return content, stats, nil
}

func buildDocumentsFromParagraphs(paragraphs []string, fullContent string) []*schema.Document {
	if len(paragraphs) == 0 {
		return []*schema.Document{}
	}
	docs := []*schema.Document{}
	sectionIndex := 0
	chunkSize := 10
	var current strings.Builder
	count := 0

	for _, p := range paragraphs {
		text := strings.TrimSpace(p)
		if text == "" {
			continue
		}
		if current.Len() > 0 {
			current.WriteString("\n")
		}
		current.WriteString(text)
		count++
		if count >= chunkSize {
			sectionIndex++
			docs = append(docs, &schema.Document{
				ID:      fmt.Sprintf("section_%d", sectionIndex),
				Content: current.String(),
				MetaData: map[string]interface{}{
					"section":   sectionIndex,
					"file_type": "docx",
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
				"file_type": "docx",
			},
		})
	}
	if len(docs) == 0 && strings.TrimSpace(fullContent) != "" {
		docs = append(docs, &schema.Document{
			ID:      "section_1",
			Content: fullContent,
			MetaData: map[string]interface{}{
				"section":   1,
				"file_type": "docx",
			},
		})
	}
	return docs
}

func normalizeLineEndings(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	return text
}

func trimEmptyTails(paragraphs []string) []string {
	start := 0
	end := len(paragraphs)
	for start < end && strings.TrimSpace(paragraphs[start]) == "" {
		start++
	}
	for end > start && strings.TrimSpace(paragraphs[end-1]) == "" {
		end--
	}
	return paragraphs[start:end]
}

type docxProperties struct {
	Pages   int
	Words   int
	Title   string
	Creator string
	Created string
}

func readDocxProperties(filePath string) (docxProperties, error) {
	props := docxProperties{}
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return props, err
	}
	defer r.Close()

	var appXML, coreXML []byte
	for _, f := range r.File {
		switch f.Name {
		case "docProps/app.xml":
			data, err := readZipFile(f)
			if err == nil {
				appXML = data
			}
		case "docProps/core.xml":
			data, err := readZipFile(f)
			if err == nil {
				coreXML = data
			}
		}
	}
	if len(appXML) > 0 {
		props.Pages = extractIntFromXML(appXML, "Pages")
		props.Words = extractIntFromXML(appXML, "Words")
	}
	if len(coreXML) > 0 {
		props.Title = extractStringFromXML(coreXML, "title")
		props.Creator = extractStringFromXML(coreXML, "creator")
		props.Created = extractStringFromXML(coreXML, "created")
	}
	return props, nil
}

func readZipFile(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

func extractIntFromXML(data []byte, localName string) int {
	dec := xml.NewDecoder(bytes.NewReader(data))
	for {
		tok, err := dec.Token()
		if err != nil {
			return 0
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if start.Name.Local == localName {
			var value string
			if err := dec.DecodeElement(&value, &start); err == nil {
				value = strings.TrimSpace(value)
				if i, err := strconv.Atoi(value); err == nil {
					return i
				}
			}
			return 0
		}
	}
}

func extractStringFromXML(data []byte, localName string) string {
	dec := xml.NewDecoder(bytes.NewReader(data))
	for {
		tok, err := dec.Token()
		if err != nil {
			return ""
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if start.Name.Local == localName {
			var value string
			if err := dec.DecodeElement(&value, &start); err == nil {
				return strings.TrimSpace(value)
			}
			return ""
		}
	}
}

func estimatePageByParagraph(paragraphs int) int {
	if paragraphs <= 0 {
		return 0
	}
	pages := (paragraphs + 39) / 40
	if pages < 1 {
		pages = 1
	}
	return pages
}
