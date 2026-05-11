package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"backend/document/extractor"
	"backend/document/parser"
	"backend/global"
	"backend/knowledge/embeddings"
	"backend/llm"
	"backend/models"
	"backend/prompts"
	"backend/utils"

	"github.com/bytedance/sonic"
)

// ContractService 合同审核服务
type ContractService struct {
	LLMClient          *llm.DeepSeekClient
	StructureExtractor *extractor.StructureExtractor
	MetadataExtractor  *extractor.MetadataExtractor
	TableExtractor     *extractor.TableExtractor
	ParserFactory      *parser.ParserFactory
}

// NewContractService 创建合同审核服务
func NewContractService() *ContractService {
	return &ContractService{
		LLMClient:          llm.NewDeepSeekClient(),
		StructureExtractor: extractor.NewStructureExtractor(),
		MetadataExtractor:  extractor.NewMetadataExtractor(),
		TableExtractor:     extractor.NewTableExtractor(),
		ParserFactory:      parser.NewParserFactory(),
	}
}

// ParseContractRequest 解析合同请求
type ParseContractRequest struct {
	FilePath      string
	FileName      string
	ProcurementID uint64
	CreatedBy     string
}

// ParseContract 解析合同
func (s *ContractService) ParseContract(ctx context.Context, req *ParseContractRequest) (*models.ContractReview, error) {
	startTime := time.Now()

	global.Log.Info("开始解析合同", "file", req.FileName)

	// 1. 解析文档获取文本
	fileExt := getFileExtension(req.FileName)
	docParser, err := s.ParserFactory.GetParser(fileExt)
	if err != nil {
		return nil, fmt.Errorf("获取解析器失败: %w", err)
	}

	content, err := docParser.Parse(req.FilePath)
	if err != nil {
		return nil, fmt.Errorf("解析文档失败: %w", err)
	}

	parseTime := int(time.Since(startTime).Milliseconds())

	// 2. 提取结构
	structure, err := s.StructureExtractor.ExtractStructure(content)
	if err != nil {
		global.Log.Warn("提取合同结构失败", "error", err)
	}

	// 3. 提取关键字段
	fields, err := s.MetadataExtractor.ExtractAll(ctx, content)
	if err != nil {
		global.Log.Warn("提取关键字段失败", "error", err)
	}

	// 4. 提取表格
	tables, err := s.TableExtractor.ExtractTables(content)
	if err != nil {
		global.Log.Warn("提取表格失败", "error", err)
	}

	// 5. 序列化提取结果
	extractedContentJSON, _ := json.Marshal(structure)
	extractedFieldsJSON, _ := json.Marshal(fields)
	extractedTablesJSON, _ := json.Marshal(tables)

	// 6. 创建审核记录
	review := &models.ContractReview{
		ID:               utils.GenerateID(),
		Title:            req.FileName,
		ContractFilePath: req.FilePath,
		ContractFileName: req.FileName,
		ProcurementID:    req.ProcurementID,
		ExtractedContent: string(extractedContentJSON),
		ExtractedFields:  string(extractedFieldsJSON),
		ExtractedTables:  string(extractedTablesJSON),
		Status:           "parsed",
		ReviewProgress:   20,
		ParseTime:        parseTime,
		CreatedBy:        req.CreatedBy,
	}

	// 7. 保存到数据库
	_, err = global.DB.Insert(review)
	if err != nil {
		return nil, fmt.Errorf("保存审核记录失败: %w", err)
	}

	global.Log.Info("合同解析完成", "review_id", review.ID, "parse_time_ms", parseTime)

	return review, nil
}

