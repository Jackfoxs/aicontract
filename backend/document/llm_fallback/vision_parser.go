package llm_fallback

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"backend/document/parser"
	"backend/global"
)

// VisionParser LLM视觉解析器（用于低质量PDF的降级处理）
type VisionParser struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

// NewVisionParser 创建LLM视觉解析器
func NewVisionParser(apiKey, baseURL, model string) *VisionParser {
	return &VisionParser{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// VisionParseResult 视觉解析结果
type VisionParseResult struct {
	Content      string  `json:"content"`
	Confidence   float64 `json:"confidence"`
	Cost         float64 `json:"cost"`
	TokensUsed   int     `json:"tokens_used"`
	ParseTimeMs  int64   `json:"parse_time_ms"`
	ErrorMessage string  `json:"error_message,omitempty"`
}

// ParsePDFWithVision 使用LLM视觉能力解析PDF
// 注意：这是一个较昂贵的操作，建议只在质量检测失败时使用
func (vp *VisionParser) ParsePDFWithVision(pdfPath string) (*VisionParseResult, error) {
	startTime := time.Now()

	// 1. 将PDF转换为图片（每页一张）
	images, err := vp.convertPDFToImages(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("PDF转图片失败: %w", err)
	}

	// 2. 对每页图片调用LLM Vision API
	var fullContent string
	var totalTokens int
	var totalCost float64

	for i, imagePath := range images {
		global.Log.Info("正在解析PDF页面", "page", i+1, "total", len(images))

		pageContent, tokens, cost, err := vp.parseImageWithLLM(imagePath)
		if err != nil {
			global.Log.Error("解析页面失败", "page", i+1, "error", err)
			continue
		}

		fullContent += fmt.Sprintf("\n--- 第 %d 页 ---\n%s\n", i+1, pageContent)
		totalTokens += tokens
		totalCost += cost

		// 清理临时图片
		os.Remove(imagePath)
	}

	parseTime := time.Since(startTime).Milliseconds()

	return &VisionParseResult{
		Content:     fullContent,
		Confidence:  0.95, // LLM视觉解析通常有较高置信度
		Cost:        totalCost,
		TokensUsed:  totalTokens,
		ParseTimeMs: parseTime,
	}, nil
}

// convertPDFToImages 将PDF转换为图片
// TODO: 实现PDF转图片功能（需要依赖 poppler-utils 或 Go图片库）
func (vp *VisionParser) convertPDFToImages(pdfPath string) ([]string, error) {
	// 这里先返回占位符，实际需要使用以下方案之一：
	// 1. 使用 exec.Command 调用 pdftoppm (需要安装 poppler-utils)
	// 2. 使用 Go 的 PDF 库 + 图片处理库
	// 3. 使用第三方API服务

	return nil, fmt.Errorf("PDF转图片功能暂未实现，需要安装额外依赖")
}

// parseImageWithLLM 使用LLM解析单张图片
func (vp *VisionParser) parseImageWithLLM(imagePath string) (content string, tokens int, cost float64, err error) {
	// 读取图片并转为Base64
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		return "", 0, 0, fmt.Errorf("读取图片失败: %w", err)
	}

	imageBase64 := base64.StdEncoding.EncodeToString(imageData)

	// 构建请求（OpenAI格式，兼容DeepSeek等）
	requestBody := map[string]interface{}{
		"model": vp.model,
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": "请准确地提取这张图片中的所有文字内容。保持原有的格式、段落和结构。如果有表格，请用Markdown表格格式输出。",
					},
					{
						"type": "image_url",
						"image_url": map[string]string{
							"url": fmt.Sprintf("data:image/png;base64,%s", imageBase64),
						},
					},
				},
			},
		},
		"max_tokens": 4096,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", 0, 0, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 发送请求
	req, err := http.NewRequest("POST", vp.baseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", 0, 0, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+vp.apiKey)

	resp, err := vp.client.Do(req)
	if err != nil {
		return "", 0, 0, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", 0, 0, fmt.Errorf("API返回错误 (状态码 %d): %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return "", 0, 0, fmt.Errorf("解析响应失败: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return "", 0, 0, fmt.Errorf("API返回空响应")
	}

	// 计算成本（示例：按DeepSeek价格 ¥0.001/1K tokens）
	tokens = apiResp.Usage.TotalTokens
	cost = float64(tokens) / 1000 * 0.001

	return apiResp.Choices[0].Message.Content, tokens, cost, nil
}

// FallbackParseStrategy 降级解析策略
type FallbackParseStrategy struct {
	qualityThreshold float64 // 质量阈值（低于此值触发降级）
	visionParser     *VisionParser
}

// NewFallbackParseStrategy 创建降级解析策略
func NewFallbackParseStrategy(qualityThreshold float64, apiKey, baseURL, model string) *FallbackParseStrategy {
	return &FallbackParseStrategy{
		qualityThreshold: qualityThreshold,
		visionParser:     NewVisionParser(apiKey, baseURL, model),
	}
}

// ParseWithFallback 使用降级策略解析文档
// 1. 先尝试原生解析
// 2. 检测质量
// 3. 如果质量低于阈值，使用LLM视觉解析
func (fps *FallbackParseStrategy) ParseWithFallback(filePath string, fileType string) (*parser.ParseResult, error) {
	// 1. 原生解析
	factory := parser.NewParserFactory()
	nativeParser, err := factory.GetParser(fileType)
	if err != nil {
		return nil, err
	}

	result, err := nativeParser.ParseWithMetadata(filePath)
	if err != nil {
		return nil, fmt.Errorf("原生解析失败: %w", err)
	}

	// 2. 检查质量
	if result.QualityScore >= fps.qualityThreshold {
		global.Log.Info("文档质量良好，使用原生解析结果",
			"file", filePath,
			"quality_score", result.QualityScore,
			"threshold", fps.qualityThreshold,
		)
		return result, nil
	}

	// 3. 质量不佳，尝试LLM视觉解析（仅支持PDF）
	if fileType != "pdf" && fileType != ".pdf" {
		global.Log.Warn("文档质量较低但不支持视觉降级",
			"file_type", fileType,
			"quality_score", result.QualityScore,
		)
		return result, nil
	}

	global.Log.Warn("文档质量较低，尝试LLM视觉解析",
		"file", filePath,
		"quality_score", result.QualityScore,
		"threshold", fps.qualityThreshold,
	)

	// 调用视觉解析
	visionResult, err := fps.visionParser.ParsePDFWithVision(filePath)
	if err != nil {
		global.Log.Error("LLM视觉解析失败，返回原生解析结果", "error", err)
		return result, nil // 降级失败，返回原生结果
	}

	// 使用视觉解析结果替换内容
	result.Content = visionResult.Content
	result.QualityScore = visionResult.Confidence
	result.ParseMethod = "llm_vision_fallback"

	global.Log.Info("LLM视觉解析成功",
		"cost", visionResult.Cost,
		"tokens", visionResult.TokensUsed,
		"time_ms", visionResult.ParseTimeMs,
	)

	return result, nil
}
