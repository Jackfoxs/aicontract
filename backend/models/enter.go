package models

// GetModels 返回所有需要同步的数据库模型
func GetModels() []interface{} {
	return []interface{}{
		new(Article),
		new(DocumentChunk),
		new(Vector),
		new(DocumentCategory),
		new(DocumentParseLog),
		new(ProcurementRequirement),
		new(ContractReview),
		new(ComplianceJob),
		new(ComplianceIssue),
		new(ComplianceHighlight),
	}
}
