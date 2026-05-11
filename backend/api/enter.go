package api

import (
	"backend/api/article_api"
	"backend/api/chat_api"
	"backend/api/compliance_api"
	"backend/api/search_api"
)

type ApiGroup struct {
	ArticleApi    article_api.ArticleAPI
	ChatApi       chat_api.ChatAPI
	SearchApi     search_api.SearchAPI
	ComplianceApi compliance_api.API
}

var ApiGroupApp = new(ApiGroup)
