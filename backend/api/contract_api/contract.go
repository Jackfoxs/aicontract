package contract_api

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"backend/global"
	"backend/knowledge"
	"backend/models"
	"backend/models/res"

	"github.com/cloudwego/hertz/pkg/app"
)

// UploadContractRequest 上传合同请求
type UploadContractRequest struct {
	ProcurementID uint64 `form:"procurement_id"` // 关联的采购需求ID（可选）
	CreatedBy     string `form:"created_by"`
}

// UploadContract 上传合同文件
// @Summary 上传合同文件
// @Description 上传合同文件并进行初步解析
// @Tags 合同审核
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "合同文件(PDF/DOCX)"
// @Param procurement_id formData int false "关联的采购需求ID"
// @Param created_by formData string false "创建人"
// @Success 200 {object} res.Response
// @Router /api/contract/upload [post]
func (api *ContractAPI) UploadContract(c context.Context, ctx *app.RequestContext) {
	// 1. 获取表单数据
	var req UploadContractRequest
	if err := ctx.BindAndValidate(&req); err != nil {
		res.FailWithMessage("参数错误: "+err.Error(), c, ctx)
		return
	}

	// 2. 获取上传的文件
	file, err := ctx.FormFile("file")
	if err != nil {
		res.FailWithMessage("获取文件失败: "+err.Error(), c, ctx)
		return
	}

	// 3. 验证文件类型
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".pdf" && ext != ".docx" && ext != ".doc" {
		res.FailWithMessage("不支持的文件格式，仅支持PDF、DOCX、DOC", c, ctx)
		return
	}

	// 4. 保存文件
	uploadDir := "uploads/contracts"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		res.FailWithMessage("创建上传目录失败", c, ctx)
		return
	}

	timestamp := time.Now().Format("20060102150405")
	filename := fmt.Sprintf("%s_%s", timestamp, file.Filename)
	savePath := filepath.Join(uploadDir, filename)

	if err := ctx.SaveUploadedFile(file, savePath); err != nil {
		res.FailWithMessage("保存文件失败: "+err.Error(), c, ctx)
		return
	}

	global.Log.Info("合同文件上传成功", "file", filename, "size", file.Size)

	// 5. 解析合同
	parseReq := &knowledge.ParseContractRequest{
		FilePath:      savePath,
		FileName:      file.Filename,
		ProcurementID: req.ProcurementID,
		CreatedBy:     req.CreatedBy,
	}

	review, err := api.ContractService.ParseContract(c, parseReq)
	if err != nil {
		global.Log.Error("解析合同失败", "error", err)
		res.FailWithMessage("解析合同失败: "+err.Error(), c, ctx)
		return
	}

	res.OkWithData(map[string]interface{}{
		"review_id":  review.ID,
		"file_name":  review.ContractFileName,
		"status":     review.Status,
		"parse_time": review.ParseTime,
		"message":    "合同上传成功，解析完成",
	}, c, ctx)
}

// ReviewContract 执行合同审核
// @Summary 执行合同智能审核
// @Description 对已上传的合同进行AI智能审核
// @Tags 合同审核
// @Accept json
// @Produce json
// @Param id path int true "审核记录ID"
// @Success 200 {object} res.Response
// @Router /api/contract/review/:id [post]
func (api *ContractAPI) ReviewContract(c context.Context, ctx *app.RequestContext) {
	// 1. 获取审核ID
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		res.FailWithMessage("无效的审核ID", c, ctx)
		return
	}

	// 2. 执行审核
	review, err := api.ContractService.ReviewContract(c, id)
	if err != nil {
		global.Log.Error("合同审核失败", "review_id", id, "error", err)
		res.FailWithMessage("合同审核失败: "+err.Error(), c, ctx)
		return
	}

	// 3. 返回审核结果
	res.OkWithData(map[string]interface{}{
		"review_id":     review.ID,
		"status":        review.Status,
		"risk_level":    review.RiskLevel,
		"overall_score": review.OverallScore,
		"review_time":   review.ReviewTime,
		"llm_cost":      review.LLMCost,
		"message":       "审核完成",
	}, c, ctx)
}

// GetReviewDetail 获取审核详情
// @Summary 获取审核详情
// @Description 获取合同审核的详细信息和结果
// @Tags 合同审核
// @Produce json
// @Param id path int true "审核记录ID"
// @Success 200 {object} res.Response
// @Router /api/contract/review/:id [get]
func (api *ContractAPI) GetReviewDetail(c context.Context, ctx *app.RequestContext) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		res.FailWithMessage("无效的审核ID", c, ctx)
		return
	}

	// 获取审核记录
	var review models.ContractReview
	has, err := global.DB.ID(id).Get(&review)
	if err != nil {
		global.Log.Error("查询审核记录失败", "error", err)
		res.FailWithMessage("查询审核记录失败", c, ctx)
		return
	}

	if !has {
		res.FailWithMessage("审核记录不存在", c, ctx)
		return
	}

	res.OkWithData(review, c, ctx)
}

