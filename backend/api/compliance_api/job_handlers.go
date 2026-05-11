package compliance_api

import (
	"context"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"backend/compliance"
	"backend/models"
	"backend/models/res"

	"github.com/cloudwego/hertz/pkg/app"
)

// SubmitJob 提交合规任务
func (api *API) SubmitJob(c context.Context, ctx *app.RequestContext) {
	var req SubmitJobRequest
	if err := ctx.BindAndValidate(&req); err != nil {
		res.FailWithMessage("参数错误: "+err.Error(), c, ctx)
		return
	}
	tenderID, err := parseStringID("tender_file_id", req.TenderFileID)
	if err != nil {
		res.FailWithMessage(err.Error(), c, ctx)
		return
	}
	responseID, err := parseStringID("response_file_id", req.ResponseFileID)
	if err != nil {
		res.FailWithMessage(err.Error(), c, ctx)
		return
	}
	normSetID, err := parseOptionalStringID(req.NormSetID)
	if err != nil {
		res.FailWithMessage("norm_set_id 格式错误", c, ctx)
		return
	}
	selectedRuleIDs, err := parseStringIDSlice(req.SelectedRuleIDs)
	if err != nil {
		res.FailWithMessage(err.Error(), c, ctx)
		return
	}

	result, err := api.Service.SubmitJob(c, &compliance.SubmitJobInput{
		TenderFileID:    tenderID,
		ResponseFileID:  responseID,
		NormSetID:       normSetID,
		SelectedRuleIDs: selectedRuleIDs,
		AIThreshold:     req.AIThreshold,
		CreatedBy:       getUserID(ctx),
	})
	if err != nil {
		res.FailWithMessage("提交任务失败: "+err.Error(), c, ctx)
		return
	}
	res.OkWithData(SubmitJobResponse{JobID: strconv.FormatUint(result.ID, 10)}, c, ctx)
}

// GetJobDetail 查询任务详情
func (api *API) GetJobDetail(c context.Context, ctx *app.RequestContext) {
	jobID, err := parseParamUint64(ctx, "id")
	if err != nil {
		res.FailWithMessage("无效的任务ID", c, ctx)
		return
	}
	job, err := api.Service.GetJobByID(c, jobID)
	if err != nil {
		res.FailWithMessage("获取任务失败: "+err.Error(), c, ctx)
		return
	}
	issues, err := api.Service.GetIssuesByJobID(c, jobID)
	if err != nil {
		res.FailWithMessage("获取任务问题失败: "+err.Error(), c, ctx)
		return
	}
	resp := JobDetailResponse{
		JobID:        strconv.FormatUint(job.ID, 10),
		Status:       job.Status,
		Progress:     job.Progress,
		ErrorMessage: job.ErrorMessage,
		ReportJSON:   job.ReportPathJSON,
		ReportCSV:    job.ReportPathCSV,
		ReportPDF:    job.ReportPathPDF,
		AIThreshold:  job.AIThreshold,
		NormSetID:    formatOptionalID(job.NormSetID),
		CreatedBy:    formatOptionalID(job.CreatedBy),
		CreatedAt:    job.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    job.UpdatedAt.Format(time.RFC3339),
		LLMModel:     strings.TrimSpace(job.LLMModel),
		LLMCost:      job.LLMCost,
		Summary:      strings.TrimSpace(job.AnalysisSummary),
	}
	issueResp := make([]IssueResponse, 0, len(issues))
	for _, issue := range issues {
		refs := strings.TrimSpace(issue.ResponseRefs)
		issueResp = append(issueResp, IssueResponse{
			ID:               strconv.FormatUint(issue.ID, 10),
			RuleID:           formatOptionalID(issue.RuleID),
			RuleTitle:        issue.RuleTitle,
			RequiredContent:  issue.RequiredContent,
			ResponseExcerpt:  issue.ResponseExcerpt,
			Status:           issue.Status,
			MatchScore:       issue.MatchScore,
			Remark:           issue.Remark,
			HighlightRef:     issue.HighlightRef,
			LLMAdvice:        issue.LLMAdvice,
			LLMModel:         issue.LLMModel,
			SourceType:       issue.SourceType,
			RequirementID:    issue.RequirementID,
			RequirementName:  issue.RequirementName,
			RequirementLevel: issue.RequirementLevel,
			Gap:              issue.Gap,
			ResponseRefs:     refs,
		})
	}
	res.OkWithData(map[string]interface{}{
		"job":    resp,
		"issues": issueResp,
	}, c, ctx)
}

// ListJobs 任务列表
func (api *API) ListJobs(c context.Context, ctx *app.RequestContext) {
	page := parseQueryInt(ctx, "page", 1)
	pageSize := parseQueryInt(ctx, "page_size", 10)
	jobs, total, err := api.Service.ListJobs(c, page, pageSize)
	if err != nil {
		res.FailWithMessage("获取任务列表失败: "+err.Error(), c, ctx)
		return
	}
	resp := JobListResponse{Total: total, List: make([]JobListItemResponse, 0, len(jobs))}
	for _, job := range jobs {
		resp.List = append(resp.List, JobListItemResponse{
			JobID:          strconv.FormatUint(job.ID, 10),
			Status:         job.Status,
			Progress:       job.Progress,
			TenderFileID:   formatOptionalID(job.TenderFileID),
			ResponseFileID: formatOptionalID(job.ResponseFileID),
			CreatedAt:      job.CreatedAt.Format(time.RFC3339),
			LLMModel:       strings.TrimSpace(job.LLMModel),
			Summary:        strings.TrimSpace(job.AnalysisSummary),
		})
	}
	res.OkWithData(resp, c, ctx)
}

