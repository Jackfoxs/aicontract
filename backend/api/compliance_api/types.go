package compliance_api

// SubmitJobRequest 提交合规任务
type SubmitJobRequest struct {
	TenderFileID    string   `json:"tender_file_id" binding:"required"`
	ResponseFileID  string   `json:"response_file_id" binding:"required"`
	NormSetID       string   `json:"norm_set_id"`
	SelectedRuleIDs []string `json:"selected_rule_ids"`
	AIThreshold     float64  `json:"ai_threshold"`
}

// SubmitJobResponse 返回创建的任务ID
type SubmitJobResponse struct {
	JobID string `json:"job_id"`
}

// JobDetailResponse 任务详情响应
type JobDetailResponse struct {
	JobID        string  `json:"job_id"`
	Status       string  `json:"status"`
	Progress     int     `json:"progress"`
	ErrorMessage string  `json:"error_message"`
	ReportJSON   string  `json:"report_path_json"`
	ReportCSV    string  `json:"report_path_csv"`
	ReportPDF    string  `json:"report_path_pdf"`
	AIThreshold  float64 `json:"ai_threshold"`
	NormSetID    string  `json:"norm_set_id"`
	CreatedBy    string  `json:"created_by"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
	LLMModel     string  `json:"llm_model"`
	LLMCost      float64 `json:"llm_cost"`
	Summary      string  `json:"analysis_summary"`
}

// JobListResponse 列表响应
type JobListResponse struct {
	Total int64                 `json:"total"`
	List  []JobListItemResponse `json:"list"`
}

// JobListItemResponse 列表项
type JobListItemResponse struct {
	JobID          string `json:"job_id"`
	Status         string `json:"status"`
	Progress       int    `json:"progress"`
	TenderFileID   string `json:"tender_file_id"`
	ResponseFileID string `json:"response_file_id"`
	CreatedAt      string `json:"created_at"`
	LLMModel       string `json:"llm_model"`
	Summary        string `json:"analysis_summary"`
}

// IssueResponse 问题详情
type IssueResponse struct {
	ID               string  `json:"id"`
	RuleID           string  `json:"rule_id"`
	RuleTitle        string  `json:"rule_title"`
	RequiredContent  string  `json:"required_content"`
	ResponseExcerpt  string  `json:"response_excerpt"`
	Status           string  `json:"status"`
	MatchScore       float64 `json:"match_score"`
	Remark           string  `json:"remark"`
	HighlightRef     string  `json:"highlight_ref"`
	LLMAdvice        string  `json:"llm_advice"`
	LLMModel         string  `json:"llm_model"`
	SourceType       string  `json:"source_type"`
	RequirementID    string  `json:"requirement_id"`
	RequirementName  string  `json:"requirement_name"`
	RequirementLevel string  `json:"requirement_level"`
	Gap              string  `json:"gap"`
	ResponseRefs     string  `json:"response_refs"`
}

// RetryResponse 重试结果
type RetryResponse struct {
	JobID      string `json:"job_id"`
	Requeued   bool   `json:"requeued"`
	NextStatus string `json:"next_status"`
}

// HighlightResponse 返回高亮记录
type HighlightResponse struct {
	ID          string `json:"id"`
	FileRole    string `json:"file_role"`
	Page        int    `json:"page"`
	OffsetStart int    `json:"offset_start"`
	OffsetEnd   int    `json:"offset_end"`
	Text        string `json:"text"`
}

// UploadComplianceFileResponse 文件上传返回
type UploadComplianceFileResponse struct {
	FileID         string  `json:"file_id"`
	FileRole       string  `json:"file_role"`
	FileName       string  `json:"file_name"`
	FilePath       string  `json:"file_path"`
	FileSize       int64   `json:"file_size"`
	FileType       string  `json:"file_type"`
	ParseMethod    string  `json:"parse_method"`
	QualityScore   float64 `json:"quality_score"`
	TextLength     int     `json:"text_length"`
	PageCount      int     `json:"page_count"`
	UploadedAt     string  `json:"uploaded_at"`
	ContentPreview string  `json:"content_preview,omitempty"`
}
