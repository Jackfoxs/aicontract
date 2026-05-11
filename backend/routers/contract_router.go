package routers

import (
	"backend/api/contract_api"
	"backend/global"
)

// ContractRouter 合同审核路由
func (router RouterGroup) ContractRouter() {
	api := contract_api.NewContractAPI()

	contractGroup := router.Group("contract")
	{
		// 合同上传和审核
		contractGroup.POST("upload", api.UploadContract)     // 上传合同
		contractGroup.POST("review/:id", api.ReviewContract) // 执行审核

		// 审核结果查询
		contractGroup.GET("review/:id", api.GetReviewDetail) // 审核详情
		contractGroup.GET("report/:id", api.GetReviewReport) // 审核报告
		contractGroup.GET("list", api.GetReviewList)         // 审核列表
		contractGroup.DELETE("review/:id", api.DeleteReview) // 删除审核

		// 一致性和风险
		contractGroup.GET("consistency/:id", api.CheckConsistency) // 一致性检查
		contractGroup.GET("risks/:id", api.GetRisks)               // 获取风险项
	}

	global.Log.Info("合同审核路由注册成功")
}
