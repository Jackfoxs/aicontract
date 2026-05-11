package compliance

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"backend/config"
	"backend/models"

	"github.com/bytedance/sonic"
	"github.com/jung-kurt/gofpdf"
)

// ReportPaths 存储生成的报告文件路径
type ReportPaths struct {
	JSON string
	CSV  string
	PDF  string
}

// ReportBuilder 负责根据问题生成报告文件
type ReportBuilder struct {
	outputDir string
}

// NewReportBuilder 创建报告构建器
func NewReportBuilder(cfg *config.Compliance) *ReportBuilder {
	dir := "uploads/compliance"
	if cfg != nil && cfg.ReportOutputDir != "" {
		dir = cfg.ReportOutputDir
	}
	return &ReportBuilder{outputDir: dir}
}

// GenerateReports 根据问题列表生成 JSON/CSV/PDF 三种格式的报告
func (b *ReportBuilder) GenerateReports(ctx context.Context, job *models.ComplianceJob, issues []models.ComplianceIssue, finalStatus string) (ReportPaths, error) {
	_ = ctx
	if err := os.MkdirAll(b.outputDir, 0o755); err != nil {
		return ReportPaths{}, err
	}
	if job == nil {
		return ReportPaths{}, fmt.Errorf("job is nil")
	}
	timestamp := time.Now().Unix()
	baseName := fmt.Sprintf("job_%d_%d", job.ID, timestamp)

	jsonPath := filepath.Join(b.outputDir, baseName+".json")
	csvPath := filepath.Join(b.outputDir, baseName+".csv")
	pdfPath := filepath.Join(b.outputDir, baseName+".pdf")

	if err := b.writeJSON(jsonPath, job, issues, finalStatus); err != nil {
		return ReportPaths{}, err
	}
	if err := b.writeCSV(csvPath, issues); err != nil {
		return ReportPaths{}, err
	}
	if err := b.writePDF(pdfPath, job, issues, finalStatus); err != nil {
		return ReportPaths{}, err
	}

	return ReportPaths{JSON: jsonPath, CSV: csvPath, PDF: pdfPath}, nil
}

