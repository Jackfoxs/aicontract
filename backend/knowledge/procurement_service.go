package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"backend/global"
	"backend/llm"
	"backend/models"
	"backend/prompts"
	"backend/utils"

	"github.com/bytedance/sonic"
)

// ProcurementService 采购需求服务
type ProcurementService struct {
	SearchEngine *VectorSearchEngine
	LLMClient    *llm.DeepSeekClient // DeepSeek Chat客户端
}

// NewProcurementService 创建采购需求服务
func NewProcurementService(searchEngine *VectorSearchEngine) *ProcurementService {
	return &ProcurementService{
		SearchEngine: searchEngine,
		LLMClient:    llm.NewDeepSeekClient(),
	}
}

// AnalyzeRequirementRequest 需求分析请求
type AnalyzeRequirementRequest struct {
	RequirementText string  `json:"requirement_text"`
	DeviceType      string  `json:"device_type"`
	Department      string  `json:"department"`
	Budget          float64 `json:"budget"`
	CreatedBy       string  `json:"created_by"`
}

// AnalyzeRequirementResponse 需求分析响应
type AnalyzeRequirementResponse struct {
	RequirementID      uint64                   `json:"requirement_id"`
	DeviceInfo         map[string]interface{}   `json:"device_info"`
	TechnicalParams    []map[string]interface{} `json:"technical_parameters"`
	ComplianceReqs     []map[string]interface{} `json:"compliance_requirements"`
	Suggestions        []map[string]interface{} `json:"suggestions"`
	ReferenceStandards []map[string]interface{} `json:"reference_standards"`
	AnalysisQuality    float64                  `json:"analysis_quality"`
	ProcessingTime     int64                    `json:"processing_time"`
}

// AnalyzeRequirement 分析采购需求并生成技术参数
func (s *ProcurementService) AnalyzeRequirement(ctx context.Context, req *AnalyzeRequirementRequest) (*AnalyzeRequirementResponse, error) {
	startTime := time.Now()

	global.Log.Info("开始分析采购需求",
		"device_type", req.DeviceType,
		"department", req.Department,
	)

	// 1. 从知识库检索相关标准和案例
	categories := []string{"medical_std", "legal", "procurement_case"}
	knowledgeChunks, err := s.SearchEngine.SearchWithCategories(
		req.RequirementText,
		categories,
		10, // 检索Top 10
	)
	if err != nil {
		global.Log.Error("检索知识库失败", "error", err)
		return nil, fmt.Errorf("检索知识库失败: %w", err)
	}

	global.Log.Info("知识库检索完成", "chunks", len(knowledgeChunks))

	// 2. 构建Prompt
	prompt := prompts.BuildRequirementAnalysisPrompt(
		req.RequirementText,
		req.DeviceType,
		req.Department,
		req.Budget,
		knowledgeChunks,
	)

	// 3. 调用LLM生成参数
	llmResponse, llmCost, err := s.LLMClient.Chat(ctx, prompt.SystemPrompt, prompt.UserPrompt)
	if err != nil {
		global.Log.Error("LLM调用失败", "error", err)
		return nil, fmt.Errorf("LLM调用失败: %w", err)
	}

	global.Log.Info("LLM调用成功", "response_length", len(llmResponse))

	// 4. 解析LLM响应（JSON格式）
	var llmResult map[string]interface{}
	if err := sonic.UnmarshalString(llmResponse, &llmResult); err != nil {
		maxLen := 500
		if len(llmResponse) < maxLen {
			maxLen = len(llmResponse)
		}
		global.Log.Error("解析LLM响应失败", "error", err, "response", llmResponse[:maxLen])
		return nil, fmt.Errorf("解析LLM响应失败: %w", err)
	}

	// 5. 提取各部分数据
	deviceInfo, _ := llmResult["device_info"].(map[string]interface{})
	technicalParams, _ := llmResult["technical_parameters"].([]interface{})
	complianceReqs, _ := llmResult["compliance_requirements"].([]interface{})
	suggestions, _ := llmResult["suggestions"].([]interface{})
	referenceStandards, _ := llmResult["reference_standards"].([]interface{})

	// 6. 计算分析质量评分
	qualityScore := s.calculateQualityScore(llmResult, len(knowledgeChunks))

	// 7. 保存到数据库
	requirementID := utils.GenerateID()
	requirement := &models.ProcurementRequirement{
		ID:                requirementID,
		Title:             extractTitle(req, deviceInfo),
		RequirementText:   req.RequirementText,
		DeviceType:        req.DeviceType,
		Department:        req.Department,
		Budget:            req.Budget,
		GeneratedParams:   mustMarshalJSON(technicalParams),
		ComplianceIssues:  mustMarshalJSON(complianceReqs),
		Suggestions:       mustMarshalJSON(suggestions),
		HistoricalCases:   "[]", // TODO: 提取历史案例
		Status:            "generated",
		AnalysisQuality:   qualityScore,
		UsedKnowledgeBase: len(knowledgeChunks) > 0,
		RetrievalCount:    len(knowledgeChunks),
		LLMModel:          global.Config.ChatConfig.ModelType,
		LLMCost:           llmCost,
		ProcessingTime:    int(time.Since(startTime).Milliseconds()),
		CreatedBy:         req.CreatedBy,
	}

	_, err = global.DB.Insert(requirement)
	if err != nil {
		global.Log.Error("保存需求记录失败", "error", err)
		return nil, fmt.Errorf("保存需求记录失败: %w", err)
	}

	global.Log.Info("需求分析完成",
		"requirement_id", requirementID,
		"quality_score", qualityScore,
		"processing_time_ms", requirement.ProcessingTime,
	)

	// 8. 构建响应
	response := &AnalyzeRequirementResponse{
		RequirementID:      requirementID,
		DeviceInfo:         deviceInfo,
		TechnicalParams:    convertToMapSlice(technicalParams),
		ComplianceReqs:     convertToMapSlice(complianceReqs),
		Suggestions:        convertToMapSlice(suggestions),
		ReferenceStandards: convertToMapSlice(referenceStandards),
		AnalysisQuality:    qualityScore,
		ProcessingTime:     int64(requirement.ProcessingTime),
	}

	return response, nil
}

