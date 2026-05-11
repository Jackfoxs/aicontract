package routers

import (
	"backend/api/procurement_api"
	"backend/global"
)

// ProcurementRouter 采购需求路由
func (router RouterGroup) ProcurementRouter() {
	api, err := procurement_api.NewProcurementAPI()
	if err != nil {
		global.Log.Error("初始化采购API失败", "error", err)
		panic(err)
	}

	procurementGroup := router.Group("procurement")
	{
		// 需求分析
		procurementGroup.POST("analyze", api.AnalyzeRequirement) // 分析需求生成参数
		procurementGroup.POST("verify", api.VerifyParameters)    // 校验参数

		// 需求管理
		procurementGroup.GET("list", api.GetRequirementList)  // 获取需求列表
		procurementGroup.GET(":id", api.GetRequirementDetail) // 获取需求详情
		procurementGroup.DELETE(":id", api.DeleteRequirement) // 删除需求

		// 历史案例
		procurementGroup.GET("historical-cases", api.GetHistoricalCases) // 获取历史案例
	}

	global.Log.Info("采购路由注册成功")
}
