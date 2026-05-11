package parser

import "github.com/cloudwego/eino/schema"

// DocumentParser 文档解析器统一接口
type DocumentParser interface {
	// Parse 解析文档文件并返回提取的文本内容
	Parse(filePath string) (string, error)

	// ParseToDocuments 解析文档文件并返回文档对象数组（按页/节分块）
	ParseToDocuments(filePath string) ([]*schema.Document, error)

	// ParseWithMetadata 解析文档并返回内容和元数据
	ParseWithMetadata(filePath string) (*ParseResult, error)

	// GetSupportedFormats 获取支持的文件格式
	GetSupportedFormats() []string
}

// ParseResult 解析结果
type ParseResult struct {
	// Content 提取的文本内容
	Content string `json:"content"`

	// Documents 文档对象数组（按页/节分块）
	Documents []*schema.Document `json:"documents"`

	// Metadata 文档元数据
	Metadata *DocumentMetadata `json:"metadata"`

	// QualityScore 文档质量评分（0.0-1.0）
	QualityScore float64 `json:"quality_score"`

	// ParseMethod 使用的解析方法（如: native, llm_vision）
	ParseMethod string `json:"parse_method"`

	// Tables 提取的表格数据
	Tables []*Table `json:"tables,omitempty"`
}

// DocumentMetadata 文档元数据
type DocumentMetadata struct {
	// FileName 文件名
	FileName string `json:"file_name"`

	// FileType 文件类型（pdf, docx等）
	FileType string `json:"file_type"`

	// FileSize 文件大小（字节）
	FileSize int64 `json:"file_size"`

	// PageCount 页数（PDF）或节数（DOCX）
	PageCount int `json:"page_count"`

	// TextLength 文本长度
	TextLength int `json:"text_length"`

	// TableCount 表格数量
	TableCount int `json:"table_count"`

	// HasImages 是否包含图片
	HasImages bool `json:"has_images"`

	// Author 作者（如果有）
	Author string `json:"author,omitempty"`

	// Title 标题（如果有）
	Title string `json:"title,omitempty"`

	// CreatedAt 创建时间（如果有）
	CreatedAt string `json:"created_at,omitempty"`
}

// Table 表格数据
type Table struct {
	// ID 表格ID
	ID string `json:"id"`

	// PageNumber 所在页码
	PageNumber int `json:"page_number"`

	// Title 表格标题
	Title string `json:"title,omitempty"`

	// Headers 表头
	Headers []string `json:"headers"`

	// Rows 行数据
	Rows [][]string `json:"rows"`

	// RawText 原始文本
	RawText string `json:"raw_text"`
}

// ParserFactory 解析器工厂
type ParserFactory struct{}

// NewParserFactory 创建解析器工厂
func NewParserFactory() *ParserFactory {
	return &ParserFactory{}
}

// GetParser 根据文件类型获取对应的解析器
func (f *ParserFactory) GetParser(fileType string) (DocumentParser, error) {
	switch fileType {
	case "pdf", ".pdf":
		return NewPDFParser(), nil
	case "docx", ".docx":
		return NewDOCXParser(), nil
	case "doc", ".doc":
		return NewDOCParser(), nil
	case "txt", ".txt":
		return NewTXTParser(), nil
	case "md", ".md":
		return NewMDParser(), nil
	case "html", ".html", "htm", ".htm":
		return NewHTMLParser(), nil
	default:
		return nil, ErrUnsupportedFormat
	}
}

// ErrUnsupportedFormat 不支持的文件格式错误
var ErrUnsupportedFormat = &ParserError{
	Code:    "UNSUPPORTED_FORMAT",
	Message: "不支持的文件格式",
}

// ParserError 解析错误
type ParserError struct {
	Code    string
	Message string
	Err     error
}

func (e *ParserError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *ParserError) Unwrap() error {
	return e.Err
}
