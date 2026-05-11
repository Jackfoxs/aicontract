package procurement_api

import (
	"context"
	"fmt"

	"backend/global"
	"backend/knowledge"
	"backend/models"
	"backend/models/res"

	"github.com/cloudwego/hertz/pkg/app"
)

// AnalyzeRequirementRequest 需求分析请求
type AnalyzeRequirementRequest struct {
	RequirementText string  `json:"requirement_text" binding:"required"`
	DeviceType      string  `json:"device_type"`
	Department      string  `json:"department"`
	Budget          float64 `json:"budget"`
	CreatedBy       string  `json:"created_by"`
}

// AnalyzeRequirement 分析采购需求
// @Summary 分析采购需求并生成技术参数
// @Description 根据用户输入的需求描述，结合知识库生成完整的技术参数表
// @Tags 采购管理
// @Accept json
// @Produce json
// @Param body body AnalyzeRequirementRequest true "需求信息"
// @Success 200 {object} res.Response
// @Router /api/procurement/analyze [post]
func (api *ProcurementAPI) AnalyzeRequirement(c context.Context, ctx *app.RequestContext) {
	var req AnalyzeRequirementRequest
	if err := ctx.BindAndValidate(&req); err != nil {
		res.FailWithMessage("参数错误: "+err.Error(), c, ctx)
		return
	}

	// 调用服务层
	serviceReq := &knowledge.AnalyzeRequirementRequest{
		RequirementText: req.RequirementText,
		DeviceType:      req.DeviceType,
		Department:      req.Department,
		Budget:          req.Budget,
		CreatedBy:       req.CreatedBy,
	}

	response, err := api.ProcurementService.AnalyzeRequirement(c, serviceReq)
	if err != nil {
		global.Log.Error("需求分析失败", "error", err)
		res.FailWithMessage("需求分析失败: "+err.Error(), c, ctx)
		return
	}

	res.OkWithData(response, c, ctx)
}

// VerifyParametersRequest 参数校验请求
type VerifyParametersRequest struct {
	DeviceType string                 `json:"device_type" binding:"required"`
	Parameters map[string]interface{} `json:"parameters" binding:"required"`
}

// VerifyParameters 校验技术参数
// @Summary 校验已有技术参数
// @Description 对已有的技术参数进行合规性和准确性校验
// @Tags 采购管理
// @Accept json
// @Produce json
// @Param body body VerifyParametersRequest true "参数信息"
// @Success 200 {object} res.Response
// @Router /api/procurement/verify [post]
func (api *ProcurementAPI) VerifyParameters(c context.Context, ctx *app.RequestContext) {
	var req VerifyParametersRequest
	if err := ctx.BindAndValidate(&req); err != nil {
		res.FailWithMessage("参数错误: "+err.Error(), c, ctx)
		return
	}

	result, err := api.ProcurementService.VerifyParameters(c, req.DeviceType, req.Parameters)
	if err != nil {
		global.Log.Error("参数校验失败", "error", err)
		res.FailWithMessage("参数校验失败: "+err.Error(), c, ctx)
		return
	}

	res.OkWithData(result, c, ctx)
}

// GetRequirementListRequest 需求列表请求
type GetRequirementListRequest struct {
	Page       int    `form:"page" json:"page"`
	PageSize   int    `form:"pageSize" json:"pageSize"`
	DeviceType string `form:"device_type" json:"device_type"`
	Status     string `form:"status" json:"status"`
	CreatedBy  string `form:"created_by" json:"created_by"`
}