// ReviewContract 智能审核合同
func (s *ContractService) ReviewContract(ctx context.Context, reviewID uint64) (*models.ContractReview, error) {
	startTime := time.Now()

	// 1. 获取审核记录
	var review models.ContractReview
	has, err := global.DB.ID(reviewID).Get(&review)
	if err != nil || !has {
		return nil, fmt.Errorf("审核记录不存在")
	}

	// 2. 更新状态为审核中
	review.Status = "reviewing"
	review.ReviewProgress = 30
	global.DB.ID(reviewID).Cols("status", "review_progress").Update(&review)

	// 3. 解析合同文本
	fileExt := getFileExtension(review.ContractFileName)
	docParser, err := s.ParserFactory.GetParser(fileExt)
	if err != nil {
		return nil, fmt.Errorf("获取解析器失败: %w", err)
	}

	content, err := docParser.Parse(review.ContractFilePath)
	if err != nil {
		return nil, fmt.Errorf("解析文档失败: %w", err)
	}

	// 3.1 基于合同文本做相似检索（RAG证据）
	var knowledgeContext []*models.DocumentChunk
	if embedder, e := embeddings.NewArkEmbedder(ctx); e == nil {
		search := NewVectorSearchEngine(embedder)
		if chunks, se := search.Search(content, 8); se == nil {
			knowledgeContext = chunks
		} else {
			global.Log.Warn("合同审核检索失败，忽略RAG", "error", se)
		}
	} else {
		global.Log.Warn("初始化嵌入器失败，忽略RAG", "error", e)
	}

	// 4. 获取采购需求数据（如果有关联）
	var procurementData *models.ProcurementRequirement
	if review.ProcurementID > 0 {
		procurementData = &models.ProcurementRequirement{}
		has, _ := global.DB.ID(review.ProcurementID).Get(procurementData)
		if !has {
			procurementData = nil
		}
	}

	// 5. 构建审核Prompt
	var extractedFields interface{}
	if review.ExtractedFields != "" {
		sonic.UnmarshalString(review.ExtractedFields, &extractedFields)
	}

	// 3.2 构建基础Prompt并注入知识库上下文（RAG）
	prompt := prompts.BuildContractReviewPrompt(content, extractedFields, procurementData)
	if len(knowledgeContext) > 0 {
		var b strings.Builder
		b.WriteString(prompt.UserPrompt)
		b.WriteString("\n# 参考知识库\n\n")
		b.WriteString("以下是从知识库检索到的相关标准/历史合同片段，请严格参考：\n\n")
		for i, chunk := range knowledgeContext {
			if i >= 5 {
				break
			}
			b.WriteString(fmt.Sprintf("## 参考文档 %d\n", i+1))
			if chunk.Title != "" {
				b.WriteString(fmt.Sprintf("标题: %s\n", chunk.Title))
			}
			// 简单截断
			snippet := chunk.Content
			if len([]rune(snippet)) > 600 {
				snippet = string([]rune(snippet)[:600]) + "\n...(截断)"
			}
			b.WriteString(fmt.Sprintf("内容: %s\n\n", snippet))
		}
		prompt.UserPrompt = b.String()
	}

	// 6. 调用LLM进行审核
	review.ReviewProgress = 50
	global.DB.ID(reviewID).Cols("review_progress").Update(&review)

	response, cost, err := s.LLMClient.Chat(ctx, prompt.SystemPrompt, prompt.UserPrompt)
	if err != nil {
		review.Status = "failed"
		global.DB.ID(reviewID).Cols("status").Update(&review)
		return nil, fmt.Errorf("LLM审核失败: %w", err)
	}

	// 7. 解析审核结果
	var reviewResult map[string]interface{}
	if err := sonic.UnmarshalString(response, &reviewResult); err != nil {
		return nil, fmt.Errorf("解析LLM响应失败: %w", err)
	}

	// 8. 提取审核结果
	review.ReviewProgress = 80

	if overall, ok := reviewResult["overall_assessment"].(map[string]interface{}); ok {
		if riskLevel, ok := overall["risk_level"].(string); ok {
			review.RiskLevel = riskLevel
		}
		if score, ok := overall["score"].(float64); ok {
			review.OverallScore = score
		}
	}

	if riskItems, ok := reviewResult["risk_items"].([]interface{}); ok {
		riskItemsJSON, _ := json.Marshal(riskItems)
		review.RiskItems = string(riskItemsJSON)
	}

	if consistency, ok := reviewResult["consistency_issues"].([]interface{}); ok {
		consistencyJSON, _ := json.Marshal(consistency)
		review.ConsistencyIssues = string(consistencyJSON)
	}

	if compliance, ok := reviewResult["compliance_issues"].([]interface{}); ok {
		complianceJSON, _ := json.Marshal(compliance)
		review.ComplianceIssues = string(complianceJSON)
	}

	if missing, ok := reviewResult["missing_clauses"].([]interface{}); ok {
		missingJSON, _ := json.Marshal(missing)
		review.MissingClauses = string(missingJSON)
	}

	if suggestions, ok := reviewResult["suggestions"].([]interface{}); ok {
		suggestionsJSON, _ := json.Marshal(suggestions)
		review.Suggestions = string(suggestionsJSON)
	}

	// 9. 更新审核结果
	review.Status = "completed"
	review.ReviewProgress = 100
	review.ReviewTime = int(time.Since(startTime).Milliseconds())
	review.LLMCost = cost
	review.LLMModel = global.Config.ChatConfig.ModelType

	_, err = global.DB.ID(reviewID).Update(&review)
	if err != nil {
		return nil, fmt.Errorf("更新审核结果失败: %w", err)
	}

	global.Log.Info("合同审核完成",
		"review_id", reviewID,
		"risk_level", review.RiskLevel,
		"score", review.OverallScore,
		"review_time_ms", review.ReviewTime,
		"llm_cost", cost,
	)

	return &review, nil
}