func (b *ReportBuilder) writeJSON(path string, job *models.ComplianceJob, issues []models.ComplianceIssue, finalStatus string) error {
	type jsonIssue struct {
		ID               uint64  `json:"id"`
		RuleID           uint64  `json:"rule_id"`
		RuleTitle        string  `json:"rule_title"`
		RequiredContent  string  `json:"required_content"`
		ResponseExcerpt  string  `json:"response_excerpt"`
		Status           string  `json:"status"`
		MatchScore       float64 `json:"match_score"`
		Remark           string  `json:"remark"`
		LLMAdvice        string  `json:"llm_advice"`
		LLMModel         string  `json:"llm_model"`
		HighlightRef     string  `json:"highlight_ref,omitempty"`
		SourceType       string  `json:"source_type"`
		RequirementID    string  `json:"requirement_id"`
		RequirementName  string  `json:"requirement_name"`
		RequirementLevel string  `json:"requirement_level"`
		Gap              string  `json:"gap"`
		ResponseRefs     string  `json:"response_refs"`
	}

	formatted := make([]jsonIssue, 0, len(issues))
	for _, issue := range issues {
		refs := strings.TrimSpace(issue.ResponseRefs)
		formatted = append(formatted, jsonIssue{
			ID:               issue.ID,
			RuleID:           issue.RuleID,
			RuleTitle:        issue.RuleTitle,
			RequiredContent:  issue.RequiredContent,
			ResponseExcerpt:  issue.ResponseExcerpt,
			Status:           issue.Status,
			MatchScore:       issue.MatchScore,
			Remark:           issue.Remark,
			LLMAdvice:        issue.LLMAdvice,
			LLMModel:         issue.LLMModel,
			HighlightRef:     issue.HighlightRef,
			SourceType:       issue.SourceType,
			RequirementID:    issue.RequirementID,
			RequirementName:  issue.RequirementName,
			RequirementLevel: issue.RequirementLevel,
			Gap:              issue.Gap,
			ResponseRefs:     refs,
		})
	}

	payload := map[string]interface{}{
		"job_id":           job.ID,
		"status":           finalStatus,
		"generated_at":     time.Now().Format(time.RFC3339),
		"issue_count":      len(formatted),
		"tender_file_id":   job.TenderFileID,
		"response_file_id": job.ResponseFileID,
		"issues":           formatted,
	}
	data, err := sonic.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func (b *ReportBuilder) writeCSV(path string, issues []models.ComplianceIssue) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.WriteString("\uFEFF"); err != nil { // BOM for Excel-friendly UTF-8
		return err
	}
	writer := csv.NewWriter(file)
	defer writer.Flush()

	headers := []string{"规则ID", "来源", "条目/要求", "要求描述", "响应内容", "状态", "置信度", "差距", "备注", "整改建议"}
	if err := writer.Write(headers); err != nil {
		return err
	}
	for _, issue := range issues {
		source := issue.SourceType
		if strings.TrimSpace(source) == "" {
			source = models.ComplianceIssueSourceRule
		}
		title := issue.RuleTitle
		if strings.TrimSpace(issue.RequirementName) != "" {
			title = issue.RequirementName
		}
		record := []string{
			fmt.Sprintf("%d", issue.RuleID),
			source,
			title,
			issue.RequiredContent,
			issue.ResponseExcerpt,
			issue.Status,
			fmt.Sprintf("%.2f", issue.MatchScore),
			issue.Gap,
			issue.Remark,
			issue.LLMAdvice,
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}
	return nil
}

func (b *ReportBuilder) writePDF(path string, job *models.ComplianceJob, issues []models.ComplianceIssue, finalStatus string) error {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddUTF8Font("DejaVu", "", "assets/fonts/DejaVuSansCondensed.ttf")
	pdf.AddUTF8Font("DejaVu", "B", "assets/fonts/DejaVuSansCondensed-Bold.ttf")
	pdf.SetFont("DejaVu", "B", 14)
	pdf.AddPage()

	pdf.Cell(0, 10, fmt.Sprintf("合规比对报告 - 任务 %d", job.ID))
	pdf.Ln(12)
	pdf.SetFont("DejaVu", "", 10)
	pdf.Cell(0, 8, fmt.Sprintf("状态: %s", finalStatus))
	pdf.Ln(8)
	pdf.Cell(0, 8, fmt.Sprintf("生成时间: %s", time.Now().Format(time.RFC3339)))
	pdf.Ln(12)

	for idx, issue := range issues {
		pdf.SetFont("DejaVu", "B", 11)
		pdf.MultiCell(0, 6, fmt.Sprintf("问题 %d", idx+1), "", "L", false)
		pdf.SetFont("DejaVu", "", 10)
		source := issue.SourceType
		if strings.TrimSpace(source) == "" {
			source = models.ComplianceIssueSourceRule
		}
		title := issue.RuleTitle
		if strings.TrimSpace(issue.RequirementName) != "" {
			title = issue.RequirementName
		}
		entries := []string{
			fmt.Sprintf("规则ID: %d", issue.RuleID),
			fmt.Sprintf("来源: %s", source),
			fmt.Sprintf("条目/要求: %s", title),
			fmt.Sprintf("要求描述: %s", issue.RequiredContent),
			fmt.Sprintf("响应内容: %s", issue.ResponseExcerpt),
			fmt.Sprintf("状态: %s", issue.Status),
			fmt.Sprintf("置信度: %.2f", issue.MatchScore),
			fmt.Sprintf("差距: %s", issue.Gap),
			fmt.Sprintf("备注: %s", issue.Remark),
			fmt.Sprintf("整改建议: %s", issue.LLMAdvice),
		}
		for _, line := range entries {
			pdf.MultiCell(0, 5, line, "", "L", false)
		}
		pdf.Ln(4)
	}

	return pdf.OutputFileAndClose(path)
}
