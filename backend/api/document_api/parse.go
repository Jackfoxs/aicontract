package document_api

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"backend/global"
	"backend/models/res"

	"github.com/cloudwego/hertz/pkg/app"
)

// ParseRequest 解析请求参数
type ParseRequest struct {
	FilePath string `json:"file_path" binding:"required"`
}

// ParseResponse 解析响应
type ParseResponse struct {
	Success      bool                     `json:"success"`
	Content      string                   `json:"content"`
	Metadata     map[string]interface{}   `json:"metadata"`
	QualityScore float64                  `json:"quality_score"`
	ParseMethod  string                   `json:"parse_method"`
	Tables       []map[string]interface{} `json:"tables,omitempty"`
	Message      string                   `json:"message,omitempty"`
}

// DocumentParse 测试文档解析
// @Summary 测试文档解析
// @Description 解析PDF或DOCX文件，返回提取的文本内容和质量报告
// @Tags 文档管理
// @Accept json
// @Produce json
// @Param file_path body string true "文件路径"
// @Success 200 {object} ParseResponse
// @Router /api/document/parse [post]
func (api *DocumentAPI) DocumentParse(c context.Context, ctx *app.RequestContext) {
	var req ParseRequest
	if err := ctx.BindAndValidate(&req); err != nil {
		res.FailWithMessage("参数错误: "+err.Error(), c, ctx)
		return
	}

	// 检查文件类型
	fileExt := strings.ToLower(filepath.Ext(req.FilePath))
	if fileExt != ".pdf" && fileExt != ".docx" && fileExt != ".doc" {
		res.FailWithMessage("不支持的文件格式，仅支持PDF、DOCX、DOC", c, ctx)
		return
	}

	// 获取解析器
	docParser, err := api.parserFactory.GetParser(fileExt)
	if err != nil {
		global.Log.Error("获取解析器失败", "error", err)
		res.FailWithMessage("获取解析器失败", c, ctx)
		return
	}

	// 解析文档
	result, err := docParser.ParseWithMetadata(req.FilePath)
	if err != nil {
		global.Log.Error("解析文档失败", "file", req.FilePath, "error", err)
		res.FailWithMessage("解析文档失败: "+err.Error(), c, ctx)
		return
	}

	// 构建响应
	metadata := map[string]interface{}{
		"file_name":   result.Metadata.FileName,
		"file_type":   result.Metadata.FileType,
		"file_size":   result.Metadata.FileSize,
		"page_count":  result.Metadata.PageCount,
		"text_length": result.Metadata.TextLength,
		"table_count": result.Metadata.TableCount,
	}

	// 转换表格数据
	tables := []map[string]interface{}{}
	for _, table := range result.Tables {
		tables = append(tables, map[string]interface{}{
			"id":          table.ID,
			"page_number": table.PageNumber,
			"title":       table.Title,
			"headers":     table.Headers,
			"rows":        table.Rows,
		})
	}

	response := ParseResponse{
		Success:      true,
		Content:      result.Content,
		Metadata:     metadata,
		QualityScore: result.QualityScore,
		ParseMethod:  result.ParseMethod,
		Tables:       tables,
		Message:      fmt.Sprintf("成功解析文档，质量评分: %.2f", result.QualityScore),
	}

	res.OkWithData(response, c, ctx)
}

// QualityCheckRequest 质量检查请求
type QualityCheckRequest struct {
	Content string `json:"content" binding:"required"`
}

// QualityCheckResponse 质量检查响应
type QualityCheckResponse struct {
	Success        bool                     `json:"success"`
	Score          float64                  `json:"score"`
	Passed         bool                     `json:"passed"`
	Issues         []map[string]interface{} `json:"issues"`
	Metrics        map[string]interface{}   `json:"metrics"`
	Recommendation string                   `json:"recommendation"`
}

