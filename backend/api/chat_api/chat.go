package chat_api

import (
	"backend/global"
	"backend/knowledge"
	"backend/models"
	"backend/models/res"
	"context"
	"fmt"
	"log/slog"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/hertz/pkg/app"
)

// NewChatAPI 创建聊天API
func NewChatAPI() (*ChatAPI, error) {
	knowledgeService, err := knowledge.NewKnowledgeService()
	if err != nil {
		return nil, err
	}

	return &ChatAPI{
		KnowledgeService: knowledgeService,
	}, nil
}

// ChatRequest 聊天请求
type ChatRequest struct {
	Query string `json:"query" binding:"required"`
}

// ChatResponse 聊天响应
type ChatResponse struct {
	Answer    string                  `json:"answer"`
	Documents []*models.DocumentChunk `json:"documents"` // 添加召回的文档信息
}

// Chat 聊天
func (api *ChatAPI) Chat(c context.Context, ctx *app.RequestContext) {
	var req ChatRequest
	if err := sonic.Unmarshal(ctx.Request.Body(), &req); err != nil {
		res.FailWithMessage("请求参数错误: "+err.Error(), c, ctx)
		global.Log.Error("请求参数错误", slog.Any("error", err))
		return
	}

	// 搜索相似文档
	documents, err := api.KnowledgeService.SearchSimilarDocuments(req.Query, 5)
	if err != nil {
		res.FailWithMessage("搜索相似文档失败: "+err.Error(), c, ctx)
		return
	}

	// 生成回答
	answer, err := api.KnowledgeService.GenerateAnswer(context.Background(), req.Query, documents)
	if err != nil {
		res.FailWithMessage("生成回答失败: "+err.Error(), c, ctx)
		return
	}

	// 返回成功响应
	response := ChatResponse{
		Answer:    answer,
		Documents: documents, // 添加召回的文档信息
	}

	res.OkWithData(response, c, ctx)
}

// ChatStreamRequest 流式聊天请求
type ChatStreamRequest struct {
	Query string `json:"query" binding:"required"`
}

// StreamEvent 流式事件
type StreamEvent struct {
	Type string      `json:"type"` // "documents", "token", "done", "error"
	Data interface{} `json:"data"`
}

// ChatStream 流式聊天
func (api *ChatAPI) ChatStream(c context.Context, ctx *app.RequestContext) {
	var req ChatStreamRequest
	if err := sonic.Unmarshal(ctx.Request.Body(), &req); err != nil {
		res.FailWithMessage("请求参数错误: "+err.Error(), c, ctx)
		global.Log.Error("请求参数错误", slog.Any("error", err))
		return
	}

	// 设置SSE响应头
	ctx.Response.Header.Set("Content-Type", "text/event-stream")
	ctx.Response.Header.Set("Cache-Control", "no-cache")
	ctx.Response.Header.Set("Connection", "keep-alive")
	ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")
	ctx.Response.Header.Set("Access-Control-Allow-Headers", "Cache-Control")

	// 确保响应立即发送
	ctx.Flush()

	// 搜索相似文档
	documents, err := api.KnowledgeService.SearchSimilarDocuments(req.Query, 5)
	if err != nil {
		api.sendSSEEvent(ctx, "error", map[string]string{"message": "搜索相似文档失败: " + err.Error()})
		return
	}

	// 发送文档信息
	api.sendSSEEvent(ctx, "documents", documents)

	// 生成流式回答
	err = api.KnowledgeService.GenerateAnswerStream(context.Background(), req.Query, documents, func(token string) {
		if token != "" {
			api.sendSSEEvent(ctx, "token", map[string]string{"content": token})
		}
	})

	if err != nil {
		api.sendSSEEvent(ctx, "error", map[string]string{"message": "生成回答失败: " + err.Error()})
		return
	}

	// 发送完成事件
	api.sendSSEEvent(ctx, "done", map[string]string{"message": "completed"})
}

// sendSSEEvent 发送SSE事件
func (api *ChatAPI) sendSSEEvent(ctx *app.RequestContext, eventType string, data interface{}) {
	event := StreamEvent{
		Type: eventType,
		Data: data,
	}

	// 使用sonic序列化
	jsonData, err := sonic.Marshal(event)
	if err != nil {
		global.Log.Error("序列化SSE事件失败", slog.Any("error", err))
		return
	}

	// 发送SSE格式的数据
	ctx.Response.BodyWriter().Write([]byte(fmt.Sprintf("data: %s\n\n", string(jsonData))))
	ctx.Flush()
}
