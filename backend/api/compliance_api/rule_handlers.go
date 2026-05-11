package compliance_api

import (
	"context"
	"strings"

	"backend/models/res"

	"github.com/cloudwego/hertz/pkg/app"
)

// ListRules 返回可选的规范规则条目
func (api *API) ListRules(c context.Context, ctx *app.RequestContext) {
	if api.Service == nil {
		res.FailWithMessage("合规服务未初始化", c, ctx)
		return
	}
	page := parseQueryInt(ctx, "page", 1)
	pageSize := parseQueryInt(ctx, "page_size", 10)
	keyword := strings.TrimSpace(string(ctx.Query("keyword")))
	includeContent := strings.ToLower(strings.TrimSpace(string(ctx.Query("include_content"))))
	withContent := includeContent == "1" || includeContent == "true"

	items, total, err := api.Service.ListRuleOptions(c, keyword, page, pageSize, withContent)
	if err != nil {
		res.FailWithMessage("获取规范条目失败: "+err.Error(), c, ctx)
		return
	}
	res.OkWithData(map[string]any{
		"total": total,
		"list":  items,
	}, c, ctx)
}