// CheckConsistency 一致性检查
func (s *ContractService) CheckConsistency(ctx context.Context, reviewID uint64) ([]models.ConsistencyIssue, error) {
	// 1. 获取审核记录
	var review models.ContractReview
	has, err := global.DB.ID(reviewID).Get(&review)
	if err != nil || !has {
		return nil, fmt.Errorf("审核记录不存在")
	}

	// 2. 获取采购需求数据
	if review.ProcurementID == 0 {
		return []models.ConsistencyIssue{}, nil // 没有关联采购需求，跳过一致性检查
	}

	var procurementData models.ProcurementRequirement
	has, err = global.DB.ID(review.ProcurementID).Get(&procurementData)
	if err != nil || !has {
		return nil, fmt.Errorf("采购需求不存在")
	}

	// 3. 解析提取的字段
	var extractedFields map[string]interface{}
	if err := sonic.UnmarshalString(review.ExtractedFields, &extractedFields); err != nil {
		return nil, fmt.Errorf("解析提取字段失败: %w", err)
	}

	// 4. 构建一致性检查Prompt
	prompt := prompts.BuildConsistencyCheckPrompt(extractedFields, &procurementData)

	// 5. 调用LLM进行一致性检查
	response, _, err := s.LLMClient.Chat(ctx, prompt.SystemPrompt, prompt.UserPrompt)
	if err != nil {
		return nil, fmt.Errorf("LLM一致性检查失败: %w", err)
	}

	// 6. 解析结果
	var result struct {
		Issues []models.ConsistencyIssue `json:"issues"`
	}
	if err := sonic.UnmarshalString(response, &result); err != nil {
		return nil, fmt.Errorf("解析一致性检查结果失败: %w", err)
	}

	return result.Issues, nil
}

// DetectRisks 风险检测
func (s *ContractService) DetectRisks(ctx context.Context, reviewID uint64) ([]models.RiskItem, error) {
	// 1. 获取审核记录
	var review models.ContractReview
	has, err := global.DB.ID(reviewID).Get(&review)
	if err != nil || !has {
		return nil, fmt.Errorf("审核记录不存在")
	}

	// 2. 解析风险项
	if review.RiskItems == "" || review.RiskItems == "[]" {
		return []models.RiskItem{}, nil
	}

	var risks []models.RiskItem
	if err := sonic.UnmarshalString(review.RiskItems, &risks); err != nil {
		return nil, fmt.Errorf("解析风险项失败: %w", err)
	}

	return risks, nil
}

// GenerateReport 生成审核报告
func (s *ContractService) GenerateReport(reviewID uint64) (map[string]interface{}, error) {
	// 1. 获取审核记录
	var review models.ContractReview
	has, err := global.DB.ID(reviewID).Get(&review)
	if err != nil || !has {
		return nil, fmt.Errorf("审核记录不存在")
	}

	// 2. 构建报告
	report := map[string]interface{}{
		"review_id":      review.ID,
		"contract_file":  review.ContractFileName,
		"review_date":    review.CreatedAt.Format("2006-01-02 15:04:05"),
		"status":         review.Status,
		"risk_level":     review.RiskLevel,
		"overall_score":  review.OverallScore,
		"parse_time_ms":  review.ParseTime,
		"review_time_ms": review.ReviewTime,
		"llm_cost":       review.LLMCost,
	}

	// 3. 解析JSON字段
	if review.RiskItems != "" {
		var risks []interface{}
		sonic.UnmarshalString(review.RiskItems, &risks)
		report["risk_items"] = risks
		report["risk_count"] = len(risks)
	}

	if review.ConsistencyIssues != "" {
		var consistency []interface{}
		sonic.UnmarshalString(review.ConsistencyIssues, &consistency)
		report["consistency_issues"] = consistency
	}

	if review.ComplianceIssues != "" {
		var compliance []interface{}
		sonic.UnmarshalString(review.ComplianceIssues, &compliance)
		report["compliance_issues"] = compliance
	}

	if review.MissingClauses != "" {
		var missing []interface{}
		sonic.UnmarshalString(review.MissingClauses, &missing)
		report["missing_clauses"] = missing
	}

	if review.Suggestions != "" {
		var suggestions []interface{}
		sonic.UnmarshalString(review.Suggestions, &suggestions)
		report["suggestions"] = suggestions
	}

	return report, nil
}

// getFileExtension 获取文件扩展名
func getFileExtension(filename string) string {
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			return filename[i:]
		}
	}
	return ""
}
