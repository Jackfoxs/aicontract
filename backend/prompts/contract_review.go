package prompts

import (
	"fmt"
	"strings"

	"backend/models"
)

// ContractReviewPrompt 合同审核Prompt
type ContractReviewPrompt struct {
	SystemPrompt string
	UserPrompt   string
}

// BuildContractReviewPrompt 构建合同审核Prompt
func BuildContractReviewPrompt(
	contractContent string,
	extractedFields interface{},
	procurementData *models.ProcurementRequirement,
) *ContractReviewPrompt {
	systemPrompt := `你是一名资深的合同审核专家，精通合同法、医疗器械采购法规和合同条款审核。

你的任务是对合同进行全面审核，识别风险、检查合规性、验证一致性。

## 审核维度
1. **合规性审核**: 检查合同是否符合法律法规
2. **一致性审核**: 验证合同与采购结果的一致性
3. **完整性审核**: 检查是否缺失关键条款
4. **逻辑性审核**: 检查条款间是否存在矛盾
5. **风险识别**: 识别潜在的法律和商业风险

## 输出格式（严格JSON）
{
  "overall_assessment": {
    "risk_level": "high/medium/low",
    "score": 0.85,
    "summary": "整体评价"
  },
  "risk_items": [
    {
      "id": "risk_001",
      "type": "logical_conflict/non_compliant/missing_clause/abnormal",
      "severity": "high/medium/low",
      "location": "第3条",
      "description": "具体问题描述",
      "suggestion": "修订建议",
      "reference": "法律依据或标准"
    }
  ],
  "consistency_issues": [
    {
      "field": "total_amount",
      "contract_value": "500万元",
      "procurement_value": "480万元",
      "is_match": false,
      "issue": "金额不一致"
    }
  ],
  "compliance_issues": [
    {
      "clause": "第5条",
      "issue": "质保期不符合GB标准",
      "standard": "GB 9706.1-2020",
      "suggestion": "质保期应不少于2年"
    }
  ],
  "missing_clauses": [
    {
      "clause_type": "warranty",
      "importance": "high",
      "description": "缺少质保条款",
      "suggestion": "建议增加质保期和质保范围条款"
    }
  ],
  "suggestions": [
    {
      "priority": "high/medium/low",
      "category": "legal/commercial/technical",
      "content": "建议内容"
    }
  ]
}

// BuildContractReviewPromptWithContext 构建带知识库上下文的审核Prompt
func BuildContractReviewPromptWithContext(
    contractContent string,
    extractedFields interface{},
    procurementData *models.ProcurementRequirement,
    knowledgeContext []*models.DocumentChunk,
) *ContractReviewPrompt {
    base := BuildContractReviewPrompt(contractContent, extractedFields, procurementData)

    if len(knowledgeContext) == 0 {
        return base
    }

    var b strings.Builder
    b.WriteString(base.UserPrompt)
    b.WriteString("\n# 参考知识库\n\n")
    b.WriteString("以下是从知识库检索到的相关标准/历史合同片段，请严格参考：\n\n")
    for i, chunk := range knowledgeContext {
        if i >= 5 {
            break
        }
        b.WriteString(fmt.Sprintf("## 参考文档 %d\n", i+1))
        if chunk.Title != "" {
            b.WriteString(fmt.Sprintf("标题: %s\n", chunk.Title))
        }
        // 截断片段，避免提示超长
        content := truncateForPrompt(chunk.Content, 600)
        b.WriteString(fmt.Sprintf("内容: %s\n\n", content))
    }

    return &ContractReviewPrompt{
        SystemPrompt: base.SystemPrompt,
        UserPrompt:   b.String(),
    }
}

## 重点关注
1. 金额一致性（数字与大写、合同与采购结果）
2. 关键条款完整性（甲乙方、金额、交付、质保、违约）
3. 合规性（医疗器械相关法规）
4. 逻辑矛盾（条款间冲突）`

	var userPromptBuilder strings.Builder
	userPromptBuilder.WriteString("# 待审核合同\n\n")
	userPromptBuilder.WriteString(truncateForPrompt(contractContent, 8000))
	userPromptBuilder.WriteString("\n\n")

	// 如果有提取的字段信息
	if extractedFields != nil {
		userPromptBuilder.WriteString("# 提取的关键信息\n\n")
		userPromptBuilder.WriteString(fmt.Sprintf("%+v\n\n", extractedFields))
	}

	// 如果有采购需求数据（用于一致性对比）
	if procurementData != nil {
		userPromptBuilder.WriteString("# 采购需求信息（用于一致性对比）\n\n")
		userPromptBuilder.WriteString(fmt.Sprintf("- 需求标题: %s\n", procurementData.Title))
		userPromptBuilder.WriteString(fmt.Sprintf("- 设备类型: %s\n", procurementData.DeviceType))
		if procurementData.Budget > 0 {
			userPromptBuilder.WriteString(fmt.Sprintf("- 预算金额: %.2f 元\n", procurementData.Budget))
		}
		userPromptBuilder.WriteString("\n")
	}

	userPromptBuilder.WriteString("# 任务\n\n")
	userPromptBuilder.WriteString("请对上述合同进行全面审核，输出必须是严格的JSON格式。\n")

	return &ContractReviewPrompt{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPromptBuilder.String(),
	}
}

