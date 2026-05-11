package document_api

import (
	"backend/document/llm_fallback"
	"backend/document/parser"
	"backend/document/validator"
)

// DocumentAPI 文档API封装
type DocumentAPI struct {
	parserFactory    *parser.ParserFactory
	qualityChecker   *validator.QualityChecker
	fallbackStrategy *llm_fallback.FallbackParseStrategy
}

// NewDocumentAPI 创建文档API实例
func NewDocumentAPI() *DocumentAPI {
	return &DocumentAPI{
		parserFactory:  parser.NewParserFactory(),
		qualityChecker: validator.NewQualityChecker(),
		// TODO: 从配置文件读取LLM配置
		fallbackStrategy: llm_fallback.NewFallbackParseStrategy(
			0.7, // 质量阈值
			"",  // API Key（从配置读取）
			"",  // Base URL（从配置读取）
			"",  // Model（从配置读取）
		),
	}
}
