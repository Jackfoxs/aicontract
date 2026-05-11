package article_api

import (
	"backend/global"
	"backend/knowledge"
	"backend/models"
	"backend/models/res"
	"context"
	"os"
	"strconv"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/hertz/pkg/app"
)

// NewArticleAPI 创建文章API
func NewArticleAPI() (*ArticleAPI, error) {
	knowledgeService, err := knowledge.NewKnowledgeService()
	if err != nil {
		return nil, err
	}

	return &ArticleAPI{
		KnowledgeService: knowledgeService,
	}, nil
}

// UploadArticleRequest 上传文章请求
type UploadArticleRequest struct {
	Title        string `form:"title" binding:"required" msg:"请填写标题" json:"title"`
	Type         string `form:"type" binding:"required" msg:"请填写类型" json:"type"`
	Content      string `form:"content" json:"content"`
	CategoryCode string `form:"category_code" json:"category_code"` // 文档分类代码
}

// UploadArticleResponse 上传文章响应
type UploadArticleResponse struct {
	ArticleID string `json:"article_id"` // 使用string避免JavaScript大整数精度丢失
}

// GetArticleListRequest 获取文章列表请求
type GetArticleListRequest struct {
	Page     int    `form:"page" json:"page"`
	PageSize int    `form:"pageSize" json:"pageSize"`
	Keyword  string `form:"keyword" json:"keyword"`
	Type     string `form:"type" json:"type"`
}

// ArticleListItem 文章列表项
type ArticleListItem struct {
	ID            string `json:"id"` // 使用string避免JavaScript大整数精度丢失
	Title         string `json:"title"`
	Type          string `json:"type"`
	Content       string `json:"content"`
	HasAttachment bool   `json:"has_attachment"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

// GetArticleListResponse 获取文章列表响应
type GetArticleListResponse struct {
	List     []ArticleListItem `json:"list"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"pageSize"`
}

// GetArticleDetailResponse 获取文章详情响应
type GetArticleDetailResponse struct {
	ID                string `json:"id"` // 使用string避免JavaScript大整数精度丢失
	Title             string `json:"title"`
	Type              string `json:"type"`
	Content           string `json:"content"`
	Attachment        string `json:"attachment"`
	AttachmentContent string `json:"attachment_content"`
	HasAttachment     bool   `json:"has_attachment"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
}

// UpdateArticleRequest 更新文章请求
type UpdateArticleRequest struct {
	Title   string `json:"title" binding:"required" msg:"请填写标题"`
	Type    string `json:"type" binding:"required" msg:"请填写类型"`
	Content string `json:"content"`
}

// UploadArticle 上传文章
func (api *ArticleAPI) UploadArticle(c context.Context, ctx *app.RequestContext) {
	var req UploadArticleRequest

	// 获取表单参数
	req.Title = string(ctx.FormValue("title"))
	req.Type = string(ctx.FormValue("type"))
	req.Content = string(ctx.FormValue("content"))
	req.CategoryCode = string(ctx.FormValue("category_code"))

	if req.Title == "" || req.Type == "" {
		res.FailWithMessage("请填写标题和类型", c, ctx)
		return
	}

	// 获取上传的文件
	file, err := ctx.FormFile("attachment")
	var attachment *os.File
	var attachmentName string

	if err == nil && file != nil {
		// 创建临时文件
		attachment, err = os.CreateTemp("", "upload-*")
		if err != nil {
			res.FailWithMessage("创建临时文件失败: "+err.Error(), c, ctx)
			global.Log.Error("创建临时文件失败", "error", err)
			return
		}
		defer attachment.Close()
		defer os.Remove(attachment.Name())

		// 打开上传的文件
		src, err := file.Open()
		if err != nil {
			res.FailWithMessage("打开上传文件失败: "+err.Error(), c, ctx)
			global.Log.Error("打开上传文件失败", "error", err)
			return
		}
		defer src.Close()

		// 将上传的文件内容复制到临时文件
		if _, err := attachment.ReadFrom(src); err != nil {
			res.FailWithMessage("保存上传文件失败: "+err.Error(), c, ctx)
			global.Log.Error("保存上传文件失败", "error", err)
			return
		}

		// 重置文件指针到开头
		if _, err := attachment.Seek(0, 0); err != nil {
			res.FailWithMessage("重置文件指针失败: "+err.Error(), c, ctx)
			global.Log.Error("重置文件指针失败", "error", err)
			return
		}

		attachmentName = file.Filename
	}

	// 上传文章
	articleID, err := api.KnowledgeService.UploadArticle(req.Title, req.Type, req.Content, req.CategoryCode, attachment, attachmentName)
	if err != nil {
		res.FailWithMessage("上传文章失败: "+err.Error(), c, ctx)
		global.Log.Error("上传文章失败", "error", err)
		return
	}

	// 返回成功响应
	response := UploadArticleResponse{
		ArticleID: strconv.FormatUint(articleID, 10),
	}

	res.OkWithData(response, c, ctx)
}

