package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"backend/global"
)

// DeepSeekClient DeepSeek Chat客户端
type DeepSeekClient struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

// NewDeepSeekClient 创建DeepSeek客户端
func NewDeepSeekClient() *DeepSeekClient {
	timeout := 60 * time.Second
	if global.Config != nil && global.Config.ChatConfig != nil && global.Config.ChatConfig.RequestTimeout > 0 {
		timeout = global.Config.ChatConfig.RequestTimeout
	}
	return &DeepSeekClient{
		apiKey:  global.Config.ChatConfig.OwnerAPIKey,
		baseURL: global.Config.ChatConfig.BaseURL,
		model:   global.Config.ChatConfig.ModelType,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// ChatRequest DeepSeek Chat请求结构
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Stream      bool      `json:"stream"`
}

// Message 消息结构
type Message struct {
	Role    string `json:"role"` // system/user/assistant
	Content string `json:"content"`
}

// ChatResponse DeepSeek Chat响应结构
type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice 选择项
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage 使用量统计
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// Chat 调用DeepSeek Chat API
// 返回: (响应内容, LLM成本, 错误)
func (c *DeepSeekClient) Chat(ctx context.Context, systemPrompt, userPrompt string) (string, float64, error) {
	return c.ChatWithConfig(ctx, systemPrompt, userPrompt, 0.7, 4000)
}

// ChatWithConfig 带配置的Chat调用
func (c *DeepSeekClient) ChatWithConfig(ctx context.Context, systemPrompt, userPrompt string, temperature float64, maxTokens int) (string, float64, error) {
	// 构建请求
	messages := []Message{}
	if systemPrompt != "" {
		messages = append(messages, Message{
			Role:    "system",
			Content: systemPrompt,
		})
	}
	messages = append(messages, Message{
		Role:    "user",
		Content: userPrompt,
	})

	reqBody := ChatRequest{
		Model:       c.model,
		Messages:    messages,
		Temperature: temperature,
		MaxTokens:   maxTokens,
		Stream:      false,
	}

	// 序列化请求
	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return "", 0, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewBuffer(reqJSON))
	if err != nil {
		return "", 0, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	// 发送请求
	startTime := time.Now()
	resp, err := c.client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, fmt.Errorf("读取响应失败: %w", err)
	}

	duration := time.Since(startTime)
	global.Log.Info("DeepSeek API调用完成",
		"duration_ms", duration.Milliseconds(),
		"status_code", resp.StatusCode,
	)

	// 处理错误响应
	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
			return "", 0, fmt.Errorf("API错误 [%d]: %s", resp.StatusCode, errResp.Error.Message)
		}
		return "", 0, fmt.Errorf("API请求失败 [%d]: %s", resp.StatusCode, string(body))
	}

	// 解析成功响应
	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", 0, fmt.Errorf("解析响应失败: %w", err)
	}

	// 检查响应
	if len(chatResp.Choices) == 0 {
		return "", 0, fmt.Errorf("API返回空响应")
	}

	content := chatResp.Choices[0].Message.Content

	// 计算成本（DeepSeek定价：¥0.001/1K tokens）
	cost := float64(chatResp.Usage.TotalTokens) * 0.001 / 1000

	global.Log.Info("DeepSeek响应",
		"prompt_tokens", chatResp.Usage.PromptTokens,
		"completion_tokens", chatResp.Usage.CompletionTokens,
		"total_tokens", chatResp.Usage.TotalTokens,
		"cost_cny", fmt.Sprintf("¥%.6f", cost),
		"content_length", len(content),
	)

	return content, cost, nil
}

// ChatStream 流式调用（占位，待实现）
func (c *DeepSeekClient) ChatStream(ctx context.Context, systemPrompt, userPrompt string, callback func(string) error) (float64, error) {
	// TODO: 实现流式调用
	return 0, fmt.Errorf("流式调用暂未实现")
}

// ValidateAPIKey 验证API Key是否有效
func (c *DeepSeekClient) ValidateAPIKey(ctx context.Context) error {
	_, _, err := c.Chat(ctx, "", "测试连接")
	return err
}
