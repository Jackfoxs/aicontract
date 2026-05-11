package routers

import (
	"backend/api/chat_api"
	"backend/global"
)

func (router RouterGroup) ChatRouter() {
	// 创建聊天API
	chatAPI, err := chat_api.NewChatAPI()
	if err != nil {
		global.Log.Error("创建聊天API失败", "error", err)
		return
	}

	// 聊天路由组
	chatRouter := router.Group("/chat")
	{
		// 问答接口
		chatRouter.POST("/query", chatAPI.Chat)
		// 流式问答接口
		chatRouter.POST("/stream", chatAPI.ChatStream)
	}
}
