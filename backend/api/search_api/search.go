package search_api

import (
	"context"
	"log/slog"
	"strconv"

	"backend/global"
	"backend/knowledge"
	"backend/models/res"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/hertz/pkg/app"
)

// NewSearchAPI 创建搜索API
func NewSearchAPI() (*SearchAPI, error) {
	knowledgeService, err := knowledge.NewKnowledgeService()
	if err != nil {
		return nil, err
	}

	return &SearchAPI{
		KnowledgeService: knowledgeService,
	}, nil
}

// SearchDocumentsRequest 搜索文档请求
type SearchDocumentsRequest struct {
	Query    string `json:"query" binding:"required" msg:"请填写搜索关键词"`
	Type     string `json:"type"`
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
}

// SearchDocumentItem 搜索结果项
type SearchDocumentItem struct {
	ID        string  `json:"id"` // 使用string避免JavaScript大整数精度丢失
	Title     string  `json:"title"`
	Type      string  `json:"type"`
	Content   string  `json:"content"`
	Relevance float64 `json:"relevance"`
	CreatedAt string  `json:"created_at"`
}

// SearchDocumentsResponse 搜索文档响应
type SearchDocumentsResponse struct {
	List     []SearchDocumentItem `json:"list"`
	Total    int64                `json:"total"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"pageSize"`
}

// SearchDocuments 搜索文档
func (api *SearchAPI) SearchDocuments(c context.Context, ctx *app.RequestContext) {
	var req SearchDocumentsRequest
	if err := sonic.Unmarshal(ctx.Request.Body(), &req); err != nil {
		res.FailWithError(err, &req, c, ctx)
		global.Log.Error("请求参数错误", slog.Any("error", err))
		return
	}

	// 设置默认值
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}

	// 搜索文档
	chunks, total, err := api.KnowledgeService.SearchDocuments(req.Query, req.Type, req.Page, req.PageSize)
	if err != nil {
		res.FailWithMessage("搜索文档失败: "+err.Error(), c, ctx)
		global.Log.Error("搜索文档失败", slog.Any("error", err))
		return
	}

	// 构建响应数据
	list := make([]SearchDocumentItem, len(chunks))
	for i, chunk := range chunks {
		// 获取文章信息以获取类型
		article, err := api.KnowledgeService.GetArticleByID(strconv.FormatUint(chunk.ArticleID, 10))
		articleType := ""
		if err == nil && article != nil {
			articleType = article.Type
		}

		// 计算相关性（这里简化处理，实际应该基于向量相似度）
		relevance := 0.8 // 默认相关性

		list[i] = SearchDocumentItem{
			ID:        strconv.FormatUint(chunk.ID, 10),
			Title:     chunk.Title,
			Type:      articleType,
			Content:   chunk.Content,
			Relevance: relevance,
			CreatedAt: chunk.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	response := SearchDocumentsResponse{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	res.OkWithData(response, c, ctx)
}
