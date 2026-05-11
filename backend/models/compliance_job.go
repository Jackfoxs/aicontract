package models

import (
	"strings"
	"time"

	"github.com/bytedance/sonic"
)

const (
	// ComplianceJobStatusPending 等待处理
	ComplianceJobStatusPending = "pending"
	// ComplianceJobStatusRunning 处理中
	ComplianceJobStatusRunning = "running"
	// ComplianceJobStatusSuccess 完成
	ComplianceJobStatusSuccess = "success"
	// ComplianceJobStatusFailed 失败
	ComplianceJobStatusFailed = "failed"

	// ComplianceFileRoleTender 比选文件
	ComplianceFileRoleTender = "tender"
	// ComplianceFileRoleResponse 响应文件
	ComplianceFileRoleResponse = "response"

	ComplianceIssueSourceRule              = "rule"
	ComplianceIssueSourceTenderRequirement = "tender_requirement"
)

// ComplianceJob 合规比对任务
type ComplianceJob struct {
	ID              uint64    `json:"id" xorm:"pk autoincr bigint 'id' comment('主键')"`
	TenderFileID    uint64    `json:"tender_file_id" xorm:"bigint notnull 'tender_file_id' comment('比选文件ID')"`
	ResponseFileID  uint64    `json:"response_file_id" xorm:"bigint notnull 'response_file_id' comment('响应文件ID')"`
	NormSetID       uint64    `json:"norm_set_id" xorm:"bigint 'norm_set_id' comment('规范集合ID')"`
	SelectedRuleIDs string    `json:"selected_rule_ids" xorm:"text 'selected_rule_ids' comment('选中的规范条目ID JSON数组')"`
	Status          string    `json:"status" xorm:"varchar(20) notnull default 'pending' index 'status' comment('任务状态')"`
	Progress        int       `json:"progress" xorm:"int default 0 'progress' comment('处理进度0-100')"`
	AIThreshold     float64   `json:"ai_confidence_threshold" xorm:"decimal(5,2) default 0.75 'ai_confidence_threshold' comment('AI置信度阈值')"`
	ReportPathPDF   string    `json:"report_path_pdf" xorm:"varchar(255) 'report_path_pdf' comment('PDF报告路径')"`
	ReportPathJSON  string    `json:"report_path_json" xorm:"varchar(255) 'report_path_json' comment('JSON报告路径')"`
	ReportPathCSV   string    `json:"report_path_csv" xorm:"varchar(255) 'report_path_csv' comment('CSV报告路径')"`
	ErrorMessage    string    `json:"error_message" xorm:"text 'error_message' comment('错误信息')"`
	LLMModel        string    `json:"llm_model" xorm:"varchar(100) 'llm_model' comment('LLM模型')"`
	LLMCost         float64   `json:"llm_cost" xorm:"decimal(10,4) 'llm_cost' comment('LLM成本(元)')"`
	AnalysisSummary string    `json:"analysis_summary" xorm:"text 'analysis_summary' comment('分析摘要')"`
	CreatedBy       uint64    `json:"created_by" xorm:"bigint 'created_by' comment('任务创建人')"`
	CreatedAt       time.Time `json:"created_at" xorm:"created 'created_at' comment('创建时间')"`
	UpdatedAt       time.Time `json:"updated_at" xorm:"updated 'updated_at' comment('更新时间')"`
}

// TableName 自定义表名
func (ComplianceJob) TableName() string {
	return "compliance_job"
}

// ComplianceIssue 合规问题结果
type ComplianceIssue struct {
	ID               uint64    `json:"id" xorm:"pk autoincr bigint 'id' comment('主键')"`
	JobID            uint64    `json:"job_id" xorm:"bigint notnull index 'job_id' comment('任务ID')"`
	RuleID           uint64    `json:"rule_id" xorm:"bigint 'rule_id' comment('规范条目ID')"`
	RuleTitle        string    `json:"rule_title" xorm:"varchar(255) 'rule_title' comment('规范标题')"`
	RequiredContent  string    `json:"required_content" xorm:"text 'required_content' comment('必选内容')"`
	ResponseExcerpt  string    `json:"response_excerpt" xorm:"text 'response_excerpt' comment('响应文件片段')"`
	MatchScore       float64   `json:"match_score" xorm:"decimal(5,2) default 0 'match_score' comment('匹配置信度')"`
	IsMandatory      bool      `json:"is_mandatory" xorm:"tinyint(1) default 0 'is_mandatory' comment('是否必选条款')"`
	Status           string    `json:"status" xorm:"varchar(20) notnull default 'missing' 'status' comment('匹配状态')"`
	Remark           string    `json:"remark" xorm:"text 'remark' comment('备注/问题描述')"`
	LLMAdvice        string    `json:"llm_advice" xorm:"text 'llm_advice' comment('LLM整改建议')"`
	LLMModel         string    `json:"llm_model" xorm:"varchar(100) 'llm_model' comment('LLM模型')"`
	LLMRaw           string    `json:"llm_raw" xorm:"mediumtext 'llm_raw' comment('LLM原始响应')"`
	SourceType       string    `json:"source_type" xorm:"varchar(50) default 'rule' 'source_type' comment('问题来源类型')"`
	RequirementID    string    `json:"requirement_id" xorm:"varchar(100) 'requirement_id' comment('比选要求ID')"`
	RequirementName  string    `json:"requirement_name" xorm:"varchar(255) 'requirement_name' comment('比选要求标题')"`
	RequirementLevel string    `json:"requirement_level" xorm:"varchar(50) 'requirement_level' comment('比选要求级别')"`
	Gap              string    `json:"gap" xorm:"text 'gap' comment('差距描述')"`
	ResponseRefs     string    `json:"response_refs" xorm:"text 'response_refs' comment('响应引用JSON')"`
	HighlightRef     string    `json:"highlight_ref" xorm:"text 'highlight_ref' comment('高亮引用JSON')"`
	CreatedAt        time.Time `json:"created_at" xorm:"created 'created_at'"`
	UpdatedAt        time.Time `json:"updated_at" xorm:"updated 'updated_at'"`
}

