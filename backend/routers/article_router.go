package routers

import (
	"backend/api/article_api"
	"backend/global"
)

func (router RouterGroup) ArticleRouter() {
	// 创建文章API
	articleAPI, err := article_api.NewArticleAPI()
	if err != nil {
		global.Log.Error("创建文章API失败", "error", err)
		return
	}

	// 文章路由组
	articleRouter := router.Group("/article")
	{
		// 上传文章
		articleRouter.POST("/upload", articleAPI.UploadArticle)
		// 获取文章列表
		articleRouter.GET("/list", articleAPI.GetArticleList)
		// 获取文章详情
		articleRouter.GET("/:id", articleAPI.GetArticleDetail)
		// 删除文章
		articleRouter.DELETE("/:id", articleAPI.DeleteArticle)
		// 更新文章
		articleRouter.PUT("/:id", articleAPI.UpdateArticle)
		// 重新切分文章
		articleRouter.POST("/:id/reprocess", articleAPI.ReprocessArticle)
	}
}