// ListJobIssues 单独获取任务问题
func (api *API) ListJobIssues(c context.Context, ctx *app.RequestContext) {
	jobID, err := parseParamUint64(ctx, "id")
	if err != nil {
		res.FailWithMessage("无效的任务ID", c, ctx)
		return
	}
	issues, err := api.Service.GetIssuesByJobID(c, jobID)
	if err != nil {
		res.FailWithMessage("获取任务问题失败: "+err.Error(), c, ctx)
		return
	}
	resp := make([]IssueResponse, 0, len(issues))
	for _, issue := range issues {
		refs := strings.TrimSpace(issue.ResponseRefs)
		resp = append(resp, IssueResponse{
			ID:               strconv.FormatUint(issue.ID, 10),
			RuleID:           formatOptionalID(issue.RuleID),
			RuleTitle:        issue.RuleTitle,
			RequiredContent:  issue.RequiredContent,
			ResponseExcerpt:  issue.ResponseExcerpt,
			Status:           issue.Status,
			MatchScore:       issue.MatchScore,
			Remark:           issue.Remark,
			HighlightRef:     issue.HighlightRef,
			LLMAdvice:        issue.LLMAdvice,
			LLMModel:         issue.LLMModel,
			SourceType:       issue.SourceType,
			RequirementID:    issue.RequirementID,
			RequirementName:  issue.RequirementName,
			RequirementLevel: issue.RequirementLevel,
			Gap:              issue.Gap,
			ResponseRefs:     refs,
		})
	}
	res.OkWithData(resp, c, ctx)
}

// ListJobHighlights 获取高亮列表
func (api *API) ListJobHighlights(c context.Context, ctx *app.RequestContext) {
	jobID, err := parseParamUint64(ctx, "id")
	if err != nil {
		res.FailWithMessage("无效的任务ID", c, ctx)
		return
	}
	fileRole := strings.TrimSpace(string(ctx.Query("file_role")))
	page := parseQueryInt(ctx, "page", 0)
	pageSize := parseQueryInt(ctx, "page_size", 0)
	highlights, err := api.Service.GetHighlightsByJobID(c, jobID, fileRole, page, pageSize)
	if err != nil {
		res.FailWithMessage("获取高亮失败: "+err.Error(), c, ctx)
		return
	}
	resp := make([]HighlightResponse, 0, len(highlights))
	for _, h := range highlights {
		resp = append(resp, HighlightResponse{
			ID:          strconv.FormatUint(h.ID, 10),
			FileRole:    h.FileRole,
			Page:        h.Page,
			OffsetStart: h.OffsetStart,
			OffsetEnd:   h.OffsetEnd,
			Text:        h.Text,
		})
	}
	res.OkWithData(resp, c, ctx)
}

// RetryJob 重新入队任务
func (api *API) RetryJob(c context.Context, ctx *app.RequestContext) {
	jobID, err := parseParamUint64(ctx, "id")
	if err != nil {
		res.FailWithMessage("无效的任务ID", c, ctx)
		return
	}
	if err := api.Service.RetryJob(c, jobID); err != nil {
		res.FailWithMessage("重试任务失败: "+err.Error(), c, ctx)
		return
	}
	res.OkWithData(RetryResponse{JobID: strconv.FormatUint(jobID, 10), Requeued: true, NextStatus: models.ComplianceJobStatusPending}, c, ctx)
}

// DownloadReport 下载报告
func (api *API) DownloadReport(c context.Context, ctx *app.RequestContext) {
	jobID, err := parseParamUint64(ctx, "id")
	if err != nil {
		res.FailWithMessage("无效的任务ID", c, ctx)
		return
	}
	format := strings.ToLower(string(ctx.Query("format")))
	if format == "" {
		format = "json"
	}
	job, err := api.Service.GetJobByID(c, jobID)
	if err != nil {
		res.FailWithMessage("获取任务失败: "+err.Error(), c, ctx)
		return
	}
	var path string
	switch format {
	case "json":
		path = job.ReportPathJSON
	case "csv":
		path = job.ReportPathCSV
	case "pdf":
		path = job.ReportPathPDF
	default:
		res.FailWithMessage("不支持的格式", c, ctx)
		return
	}
	if strings.TrimSpace(path) == "" {
		res.FailWithMessage("报告文件不存在", c, ctx)
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		res.FailWithMessage("读取报告失败: "+err.Error(), c, ctx)
		return
	}
	filename := filepath.Base(path)
	contentType := mime.TypeByExtension(filepath.Ext(path))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	ctx.Response.Header.Set("Content-Type", contentType)
	ctx.Response.Header.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	ctx.Write(data)
}
