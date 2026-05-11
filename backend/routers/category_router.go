package routers

import (
	"backend/api/category_api"
	"backend/global"
)

// CategoryRouter 分类路由
func (router RouterGroup) CategoryRouter() {
	api := category_api.NewCategoryAPI()

	categoryGroup := router.Group("category")
	{
		categoryGroup.GET("list", api.GetCategoryList)   // 获取分类列表
		categoryGroup.POST("create", api.CreateCategory) // 创建分类
		categoryGroup.PUT(":id", api.UpdateCategory)     // 更新分类
		categoryGroup.GET("stats", api.GetCategoryStats) // 获取分类统计
	}

	global.Log.Info("分类路由注册成功")
}