// GetRequirementList 获取需求列表
// @Summary 获取采购需求列表
// @Description 分页获取采购需求记录列表
// @Tags 采购管理
// @Accept json
// @Produce json
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量"
// @Param device_type query string false "设备类型"
// @Param status query string false "状态"
// @Success 200 {object} res.Response
// @Router /api/procurement/list [get]
func (api *ProcurementAPI) GetRequirementList(c context.Context, ctx *app.RequestContext) {
	var req GetRequirementListRequest
	if err := ctx.BindAndValidate(&req); err != nil {
		res.FailWithMessage("参数错误: "+err.Error(), c, ctx)
		return
	}

	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}

	// 构建查询
	session := global.DB.NewSession()
	defer session.Close()

	if req.DeviceType != "" {
		session.Where("device_type = ?", req.DeviceType)
	}
	if req.Status != "" {
		session.Where("status = ?", req.Status)
	}
	if req.CreatedBy != "" {
		session.Where("created_by = ?", req.CreatedBy)
	}

	session.OrderBy("created_at DESC")

	// 查询总数
	var total int64
	total, err := session.Count(&models.ProcurementRequirement{})
	if err != nil {
		global.Log.Error("查询需求总数失败", "error", err)
		res.FailWithMessage("查询需求总数失败", c, ctx)
		return
	}

	// 分页查询
	var requirements []models.ProcurementRequirement
	err = session.Limit(req.PageSize, (req.Page-1)*req.PageSize).Find(&requirements)
	if err != nil {
		global.Log.Error("查询需求列表失败", "error", err)
		res.FailWithMessage("查询需求列表失败", c, ctx)
		return
	}

	res.OkWithList(requirements, total, c, ctx)
}

// GetRequirementDetail 获取需求详情
// @Summary 获取需求详情
// @Description 根据ID获取采购需求的详细信息
// @Tags 采购管理
// @Produce json
// @Param id path string true "需求ID"
// @Success 200 {object} res.Response
// @Router /api/procurement/{id} [get]
func (api *ProcurementAPI) GetRequirementDetail(c context.Context, ctx *app.RequestContext) {
	id := ctx.Param("id")
	if id == "" {
		res.FailWithMessage("需求ID不能为空", c, ctx)
		return
	}

	var requirement models.ProcurementRequirement
	has, err := global.DB.Where("id = ?", id).Get(&requirement)
	if err != nil {
		global.Log.Error("查询需求失败", "error", err)
		res.FailWithMessage("查询需求失败", c, ctx)
		return
	}

	if !has {
		res.FailWithMessage("需求不存在", c, ctx)
		return
	}

	res.OkWithData(requirement, c, ctx)
}

// DeleteRequirement 删除需求
// @Summary 删除需求
// @Description 删除指定的采购需求记录
// @Tags 采购管理
// @Produce json
// @Param id path string true "需求ID"
// @Success 200 {object} res.Response
// @Router /api/procurement/{id} [delete]
func (api *ProcurementAPI) DeleteRequirement(c context.Context, ctx *app.RequestContext) {
	id := ctx.Param("id")
	if id == "" {
		res.FailWithMessage("需求ID不能为空", c, ctx)
		return
	}

	_, err := global.DB.Where("id = ?", id).Delete(&models.ProcurementRequirement{})
	if err != nil {
		global.Log.Error("删除需求失败", "error", err)
		res.FailWithMessage("删除需求失败", c, ctx)
		return
	}

	res.OkWithMessage("删除成功", c, ctx)
}

// GetHistoricalCases 获取历史案例
// @Summary 获取历史采购案例
// @Description 根据设备类型检索相关的历史采购案例
// @Tags 采购管理
// @Produce json
// @Param device_type query string true "设备类型"
// @Param limit query int false "返回数量"
// @Success 200 {object} res.Response
// @Router /api/procurement/historical-cases [get]
func (api *ProcurementAPI) GetHistoricalCases(c context.Context, ctx *app.RequestContext) {
	deviceType := ctx.Query("device_type")
	if deviceType == "" {
		res.FailWithMessage("设备类型不能为空", c, ctx)
		return
	}

	limit := 10
	if limitStr := ctx.Query("limit"); limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}

	cases, err := api.ProcurementService.GetHistoricalCases(c, deviceType, limit)
	if err != nil {
		global.Log.Error("获取历史案例失败", "error", err)
		res.FailWithMessage("获取历史案例失败: "+err.Error(), c, ctx)
		return
	}

	res.OkWithData(cases, c, ctx)
}
