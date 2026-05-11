package models

import "time"

// ContractReview 合同审核模型
type ContractReview struct {
	ID                  uint64    `json:"id" xorm:"pk bigint 'id'"`
	Title               string    `json:"title" xorm:"varchar(255) notnull 'title' comment('合同标题')"`
	ContractFilePath    string    `json:"contract_file_path" xorm:"varchar(500) notnull 'contract_file_path' comment('合同文件路径')"`
	ContractFileName    string    `json:"contract_file_name" xorm:"varchar(255) notnull 'contract_file_name' comment('合同文件名')"`
	ProcurementID       uint64    `json:"procurement_id" xorm:"bigint 'procurement_id' comment('关联的采购需求ID')"`
	
	// 提取的内容
	ExtractedContent    string    `json:"extracted_content" xorm:"json 'extracted_content' comment('提取的结构化内容JSON')"`
	ExtractedFields     string    `json:"extracted_fields" xorm:"json 'extracted_fields' comment('提取的关键字段JSON')"`
	ExtractedTables     string    `json:"extracted_tables" xorm:"json 'extracted_tables' comment('提取的表格数据JSON')"`
	
	// 审核结果
	RiskLevel           string    `json:"risk_level" xorm:"varchar(20) index 'risk_level' comment('风险等级：high/medium/low')"`
	OverallScore        float64   `json:"overall_score" xorm:"decimal(3,2) 'overall_score' comment('综合评分 0-1')"`
	RiskItems           string    `json:"risk_items" xorm:"json 'risk_items' comment('风险项列表JSON')"`
	ConsistencyIssues   string    `json:"consistency_issues" xorm:"json 'consistency_issues' comment('一致性问题JSON')"`
	ComplianceIssues    string    `json:"compliance_issues" xorm:"json 'compliance_issues' comment('合规性问题JSON')"`
	MissingClauses      string    `json:"missing_clauses" xorm:"json 'missing_clauses' comment('缺失条款JSON')"`
	Suggestions         string    `json:"suggestions" xorm:"json 'suggestions' comment('修订建议JSON')"`
	
	// 审核状态
	Status              string    `json:"status" xorm:"varchar(50) index 'status' default 'pending' comment('状态：pending/reviewing/completed/failed')"`
	ReviewProgress      int       `json:"review_progress" xorm:"int 'review_progress' default 0 comment('审核进度 0-100')"`
	
	// 成本和性能
	ParseTime           int       `json:"parse_time" xorm:"int 'parse_time' comment('解析耗时（毫秒）')"`
	ReviewTime          int       `json:"review_time" xorm:"int 'review_time' comment('审核耗时（毫秒）')"`
	LLMCost             float64   `json:"llm_cost" xorm:"decimal(10,4) 'llm_cost' comment('LLM调用成本')"`
	LLMModel            string    `json:"llm_model" xorm:"varchar(50) 'llm_model' comment('使用的LLM模型')"`
	
	// 元数据
	CreatedBy           string    `json:"created_by" xorm:"varchar(100) 'created_by' comment('创建人')"`
	ReviewedBy          string    `json:"reviewed_by" xorm:"varchar(100) 'reviewed_by' comment('审核人')"`
	CreatedAt           time.Time `json:"created_at" xorm:"created 'created_at'"`
	UpdatedAt           time.Time `json:"updated_at" xorm:"updated 'updated_at'"`
}

// TableName 指定表名
func (ContractReview) TableName() string {
	return "contract_reviews"
}

// RiskItem 风险项结构
type RiskItem struct {
	ID          string `json:"id"`
	Type        string `json:"type"`        // 风险类型：logical_conflict/non_compliant/missing_clause/abnormal
	Severity    string `json:"severity"`    // 严重程度：high/medium/low
	Location    string `json:"location"`    // 位置：页码、条款号等
	Description string `json:"description"` // 问题描述
	Suggestion  string `json:"suggestion"`  // 修订建议
	Reference   string `json:"reference"`   // 参考依据
}

// ConsistencyIssue 一致性问题
type ConsistencyIssue struct {
	Field           string  `json:"field"`            // 字段名称：amount/model/supplier等
	ContractValue   string  `json:"contract_value"`   // 合同中的值
	ProcurementValue string `json:"procurement_value"` // 采购结果中的值
	Similarity      float64 `json:"similarity"`       // 相似度 0-1
	IsMatch         bool    `json:"is_match"`         // 是否匹配
	Issue           string  `json:"issue"`            // 问题描述
}

// ExtractedField 提取的关键字段
type ExtractedField struct {
	FieldName  string  `json:"field_name"`  // 字段名称
	Value      string  `json:"value"`       // 字段值
	Confidence float64 `json:"confidence"`  // 置信度 0-1
	Location   string  `json:"location"`    // 位置
}

// ExtractedTable 提取的表格
type ExtractedTable struct {
	ID         string     `json:"id"`
	Title      string     `json:"title"`
	PageNumber int        `json:"page_number"`
	Headers    []string   `json:"headers"`
	Rows       [][]string `json:"rows"`
	Type       string     `json:"type"` // 表格类型：tech_params/price_breakdown等
}

// ContractStructure 合同结构
type ContractStructure struct {
	Sections []ContractSection `json:"sections"` // 章节列表
	Clauses  []ContractClause  `json:"clauses"`  // 条款列表
}

// ContractSection 合同章节
type ContractSection struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Level    int    `json:"level"`    // 层级：1=一级章节，2=二级章节
	Content  string `json:"content"`  // 章节内容
	StartPos int    `json:"start_pos"` // 起始位置
	EndPos   int    `json:"end_pos"`   // 结束位置
}

// ContractClause 合同条款
type ContractClause struct {
	ID        string `json:"id"`
	Number    string `json:"number"`    // 条款编号：如"第一条"、"1.1"
	Title     string `json:"title"`     // 条款标题
	Content   string `json:"content"`   // 条款内容
	Type      string `json:"type"`      // 条款类型：parties/amount/delivery/warranty等
	SectionID string `json:"section_id"` // 所属章节ID
}