// BuildFieldExtractionPrompt 构建字段提取Prompt
func BuildFieldExtractionPrompt(contractContent string) *ContractReviewPrompt {
	systemPrompt := `你是一名合同信息提取专家，擅长从合同文本中准确提取关键字段。

## 提取字段
1. **甲乙方信息**: 完整名称、地址、联系方式
2. **金额信息**: 总金额（数字和大写）、币种、付款方式
3. **设备信息**: 设备名称、型号、数量、制造商
4. **日期信息**: 签订日期、交付日期、质保期
5. **其他关键条款**: 违约责任、争议解决等

## 输出格式（严格JSON）
{
  "parties_info": {
    "party_a": "甲方名称",
    "party_a_addr": "甲方地址",
    "party_b": "乙方名称",
    "party_b_addr": "乙方地址",
    "confidence": 0.9
  },
  "amount_info": {
    "total_amount": "5000000",
    "total_amount_cn": "伍佰万元整",
    "currency": "人民币",
    "consistent": true,
    "confidence": 0.95
  },
  "device_info": {
    "device_name": "高场强磁共振成像系统",
    "model": "MRI-3000T",
    "quantity": 1,
    "manufacturer": "西门子医疗",
    "confidence": 0.85
  },
  "date_info": {
    "contract_date": "2024-03-15",
    "delivery_date": "2024-06-30",
    "warranty_period": "2年",
    "confidence": 0.9
  },
  "other_fields": [
    {
      "field_name": "付款方式",
      "value": "分期付款，签订后30天内支付30%",
      "confidence": 0.8
    }
  ]
}

## 注意事项
1. 处理同义词：甲方=需方=买方，乙方=供方=卖方
2. 验证金额一致性：数字与中文大写必须匹配
3. 置信度评估：根据提取的准确性给出0-1的置信度
4. 如果某个字段未找到，返回空字符串，置信度为0`

	userPrompt := fmt.Sprintf("# 合同文本\n\n%s\n\n# 任务\n\n请提取上述合同中的所有关键字段，输出必须是严格的JSON格式。",
		truncateForPrompt(contractContent, 10000))

	return &ContractReviewPrompt{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
	}
}

// BuildConsistencyCheckPrompt 构建一致性检查Prompt
func BuildConsistencyCheckPrompt(
	contractFields interface{},
	procurementData *models.ProcurementRequirement,
) *ContractReviewPrompt {
	systemPrompt := `你是一名合同一致性审核专家，负责检查合同与采购结果的一致性。

## 检查项目
1. **金额一致性**: 合同金额是否与预算匹配（允许±10%偏差）
2. **设备信息一致性**: 设备名称、型号是否匹配
3. **供应商一致性**: 供应商名称是否一致
4. **技术参数一致性**: 关键技术参数是否符合要求

## 输出格式（严格JSON）
{
  "consistency_score": 0.85,
  "issues": [
    {
      "field": "total_amount",
      "contract_value": "500万元",
      "procurement_value": "480万元",
      "similarity": 0.96,
      "is_match": true,
      "issue": "金额在合理范围内"
    }
  ],
  "summary": "整体一致性评价"
}`

	userPrompt := fmt.Sprintf(`# 合同字段
%+v

# 采购需求信息
- 标题: %s
- 设备类型: %s
- 预算: %.2f 元

# 任务
请检查合同与采购需求的一致性，输出JSON格式。`,
		contractFields,
		procurementData.Title,
		procurementData.DeviceType,
		procurementData.Budget,
	)

	return &ContractReviewPrompt{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
	}
}

// truncateForPrompt 为Prompt截断文本
func truncateForPrompt(text string, maxLen int) string {
	if len([]rune(text)) <= maxLen {
		return text
	}
	return string([]rune(text)[:maxLen]) + "\n\n...(文本过长，已截断)"
}
