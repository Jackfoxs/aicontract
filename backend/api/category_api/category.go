package category_api

import (
	"context"
	"strconv"

	"backend/global"
	"backend/models"
	"backend/models/res"
	"backend/utils"

	"github.com/cloudwego/hertz/pkg/app"
)

// CategoryListResponse 分类列表响应
type CategoryListResponse struct {
	ID          string `json:"id"` // 使用string避免JavaScript大整数精度丢失
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	SortOrder   int    `json:"sort_order"`
	DocCount    int64  `json:"doc_count"` // 文档数量
	CreatedAt   string `json:"created_at"`
}

// CategoryListRequest 分类列表请求
type CategoryListRequest struct {
	Keyword string `form:"keyword" json:"keyword"` // 搜索关键词
}

// GetCategoryList 获取分类列表
// @Summary 获取文档分类列表
// @Description 获取所有文档分类，包含每个分类下的文档数量
// @Tags 分类管理
// @Accept json
// @Produce json
// @Param keyword query string false "搜索关键词"
// @Success 200 {object} res.Response
// @Router /api/category/list [get]
func (api *CategoryAPI) GetCategoryList(c context.Context, ctx *app.RequestContext) {
	var req CategoryListRequest
	if err := ctx.BindAndValidate(&req); err != nil {
		res.FailWithMessage("参数错误: "+err.Error(), c, ctx)
		return
	}

	// 查询分类
	var categories []models.DocumentCategory
	session := global.DB.NewSession()
	defer session.Close()

	if req.Keyword != "" {
		session.Where("name LIKE ? OR code LIKE ? OR description LIKE ?",
			"%"+req.Keyword+"%", "%"+req.Keyword+"%", "%"+req.Keyword+"%")
	}

	session.OrderBy("sort_order ASC, id ASC")

	err := session.Find(&categories)
	if err != nil {
		global.Log.Error("查询分类列表失败", "error", err)
		res.FailWithMessage("查询分类列表失败", c, ctx)
		return
	}

	// 构建响应
	var responseList []CategoryListResponse
	for _, category := range categories {
		// 获取分类下的文档数量
		docCount, _ := getCategoryCount(category.Code)

		responseList = append(responseList, CategoryListResponse{
			ID:          strconv.FormatUint(category.ID, 10),
			Code:        category.Code,
			Name:        category.Name,
			Description: category.Description,
			Icon:        category.Icon,
			SortOrder:   category.SortOrder,
			DocCount:    docCount,
			CreatedAt:   category.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	res.OkWithData(responseList, c, ctx)
}

// CreateCategoryRequest 创建分类请求
type CreateCategoryRequest struct {
	Code        string `json:"code" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	SortOrder   int    `json:"sort_order"`
}

// CreateCategory 创建分类
// @Summary 创建文档分类
// @Description 创建新的文档分类（管理员功能）
// @Tags 分类管理
// @Accept json
// @Produce json
// @Param body body CreateCategoryRequest true "分类信息"
// @Success 200 {object} res.Response
// @Router /api/category/create [post]
func (api *CategoryAPI) CreateCategory(c context.Context, ctx *app.RequestContext) {
	var req CreateCategoryRequest
	if err := ctx.BindAndValidate(&req); err != nil {
		res.FailWithMessage("参数错误: "+err.Error(), c, ctx)
		return
	}

	// 检查代码是否已存在
	count, err := global.DB.Where("code = ?", req.Code).Count(&models.DocumentCategory{})
	if err != nil {
		global.Log.Error("检查分类代码失败", "error", err)
		res.FailWithMessage("检查分类代码失败", c, ctx)
		return
	}

	if count > 0 {
		res.FailWithMessage("分类代码已存在", c, ctx)
		return
	}

	// 创建分类
	category := &models.DocumentCategory{
		ID:          utils.GenerateID(),
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
		Icon:        req.Icon,
		SortOrder:   req.SortOrder,
	}

	_, err = global.DB.Insert(category)
	if err != nil {
		global.Log.Error("创建分类失败", "error", err)
		res.FailWithMessage("创建分类失败", c, ctx)
		return
	}

	res.OkWithData(map[string]interface{}{
		"id": strconv.FormatUint(category.ID, 10),
	}, c, ctx)
}

// UpdateCategoryRequest 更新分类请求
type UpdateCategoryRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	SortOrder   int    `json:"sort_order"`
}

// UpdateCategory 更新分类
// @Summary 更新文档分类
// @Description 更新文档分类信息（管理员功能）
// @Tags 分类管理
// @Accept json
// @Produce json
// @Param id path string true "分类ID"
// @Param body body UpdateCategoryRequest true "分类信息"
// @Success 200 {object} res.Response
// @Router /api/category/{id} [put]
func (api *CategoryAPI) UpdateCategory(c context.Context, ctx *app.RequestContext) {
	id := ctx.Param("id")
	if id == "" {
		res.FailWithMessage("分类ID不能为空", c, ctx)
		return
	}

	var req UpdateCategoryRequest
	if err := ctx.BindAndValidate(&req); err != nil {
		res.FailWithMessage("参数错误: "+err.Error(), c, ctx)
		return
	}

	// 更新分类
	category := &models.DocumentCategory{}
	_, err := global.DB.Where("id = ?", id).Get(category)
	if err != nil {
		global.Log.Error("查询分类失败", "error", err)
		res.FailWithMessage("查询分类失败", c, ctx)
		return
	}

	if category.ID == 0 {
		res.FailWithMessage("分类不存在", c, ctx)
		return
	}

	// 更新字段
	if req.Name != "" {
		category.Name = req.Name
	}
	if req.Description != "" {
		category.Description = req.Description
	}
	if req.Icon != "" {
		category.Icon = req.Icon
	}
	category.SortOrder = req.SortOrder

	_, err = global.DB.Where("id = ?", id).Update(category)
	if err != nil {
		global.Log.Error("更新分类失败", "error", err)
		res.FailWithMessage("更新分类失败", c, ctx)
		return
	}

	res.OkWithMessage("更新成功", c, ctx)
}

// GetCategoryStats 获取分类统计
// @Summary 获取分类统计信息
// @Description 获取所有分类的文档数量统计
// @Tags 分类管理
// @Produce json
// @Success 200 {object} res.Response
// @Router /api/category/stats [get]
func (api *CategoryAPI) GetCategoryStats(c context.Context, ctx *app.RequestContext) {
	var categories []models.DocumentCategory
	err := global.DB.OrderBy("sort_order ASC").Find(&categories)
	if err != nil {
		global.Log.Error("查询分类失败", "error", err)
		res.FailWithMessage("查询分类失败", c, ctx)
		return
	}

	stats := make(map[string]interface{})
	var totalDocs int64

	for _, category := range categories {
		count, _ := getCategoryCount(category.Code)
		stats[category.Code] = map[string]interface{}{
			"name":      category.Name,
			"doc_count": count,
			"icon":      category.Icon,
		}
		totalDocs += count
	}

	// 未分类文档数量
	uncategorized, _ := global.DB.Where("category_code = '' OR category_code IS NULL").Count(&models.Article{})
	stats["uncategorized"] = map[string]interface{}{
		"name":      "未分类",
		"doc_count": uncategorized,
		"icon":      "📁",
	}
	totalDocs += uncategorized

	res.OkWithData(map[string]interface{}{
		"stats":      stats,
		"total_docs": totalDocs,
	}, c, ctx)
}