// TableName 自定义表名
func (ComplianceIssue) TableName() string {
	return "compliance_issue"
}

// ComplianceHighlight 合规高亮片段
type ComplianceHighlight struct {
	ID          uint64    `json:"id" xorm:"pk autoincr bigint 'id' comment('主键')"`
	JobID       uint64    `json:"job_id" xorm:"bigint notnull index 'job_id' comment('任务ID')"`
	FileRole    string    `json:"file_role" xorm:"varchar(20) notnull index 'file_role' comment('文件角色')"`
	Page        int       `json:"page" xorm:"int 'page' comment('页码')"`
	OffsetStart int       `json:"offset_start" xorm:"int 'offset_start' comment('文本起始偏移')"`
	OffsetEnd   int       `json:"offset_end" xorm:"int 'offset_end' comment('文本结束偏移')"`
	BBoxes      string    `json:"bboxes" xorm:"text 'bboxes' comment('坐标信息JSON')"`
	Text        string    `json:"text" xorm:"mediumtext 'text' comment('原文片段')"`
	CreatedAt   time.Time `json:"created_at" xorm:"created 'created_at'"`
	UpdatedAt   time.Time `json:"updated_at" xorm:"updated 'updated_at'"`
}

// TableName 自定义表名
func (ComplianceHighlight) TableName() string {
	return "compliance_highlight"
}

// HighlightReference 用于描述高亮引用
type HighlightReference struct {
	FileRole    string         `json:"file"`
	Page        int            `json:"page"`
	OffsetStart int            `json:"offsetStart"`
	OffsetEnd   int            `json:"offsetEnd"`
	BBoxes      []HighlightBox `json:"bboxes,omitempty"`
}

// HighlightBox 表示单个矩形区域，坐标为归一化0-1
type HighlightBox struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// SetSelectedRuleIDs 使用 sonic 序列化规范ID列表
func (c *ComplianceJob) SetSelectedRuleIDs(ids []uint64) error {
	if len(ids) == 0 {
		c.SelectedRuleIDs = "[]"
		return nil
	}
	data, err := sonic.Marshal(ids)
	if err != nil {
		return err
	}
	c.SelectedRuleIDs = string(data)
	return nil
}

// SelectedRuleIDList 解析规范ID列表
func (c *ComplianceJob) SelectedRuleIDList() ([]uint64, error) {
	raw := strings.TrimSpace(c.SelectedRuleIDs)
	if raw == "" {
		return []uint64{}, nil
	}
	var ids []uint64
	if err := sonic.UnmarshalString(raw, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// SetHighlightReference 使用 sonic 序列化高亮引用
func (c *ComplianceIssue) SetHighlightReference(ref *HighlightReference) error {
	if ref == nil {
		c.HighlightRef = ""
		return nil
	}
	data, err := sonic.Marshal(ref)
	if err != nil {
		return err
	}
	c.HighlightRef = string(data)
	return nil
}

// HighlightReferenceObj 返回反序列化后的引用
func (c *ComplianceIssue) HighlightReferenceObj() (*HighlightReference, error) {
	raw := strings.TrimSpace(c.HighlightRef)
	if raw == "" {
		return nil, nil
	}
	ref := &HighlightReference{}
	if err := sonic.UnmarshalString(raw, ref); err != nil {
		return nil, err
	}
	return ref, nil
}

// SetBBoxes 序列化高亮坐标
func (c *ComplianceHighlight) SetBBoxes(boxes []HighlightBox) error {
	if len(boxes) == 0 {
		c.BBoxes = "[]"
		return nil
	}
	data, err := sonic.Marshal(boxes)
	if err != nil {
		return err
	}
	c.BBoxes = string(data)
	return nil
}

// BBoxList 解析高亮坐标
func (c *ComplianceHighlight) BBoxList() ([]HighlightBox, error) {
	raw := strings.TrimSpace(c.BBoxes)
	if raw == "" {
		return []HighlightBox{}, nil
	}
	var boxes []HighlightBox
	if err := sonic.UnmarshalString(raw, &boxes); err != nil {
		return nil, err
	}
	return boxes, nil
}
