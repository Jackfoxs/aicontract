package routers

import (
	"backend/api/search_api"
	"backend/global"
)

func (router RouterGroup) SearchRouter() {
	// 创建搜索API
	searchAPI, err := search_api.NewSearchAPI()
	if err != nil {
		global.Log.Error("创建搜索API失败", "error", err)
		return
	}

	// 搜索路由组
	searchRouter := router.Group("/search")
	{
		// 搜索文档
		searchRouter.POST("/documents", searchAPI.SearchDocuments)
	}
}
