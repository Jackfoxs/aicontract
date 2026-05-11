package routers

import (
	"backend/api"
	"backend/api/compliance_api"
	"backend/global"
)

// ComplianceRouter 合规相关路由
func (router RouterGroup) ComplianceRouter() {
	if router.complianceService == nil {
		if global.Log != nil {
			global.Log.Warn("合规服务未初始化，跳过合规路由")
		}
		return
	}
	api.ApiGroupApp.ComplianceApi = compliance_api.API{Service: router.complianceService}

	group := router.Group("compliance")
	{
		group.GET("rules", api.ApiGroupApp.ComplianceApi.ListRules)
		group.POST("upload", api.ApiGroupApp.ComplianceApi.UploadComplianceFile)
		group.POST("jobs", api.ApiGroupApp.ComplianceApi.SubmitJob)
		group.GET("jobs", api.ApiGroupApp.ComplianceApi.ListJobs)
		group.GET("jobs/:id", api.ApiGroupApp.ComplianceApi.GetJobDetail)
		group.GET("jobs/:id/issues", api.ApiGroupApp.ComplianceApi.ListJobIssues)
		group.GET("jobs/:id/highlights", api.ApiGroupApp.ComplianceApi.ListJobHighlights)
		group.POST("jobs/:id/retry", api.ApiGroupApp.ComplianceApi.RetryJob)
		group.GET("jobs/:id/report", api.ApiGroupApp.ComplianceApi.DownloadReport)
	}

	if global.Log != nil {
		global.Log.Info("合规路由注册成功")
	}
}