// VerifyParameters 校验已有技术参数
func (s *ProcurementService) VerifyParameters(ctx context.Context, deviceType string, parameters map[string]interface{}) (map[string]interface{}, error) {
	// 1. 检索相关标准
	categories := []string{"medical_std", "legal"}
	knowledgeChunks, err := s.SearchEngine.SearchWithCategories(
		deviceType,
		categories,
		5,
	)
	if err != nil {
		return nil, fmt.Errorf("检索知识库失败: %w", err)
	}

	// 2. 构建校验Prompt
	prompt := prompts.BuildParameterVerifyPrompt(deviceType, parameters, knowledgeChunks)

	// 3. 调用LLM校验
	llmResponse, _, err := s.LLMClient.Chat(ctx, prompt.SystemPrompt, prompt.UserPrompt)
	if err != nil {
		return nil, fmt.Errorf("LLM调用失败: %w", err)
	}

	// 4. 解析响应
	var result map[string]interface{}
	if err := sonic.UnmarshalString(llmResponse, &result); err != nil {
		return nil, fmt.Errorf("解析LLM响应失败: %w", err)
	}

	return result, nil
}

// GetHistoricalCases 获取历史采购案例
func (s *ProcurementService) GetHistoricalCases(ctx context.Context, deviceType string, limit int) ([]*models.DocumentChunk, error) {
	categories := []string{"procurement_case"}
	return s.SearchEngine.SearchWithCategories(deviceType, categories, limit)
}

// calculateQualityScore 计算分析质量评分
func (s *ProcurementService) calculateQualityScore(result map[string]interface{}, knowledgeCount int) float64 {
	score := 0.0

	// 1. 检查设备信息完整性 (20分)
	if deviceInfo, ok := result["device_info"].(map[string]interface{}); ok {
		if deviceInfo["device_type"] != nil {
			score += 5
		}
		if deviceInfo["device_name"] != nil {
			score += 5
		}
		if deviceInfo["category"] != nil {
			score += 5
		}
		if deviceInfo["usage_scenario"] != nil {
			score += 5
		}
	}

	// 2. 检查技术参数数量和质量 (40分)
	if params, ok := result["technical_parameters"].([]interface{}); ok {
		paramCount := 0
		for _, p := range params {
			if paramMap, ok := p.(map[string]interface{}); ok {
				if paramList, ok := paramMap["parameters"].([]interface{}); ok {
					paramCount += len(paramList)
				}
			}
		}
		if paramCount > 0 {
			score += minFloat(40, float64(paramCount)*2) // 每个参数2分，最多40分
		}
	}

	// 3. 检查合规要求 (20分)
	if compliance, ok := result["compliance_requirements"].([]interface{}); ok {
		score += minFloat(20, float64(len(compliance))*5) // 每项5分，最多20分
	}

	// 4. 检查参考标准 (10分)
	if standards, ok := result["reference_standards"].([]interface{}); ok {
		score += minFloat(10, float64(len(standards))*2) // 每项2分，最多10分
	}

	// 5. 知识库使用加分 (10分)
	if knowledgeCount > 0 {
		score += minFloat(10, float64(knowledgeCount))
	}

	return score / 100.0 // 归一化到0-1
}

// 辅助函数

func extractTitle(req *AnalyzeRequirementRequest, deviceInfo map[string]interface{}) string {
	if deviceInfo != nil {
		if name, ok := deviceInfo["device_name"].(string); ok && name != "" {
			return name + " 采购需求"
		}
	}
	if req.DeviceType != "" {
		return req.DeviceType + " 采购需求"
	}
	return "采购需求分析"
}

func mustMarshalJSON(v interface{}) string {
	if v == nil {
		return "[]"
	}
	data, err := json.Marshal(v)
	if err != nil {
		return "[]"
	}
	return string(data)
}

func convertToMapSlice(data interface{}) []map[string]interface{} {
	if data == nil {
		return []map[string]interface{}{}
	}
	if slice, ok := data.([]interface{}); ok {
		result := make([]map[string]interface{}, 0, len(slice))
		for _, item := range slice {
			if m, ok := item.(map[string]interface{}); ok {
				result = append(result, m)
			}
		}
		return result
	}
	return []map[string]interface{}{}
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
