package category_api

import (
	"backend/global"
	"backend/models"
)

// CategoryAPI 分类API
type CategoryAPI struct{}

// NewCategoryAPI 创建分类API实例
func NewCategoryAPI() *CategoryAPI {
	return &CategoryAPI{}
}

// getCategoryCount 获取某分类下的文档数量
func getCategoryCount(code string) (int64, error) {
	count, err := global.DB.Where("category_code = ?", code).Count(&models.Article{})
	if err != nil {
		global.Log.Error("获取分类文档数量失败", "category_code", code, "error", err)
		return 0, err
	}
	return count, nil
}
