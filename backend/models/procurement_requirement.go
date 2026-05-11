package models

import "time"

// ProcurementRequirement 采购需求表
type ProcurementRequirement struct {
	ID                uint64    `json:"id" xorm:"pk autoincr bigint 'id'"`
	Title             string    `json:"title" xorm:"varchar(255) notnull 'title' comment('需求标题')"`
	RequirementText   string    `json:"requirement_text" xorm:"text notnull 'requirement_text' comment('需求描述原文')"`
	DeviceType        string    `json:"device_type" xorm:"varchar(100) 'device_type' comment('设备类型')"`
	Department        string    `json:"department" xorm:"varchar(100) 'department' comment('科室')"`
	Budget            float64   `json:"budget" xorm:"decimal(15,2) 'budget' comment('预算金额')"`
	GeneratedParams   string    `json:"generated_params" xorm:"json 'generated_params' comment('生成的技术参数JSON')"`
	ComplianceIssues  string    `json:"compliance_issues" xorm:"json 'compliance_issues' comment('合规性问题JSON')"`
	Suggestions       string    `json:"suggestions" xorm:"json 'suggestions' comment('优化建议JSON')"`
	HistoricalCases   string    `json:"historical_cases" xorm:"json 'historical_cases' comment('历史案例JSON')"`
	Status            string    `json:"status" xorm:"varchar(50) default 'draft' 'status' comment('状态: draft,generated,approved')"`
	AnalysisQuality   float64   `json:"analysis_quality" xorm:"decimal(3,2) 'analysis_quality' comment('分析质量评分0.0-1.0')"`
	UsedKnowledgeBase bool      `json:"used_knowledge_base" xorm:"bool default 1 'used_knowledge_base' comment('是否使用知识库')"`
	RetrievalCount    int       `json:"retrieval_count" xorm:"int 'retrieval_count' comment('检索文档数量')"`
	LLMModel          string    `json:"llm_model" xorm:"varchar(100) 'llm_model' comment('使用的LLM模型')"`
	LLMCost           float64   `json:"llm_cost" xorm:"decimal(10,4) 'llm_cost' comment('LLM成本(元)')"`
	ProcessingTime    int       `json:"processing_time" xorm:"int 'processing_time' comment('处理耗时(ms)')"`
	CreatedBy         string    `json:"created_by" xorm:"varchar(100) 'created_by' comment('创建人')"`
	CreatedAt         time.Time `json:"created_at" xorm:"created 'created_at'"`
	UpdatedAt         time.Time `json:"updated_at" xorm:"updated 'updated_at'"`
}

// TableName 指定表名
func (ProcurementRequirement) TableName() string {
	return "procurement_requirements"
}
