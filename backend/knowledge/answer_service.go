package knowledge

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"backend/global"
	"backend/models"
)

// GenerateAnswer 生成回答
func (s *KnowledgeServiceImpl) GenerateAnswer(ctx context.Context, query string, documents []*models.DocumentChunk) (string, error) {
	// 临时Mock响应 - 用于测试API结构
	// TODO: 配置有效的DeepSeek API Key后移除此部分
	if global.Config.ChatConfig.OwnerAPIKey == "sk-653a1dbeaff94852b46bcf5230284bce" {
		mockAnswer := fmt.Sprintf("这是一个模拟回答，用于测试API结构。您的问题是：%s", query)
		if len(documents) > 0 {
			mockAnswer += "\n\n根据知识库检索到的相关文档："
			for i, doc := range documents {
				mockAnswer += fmt.Sprintf("\n[%d] %s", i+1, doc.Title)
			}
		} else {
			mockAnswer += "\n\n暂未检索到相关文档，请先上传相关知识文档。"
		}
		return mockAnswer, nil
	}

	// 构建提示词
	systemPrompt := global.Config.ChatConfig.SystemDefault + "\n5.引用来源——在回答中明确引用知识库中的信息来源，标注引用的文档标题。"
	userPrompt := strings.Replace(global.Config.ChatConfig.UserDefault, "{question}", query, -1)

	// 构建文档内容
	docContent := ""
	docSources := []string{}
	for i, doc := range documents {
		docID := fmt.Sprintf("[%d]", i+1)
		docContent += fmt.Sprintf("## %s %s\n%s\n\n", docID, doc.Title, doc.Content)
		docSources = append(docSources, fmt.Sprintf("%s %s", docID, doc.Title))
	}

	// 构建引用指南
	citationGuide := "\n\n请在回答中引用相关文档的编号，例如[1]、[2]等，以便用户了解信息来源。"
	if len(docSources) > 0 {
		citationGuide += "\n\n参考文档列表:\n" + strings.Join(docSources, "\n")
	}

	// 构建请求体
	reqBody := map[string]interface{}{
		"model": global.Config.ChatConfig.ModelType,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": systemPrompt,
			},
			{
				"role":    "user",
				"content": userPrompt + "\n\n参考文档:\n" + docContent + citationGuide,
			},
		},
	}

	// 序列化请求体
	reqData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("序列化请求体失败: %w", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequest("POST", global.Config.ChatConfig.BaseURL+"/chat/completions", strings.NewReader(string(reqData)))
	if err != nil {
		return "", fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+global.Config.ChatConfig.OwnerAPIKey)

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	// 解析响应
	var respBody map[string]interface{}
	if err := json.Unmarshal(respData, &respBody); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	// 提取回答
	choices, ok := respBody["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", errors.New("响应格式错误")
	}

	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return "", errors.New("响应格式错误")
	}

	message, ok := choice["message"].(map[string]interface{})
	if !ok {
		return "", errors.New("响应格式错误")
	}

	content, ok := message["content"].(string)
	if !ok {
		return "", errors.New("响应格式错误")
	}

	// 如果文档列表不为空，确保在回答中包含文档来源信息
	if len(docSources) > 0 && !strings.Contains(content, "参考文档列表") {
		// 检查回答中是否已经包含了文档引用信息
		if !strings.Contains(content, "参考文档列表") {
			content += "\n\n参考文档列表:\n" + strings.Join(docSources, "\n")
		}
	}

	return content, nil
}

// GenerateAnswerStream 生成流式回答
func (s *KnowledgeServiceImpl) GenerateAnswerStream(ctx context.Context, query string, documents []*models.DocumentChunk, onToken func(string)) error {
	// 临时Mock流式响应 - 用于测试API结构
	// TODO: 配置有效的DeepSeek API Key后移除此部分
	if global.Config.ChatConfig.OwnerAPIKey == "sk-653a1dbeaff94852b46bcf5230284bce" {
		mockAnswer := fmt.Sprintf("这是一个模拟的流式回答，用于测试API结构。您的问题是：%s", query)
		if len(documents) > 0 {
			mockAnswer += "\n\n根据知识库检索到的相关文档："
			for i, doc := range documents {
				mockAnswer += fmt.Sprintf("\n[%d] %s", i+1, doc.Title)
			}
		} else {
			mockAnswer += "\n\n暂未检索到相关文档，请先上传相关知识文档。"
		}

		// 模拟流式输出，逐字符发送
		for i, char := range mockAnswer {
			onToken(string(char))
			// 模拟打字机效果
			if i%10 == 0 {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
					// 非阻塞延时
				}
			}
		}
		return nil
	}

	// 构建提示词
	systemPrompt := global.Config.ChatConfig.SystemDefault + "\n5.引用来源——在回答中明确引用知识库中的信息来源，标注引用的文档标题。"
	userPrompt := strings.Replace(global.Config.ChatConfig.UserDefault, "{question}", query, -1)

	// 构建文档内容
	docContent := ""
	docSources := []string{}
	for i, doc := range documents {
		docID := fmt.Sprintf("[%d]", i+1)
		docContent += fmt.Sprintf("## %s %s\n%s\n\n", docID, doc.Title, doc.Content)
		docSources = append(docSources, fmt.Sprintf("%s %s", docID, doc.Title))
	}

	// 构建引用指南
	citationGuide := "\n\n请在回答中引用相关文档的编号，例如[1]、[2]等，以便用户了解信息来源。"
	if len(docSources) > 0 {
		citationGuide += "\n\n参考文档列表:\n" + strings.Join(docSources, "\n")
	}

	// 构建请求体，添加stream参数
	reqBody := map[string]interface{}{
		"model":  global.Config.ChatConfig.ModelType,
		"stream": true, // 启用流式输出
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": systemPrompt,
			},
			{
				"role":    "user",
				"content": userPrompt + "\n\n参考文档:\n" + docContent + citationGuide,
			},
		},
	}

	// 序列化请求体
	reqData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("序列化请求体失败: %w", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequest("POST", global.Config.ChatConfig.BaseURL+"/chat/completions", strings.NewReader(string(reqData)))
	if err != nil {
		return fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+global.Config.ChatConfig.OwnerAPIKey)
	req.Header.Set("Accept", "text/event-stream")

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取流式响应
	return s.processStreamResponse(resp.Body, onToken)
}

// processStreamResponse 处理流式响应
func (s *KnowledgeServiceImpl) processStreamResponse(body io.Reader, onToken func(string)) error {
	reader := bufio.NewReader(body)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("读取流式响应失败: %w", err)
		}

		line = strings.TrimSpace(line)

		// 跳过空行和注释行
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		// 处理SSE数据
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			// 检查是否为结束标记
			if data == "[DONE]" {
				break
			}

			// 解析JSON数据
			var streamResp map[string]interface{}
			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				continue // 跳过无法解析的数据
			}

			// 提取token
			choices, ok := streamResp["choices"].([]interface{})
			if !ok || len(choices) == 0 {
				continue
			}

			choice, ok := choices[0].(map[string]interface{})
			if !ok {
				continue
			}

			delta, ok := choice["delta"].(map[string]interface{})
			if !ok {
				continue
			}

			content, ok := delta["content"].(string)
			if ok && content != "" {
				onToken(content)
			}
		}
	}

	return nil
}
