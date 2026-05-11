package models

import "time"

// DocumentParseLog 文档解析日志表
type DocumentParseLog struct {
	ID           uint64    `json:"id" xorm:"pk autoincr bigint 'id'"`
	FilePath     string    `json:"file_path" xorm:"varchar(500) 'file_path' comment('文件路径')"`
	FileName     string    `json:"file_name" xorm:"varchar(255) 'file_name' comment('文件名')"`
	FileType     string    `json:"file_type" xorm:"varchar(20) 'file_type' comment('文件类型')"`
	FileSize     int64     `json:"file_size" xorm:"bigint 'file_size' comment('文件大小(字节)')"`
	ParseMethod  string    `json:"parse_method" xorm:"varchar(50) 'parse_method' comment('解析方法: native_pdf, native_docx, llm_vision_fallback')"`
	QualityScore float64   `json:"quality_score" xorm:"decimal(3,2) 'quality_score' comment('质量评分 0.0-1.0')"`
	ParseTime    int       `json:"parse_time" xorm:"int 'parse_time' comment('解析耗时(ms)')"`
	TextLength   int       `json:"text_length" xorm:"int 'text_length' comment('提取文本长度')"`
	TableCount   int       `json:"table_count" xorm:"int 'table_count' comment('表格数量')"`
	Success      bool      `json:"success" xorm:"bool default 1 'success' comment('是否成功')"`
	ErrorMsg     string    `json:"error_msg" xorm:"text 'error_msg' comment('错误信息')"`
	LLMCost      float64   `json:"llm_cost" xorm:"decimal(10,4) 'llm_cost' comment('LLM成本(元)')"`
	CreatedAt    time.Time `json:"created_at" xorm:"created 'created_at'"`
}

// TableName 指定表名
func (DocumentParseLog) TableName() string {
	return "document_parse_logs"
}