// GetArticleList 获取文章列表
func (api *ArticleAPI) GetArticleList(c context.Context, ctx *app.RequestContext) {
	var req GetArticleListRequest
	if err := ctx.BindAndValidate(&req); err != nil {
		res.FailWithError(err, &req, c, ctx)
		global.Log.Error("请求参数错误", "error", err)
		return
	}

	// 设置默认值
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}

	// 获取文章列表
	articles, total, err := api.KnowledgeService.GetArticleList(req.Page, req.PageSize, req.Keyword, req.Type)
	if err != nil {
		res.FailWithMessage("获取文章列表失败: "+err.Error(), c, ctx)
		global.Log.Error("获取文章列表失败", "error", err)
		return
	}

	// 构建响应数据
	list := make([]ArticleListItem, len(articles))
	for i, article := range articles {
		list[i] = ArticleListItem{
			ID:            strconv.FormatUint(article.ID, 10),
			Title:         article.Title,
			Type:          article.Type,
			Content:       article.Content,
			HasAttachment: article.HasAttachment,
			CreatedAt:     article.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt:     article.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	response := GetArticleListResponse{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	res.OkWithData(response, c, ctx)
}

// GetArticleDetail 获取文章详情
func (api *ArticleAPI) GetArticleDetail(c context.Context, ctx *app.RequestContext) {
	id := ctx.Param("id")
	if id == "" {
		res.FailWithMessage("文章ID不能为空", c, ctx)
		return
	}

	// 获取文章详情
	article, err := api.KnowledgeService.GetArticleByID(id)
	if err != nil {
		res.FailWithMessage("获取文章详情失败: "+err.Error(), c, ctx)
		global.Log.Error("获取文章详情失败", "error", err)
		return
	}

	if article == nil {
		res.FailWithMessage("文章不存在", c, ctx)
		return
	}

	response := GetArticleDetailResponse{
		ID:                strconv.FormatUint(article.ID, 10),
		Title:             article.Title,
		Type:              article.Type,
		Content:           article.Content,
		Attachment:        article.Attachment,
		AttachmentContent: article.AttachmentContent,
		HasAttachment:     article.HasAttachment,
		CreatedAt:         article.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:         article.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	res.OkWithData(response, c, ctx)
}

// DeleteArticle 删除文章
func (api *ArticleAPI) DeleteArticle(c context.Context, ctx *app.RequestContext) {
	id := ctx.Param("id")
	if id == "" {
		res.FailWithMessage("文章ID不能为空", c, ctx)
		return
	}

	// 删除文章
	err := api.KnowledgeService.DeleteArticle(id)
	if err != nil {
		res.FailWithMessage("删除文章失败: "+err.Error(), c, ctx)
		global.Log.Error("删除文章失败", "error", err)
		return
	}

	res.OkWithData(map[string]any{}, c, ctx)
}

// UpdateArticle 更新文章
func (api *ArticleAPI) UpdateArticle(c context.Context, ctx *app.RequestContext) {
	id := ctx.Param("id")
	if id == "" {
		res.FailWithMessage("文章ID不能为空", c, ctx)
		return
	}

	var req UpdateArticleRequest
	if err := sonic.Unmarshal(ctx.Request.Body(), &req); err != nil {
		res.FailWithError(err, &req, c, ctx)
		global.Log.Error("请求参数错误", "error", err)
		return
	}

	// 更新文章
	err := api.KnowledgeService.UpdateArticle(id, req.Title, req.Type, req.Content)
	if err != nil {
		res.FailWithMessage("更新文章失败: "+err.Error(), c, ctx)
		global.Log.Error("更新文章失败", "error", err)
		return
	}

	res.OkWithData(map[string]any{}, c, ctx)
}

// ReprocessArticle 重新按当前类型/分类切分并重建向量
func (api *ArticleAPI) ReprocessArticle(c context.Context, ctx *app.RequestContext) {
	id := ctx.Param("id")
	if id == "" {
		res.FailWithMessage("文章ID不能为空", c, ctx)
		return
	}
	article, err := api.KnowledgeService.GetArticleByID(id)
	if err != nil || article == nil {
		res.FailWithMessage("文章不存在", c, ctx)
		return
	}
	// 删除旧chunks与vectors
	// 查出所有chunk
	var chunks []models.DocumentChunk
	if err := global.DB.Where("article_id = ?", article.ID).Find(&chunks); err != nil {
		res.FailWithMessage("查询文档块失败: "+err.Error(), c, ctx)
		return
	}
	for _, ch := range chunks {
		// 删向量
		if _, err := global.DB.Where("chunk_id = ?", ch.ChunkID).Delete(&models.Vector{}); err != nil {
			res.FailWithMessage("清理向量失败: "+err.Error(), c, ctx)
			return
		}
	}
	if _, err := global.DB.Where("article_id = ?", article.ID).Delete(&models.DocumentChunk{}); err != nil {
		res.FailWithMessage("清理文档块失败: "+err.Error(), c, ctx)
		return
	}
	// 重新处理
	if err := api.KnowledgeService.ProcessDocument(article); err != nil {
		res.FailWithMessage("重新处理失败: "+err.Error(), c, ctx)
		return
	}
	res.OkWithData(map[string]any{}, c, ctx)
}
