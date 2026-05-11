package routers

import (
	"context"

	"backend/compliance"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/route"
)

type RouterGroup struct {
	*route.RouterGroup
	complianceService *compliance.Service
}

// Context 为了方便统一管理，封装Hertz的Context
type Context struct {
	*app.RequestContext
}

func InitRouter(addr string, complianceService *compliance.Service) *server.Hertz {
	h := server.Default(
		server.WithHostPorts(addr),
		server.WithMaxRequestBodySize(100<<20), // 100MB，满足合规文件上传需求
	)

	apiRouterGroup := h.Group("/api")
	routerGroupApp := RouterGroup{RouterGroup: apiRouterGroup, complianceService: complianceService}

	// 注册路由
	routerGroupApp.ArticleRouter()
	routerGroupApp.ChatRouter()
	routerGroupApp.SearchRouter()
	routerGroupApp.DocumentRouter()    // 文档路由
	routerGroupApp.CategoryRouter()    // 分类路由
	routerGroupApp.ProcurementRouter() // 采购路由
	routerGroupApp.ContractRouter()    // 合同路由
	routerGroupApp.ComplianceRouter()

	return h
}

// NewContext 创建新的Context包装
func NewContext(c context.Context, ctx *app.RequestContext) *Context {
	return &Context{RequestContext: ctx}
}