// DocumentQualityCheck 文档质量检查
// @Summary 文档质量检查
// @Description 检查文档内容的质量，返回质量评分和问题报告
// @Tags 文档管理
// @Accept json
// @Produce json
// @Param content body string true "文档内容"
// @Success 200 {object} QualityCheckResponse
// @Router /api/document/quality-check [post]
func (api *DocumentAPI) DocumentQualityCheck(c context.Context, ctx *app.RequestContext) {
	var req QualityCheckRequest
	if err := ctx.BindAndValidate(&req); err != nil {
		res.FailWithMessage("参数错误: "+err.Error(), c, ctx)
		return
	}

	// 执行质量检查
	report := api.qualityChecker.Check(req.Content)

	// 转换问题列表
	issues := []map[string]interface{}{}
	for _, issue := range report.Issues {
		issues = append(issues, map[string]interface{}{
			"type":        issue.Type,
			"severity":    issue.Severity,
			"description": issue.Description,
			"impact":      issue.Impact,
		})
	}

	// 转换指标
	metrics := map[string]interface{}{
		"total_chars":        report.Metrics.TotalChars,
		"total_words":        report.Metrics.TotalWords,
		"total_lines":        report.Metrics.TotalLines,
		"avg_chars_per_line": report.Metrics.AvgCharsPerLine,
		"garbled_ratio":      report.Metrics.GarbledRatio,
		"punctuation_ratio":  report.Metrics.PunctuationRatio,
		"chinese_ratio":      report.Metrics.ChineseRatio,
		"english_ratio":      report.Metrics.EnglishRatio,
	}

	response := QualityCheckResponse{
		Success:        true,
		Score:          report.Score,
		Passed:         report.Passed,
		Issues:         issues,
		Metrics:        metrics,
		Recommendation: report.Recommendation,
	}

	res.OkWithData(response, c, ctx)
}

// ParseWithFallbackRequest 带降级解析请求
type ParseWithFallbackRequest struct {
	FilePath string `json:"file_path" binding:"required"`
}

// DocumentParseWithFallback 带降级策略的文档解析
// @Summary 带降级策略的文档解析
// @Description 先尝试原生解析，质量低则自动降级到LLM视觉解析
// @Tags 文档管理
// @Accept json
// @Produce json
// @Param file_path body string true "文件路径"
// @Success 200 {object} ParseResponse
// @Router /api/document/parse-with-fallback [post]
func (api *DocumentAPI) DocumentParseWithFallback(c context.Context, ctx *app.RequestContext) {
	var req ParseWithFallbackRequest
	if err := ctx.BindAndValidate(&req); err != nil {
		res.FailWithMessage("参数错误: "+err.Error(), c, ctx)
		return
	}

	// 检查文件类型
	fileExt := strings.ToLower(filepath.Ext(req.FilePath))
	if fileExt != ".pdf" && fileExt != ".docx" && fileExt != ".doc" {
		res.FailWithMessage("不支持的文件格式，仅支持PDF、DOCX、DOC", c, ctx)
		return
	}

	// 使用降级策略解析
	result, err := api.fallbackStrategy.ParseWithFallback(req.FilePath, fileExt)
	if err != nil {
		global.Log.Error("解析文档失败", "file", req.FilePath, "error", err)
		res.FailWithMessage("解析文档失败: "+err.Error(), c, ctx)
		return
	}

	// 构建响应
	metadata := map[string]interface{}{
		"file_name":   result.Metadata.FileName,
		"file_type":   result.Metadata.FileType,
		"file_size":   result.Metadata.FileSize,
		"page_count":  result.Metadata.PageCount,
		"text_length": result.Metadata.TextLength,
		"table_count": result.Metadata.TableCount,
	}

	tables := []map[string]interface{}{}
	for _, table := range result.Tables {
		tables = append(tables, map[string]interface{}{
			"id":          table.ID,
			"page_number": table.PageNumber,
			"title":       table.Title,
			"headers":     table.Headers,
			"rows":        table.Rows,
		})
	}

	var message string
	if result.ParseMethod == "llm_vision_fallback" {
		message = fmt.Sprintf("原生解析质量不佳，已使用LLM视觉解析。质量评分: %.2f", result.QualityScore)
	} else {
		message = fmt.Sprintf("使用原生解析成功。质量评分: %.2f", result.QualityScore)
	}

	response := ParseResponse{
		Success:      true,
		Content:      result.Content,
		Metadata:     metadata,
		QualityScore: result.QualityScore,
		ParseMethod:  result.ParseMethod,
		Tables:       tables,
		Message:      message,
	}

	res.OkWithData(response, c, ctx)
}