// GetReviewReport 获取审核报告
// @Summary 获取审核报告
// @Description 获取格式化的审核报告
// @Tags 合同审核
// @Produce json
// @Param id path int true "审核记录ID"
// @Success 200 {object} res.Response
// @Router /api/contract/report/:id [get]
func (api *ContractAPI) GetReviewReport(c context.Context, ctx *app.RequestContext) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		res.FailWithMessage("无效的审核ID", c, ctx)
		return
	}

	// 生成报告
	report, err := api.ContractService.GenerateReport(id)
	if err != nil {
		global.Log.Error("生成审核报告失败", "error", err)
		res.FailWithMessage("生成审核报告失败: "+err.Error(), c, ctx)
		return
	}

	res.OkWithData(report, c, ctx)
}

// GetReviewList 获取审核列表
// @Summary 获取审核列表
// @Description 分页获取合同审核记录列表
// @Tags 合同审核
// @Produce json
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量"
// @Param risk_level query string false "风险等级"
// @Param status query string false "状态"
// @Success 200 {object} res.Response
// @Router /api/contract/list [get]
func (api *ContractAPI) GetReviewList(c context.Context, ctx *app.RequestContext) {
	// 获取分页参数
	page, _ := strconv.Atoi(ctx.Query("page"))
	pageSize, _ := strconv.Atoi(ctx.Query("pageSize"))
	riskLevel := ctx.Query("risk_level")
	status := ctx.Query("status")

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	// 构建查询
	session := global.DB.NewSession()
	defer session.Close()

	if riskLevel != "" {
		session.Where("risk_level = ?", riskLevel)
	}
	if status != "" {
		session.Where("status = ?", status)
	}

	session.OrderBy("created_at DESC")

	// 查询总数
	var total int64
	total, err := session.Count(&models.ContractReview{})
	if err != nil {
		global.Log.Error("查询审核总数失败", "error", err)
		res.FailWithMessage("查询审核总数失败", c, ctx)
		return
	}

	// 分页查询
	var reviews []models.ContractReview
	err = session.Limit(pageSize, (page-1)*pageSize).Find(&reviews)
	if err != nil {
		global.Log.Error("查询审核列表失败", "error", err)
		res.FailWithMessage("查询审核列表失败", c, ctx)
		return
	}

	res.OkWithList(reviews, total, c, ctx)
}

// DeleteReview 删除审核记录
// @Summary 删除审核记录
// @Description 删除指定的合同审核记录
// @Tags 合同审核
// @Produce json
// @Param id path int true "审核记录ID"
// @Success 200 {object} res.Response
// @Router /api/contract/review/:id [delete]
func (api *ContractAPI) DeleteReview(c context.Context, ctx *app.RequestContext) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		res.FailWithMessage("无效的审核ID", c, ctx)
		return
	}

	// 获取审核记录（用于删除文件）
	var review models.ContractReview
	has, err := global.DB.ID(id).Get(&review)
	if err != nil {
		res.FailWithMessage("查询审核记录失败", c, ctx)
		return
	}

	if !has {
		res.FailWithMessage("审核记录不存在", c, ctx)
		return
	}

	// 删除数据库记录
	_, err = global.DB.ID(id).Delete(&models.ContractReview{})
	if err != nil {
		global.Log.Error("删除审核记录失败", "error", err)
		res.FailWithMessage("删除审核记录失败", c, ctx)
		return
	}

	// 删除文件（如果存在）
	if review.ContractFilePath != "" {
		if err := os.Remove(review.ContractFilePath); err != nil {
			global.Log.Warn("删除合同文件失败", "file", review.ContractFilePath, "error", err)
		}
	}

	res.OkWithMessage("删除成功", c, ctx)
}

// CheckConsistency 一致性检查
// @Summary 一致性检查
// @Description 检查合同与采购需求的一致性
// @Tags 合同审核
// @Produce json
// @Param id path int true "审核记录ID"
// @Success 200 {object} res.Response
// @Router /api/contract/consistency/:id [get]
func (api *ContractAPI) CheckConsistency(c context.Context, ctx *app.RequestContext) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		res.FailWithMessage("无效的审核ID", c, ctx)
		return
	}

	issues, err := api.ContractService.CheckConsistency(c, id)
	if err != nil {
		global.Log.Error("一致性检查失败", "error", err)
		res.FailWithMessage("一致性检查失败: "+err.Error(), c, ctx)
		return
	}

	res.OkWithData(map[string]interface{}{
		"issues": issues,
		"count":  len(issues),
	}, c, ctx)
}

// GetRisks 获取风险项
// @Summary 获取风险项
// @Description 获取合同审核的风险项列表
// @Tags 合同审核
// @Produce json
// @Param id path int true "审核记录ID"
// @Success 200 {object} res.Response
// @Router /api/contract/risks/:id [get]
func (api *ContractAPI) GetRisks(c context.Context, ctx *app.RequestContext) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		res.FailWithMessage("无效的审核ID", c, ctx)
		return
	}

	risks, err := api.ContractService.DetectRisks(c, id)
	if err != nil {
		global.Log.Error("获取风险项失败", "error", err)
		res.FailWithMessage("获取风险项失败: "+err.Error(), c, ctx)
		return
	}

	// 统计风险级别
	riskStats := map[string]int{
		"high":   0,
		"medium": 0,
		"low":    0,
	}

	for _, risk := range risks {
		riskStats[risk.Severity]++
	}

	res.OkWithData(map[string]interface{}{
		"risks":       risks,
		"total_count": len(risks),
		"statistics":  riskStats,
	}, c, ctx)
}
