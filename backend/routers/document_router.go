package routers

import (
	"backend/api/document_api"
	"backend/global"
)

// DocumentRouter 文档路由
func (router RouterGroup) DocumentRouter() {
	api := document_api.NewDocumentAPI()

	documentGroup := router.Group("document")
	{
		// 文档解析相关
		documentGroup.POST("parse", api.DocumentParse)                           // 测试解析
		documentGroup.POST("quality-check", api.DocumentQualityCheck)            // 质量检查
		documentGroup.POST("parse-with-fallback", api.DocumentParseWithFallback) // 带降级的解析

		// 文档块手工编辑
		documentGroup.GET("chunks", api.ListChunks)               // ?article_id=
		documentGroup.PUT("chunks/:chunk_id", api.UpdateChunk)    // 更新并重嵌入
		documentGroup.POST("chunks/split", api.SplitChunk)        // 按“第X条”拆分
		documentGroup.POST("chunks/merge", api.MergeChunks)       // 合并
		documentGroup.POST("chunks/reorder", api.ReorderChunks)   // 重排
		documentGroup.DELETE("chunks/:chunk_id", api.DeleteChunk) // 删除
	}

	global.Log.Info("文档路由注册成功")
}
