package prompts

import (
	"fmt"
	"strings"

	"backend/models"
)

// RequirementAnalysisPrompt 需求分析Prompt模板
type RequirementAnalysisPrompt struct {
	SystemPrompt string
	UserPrompt   string
}

// BuildRequirementAnalysisPrompt 构建需求分析Prompt
// 参数:
//   - requirementText: 用户输入的需求描述
//   - deviceType: 设备类型（可选）
//   - department: 科室（可选）
//   - budget: 预算（可选）
//   - knowledgeContext: 检索到的知识库上下文
//
// 返回: Prompt结构体
func BuildRequirementAnalysisPrompt(
	requirementText string,
	deviceType string,
	department string,
	budget float64,
	knowledgeContext []*models.DocumentChunk,
) *RequirementAnalysisPrompt {
	// System Prompt
	systemPrompt := `你是一名资深的医疗器械采购专家，精通医疗器械标准、技术规范和采购流程。

你的任务是根据用户的采购需求描述，结合相关的标准文档和历史采购案例，生成一份完整、专业的医疗器械技术参数表。

## 核心能力
1. 理解医疗器械采购需求（包括模糊的、不完整的需求）
2. 精通医疗器械相关标准（GB、YY等国家标准和行业标准）
3. 掌握医疗器械技术参数规范和命名
4. 了解医疗器械市场情况和价格水平
5. 熟悉医疗器械采购的合规性要求

## 工作流程
1. 分析需求：提取关键信息（设备类型、用途、科室需求等）
2. 匹配标准：在知识库中找到适用的国家标准和行业标准
3. 生成参数：根据标准和需求生成技术参数表
4. 合规检查：检查参数是否符合相关标准和法规要求
5. 优化建议：提供优化建议和注意事项

## 输出格式（严格按照JSON格式）
{
  "device_info": {
    "device_type": "设备类型",
    "device_name": "设备名称",
    "category": "设备分类（如：医用电气设备、体外诊断设备等）",
    "usage_scenario": "使用场景描述"
  },
  "technical_parameters": [
    {
      "category": "参数分类（如：基本参数、性能参数、安全参数等）",
      "parameters": [
        {
          "name": "参数名称",
          "value": "参数值或范围",
          "unit": "单位",
          "standard_reference": "依据的标准（如：GB 9706.1-2020）",
          "importance": "关键|重要|一般",
          "compliance_note": "合规性说明"
        }
      ]
    }
  ],
  "compliance_requirements": [
    {
      "requirement": "合规要求描述",
      "standard": "依据标准",
      "verification": "验证方法"
    }
  ],
  "suggestions": [
    {
      "type": "优化|注意事项|风险提示",
      "content": "建议内容",
      "reason": "理由说明"
    }
  ],
  "reference_standards": [
    {
      "code": "标准编号（如：GB 9706.1-2020）",
      "name": "标准名称",
      "relevance": "相关性说明"
    }
  ]
}

## 重要提示
1. 参数必须准确、完整、符合标准
2. 优先引用最新的国家标准和行业标准
3. 如果需求信息不足，在suggestions中提示需要补充的信息
4. 参数值应该是具体的数值或范围，避免模糊描述
5. 必须标注参数的重要程度，帮助采购人员抓重点
6. 必须进行合规性检查，避免不合规的参数`

	// User Prompt
	var userPromptBuilder strings.Builder
	userPromptBuilder.WriteString("# 采购需求\n\n")
	userPromptBuilder.WriteString(fmt.Sprintf("**需求描述**: %s\n\n", requirementText))

	if deviceType != "" {
		userPromptBuilder.WriteString(fmt.Sprintf("**设备类型**: %s\n", deviceType))
	}
	if department != "" {
		userPromptBuilder.WriteString(fmt.Sprintf("**使用科室**: %s\n", department))
	}
	if budget > 0 {
		userPromptBuilder.WriteString(fmt.Sprintf("**预算金额**: %.2f 元\n", budget))
	}

	// 添加知识库上下文
	if len(knowledgeContext) > 0 {
		userPromptBuilder.WriteString("\n# 参考知识库\n\n")
		userPromptBuilder.WriteString("以下是从知识库中检索到的相关标准和案例，请结合这些信息生成技术参数：\n\n")

		for i, chunk := range knowledgeContext {
			if i >= 5 { // 最多展示5个上下文
				break
			}
			userPromptBuilder.WriteString(fmt.Sprintf("## 参考文档 %d\n", i+1))
			if chunk.Title != "" {
				userPromptBuilder.WriteString(fmt.Sprintf("**标题**: %s\n", chunk.Title))
			}
			userPromptBuilder.WriteString(fmt.Sprintf("**内容**: %s\n\n", truncateText(chunk.Content, 500)))
		}
	}

	userPromptBuilder.WriteString("\n# 任务\n\n")
	userPromptBuilder.WriteString("请根据上述需求和参考资料，生成完整的医疗器械技术参数表。\n")
	userPromptBuilder.WriteString("输出必须是严格的JSON格式，不要包含任何其他文本。\n")

	return &RequirementAnalysisPrompt{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPromptBuilder.String(),
	}
}

// truncateText 截断文本到指定长度
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

// BuildParameterVerifyPrompt 构建参数校验Prompt
func BuildParameterVerifyPrompt(
	deviceType string,
	parameters map[string]interface{},
	knowledgeContext []*models.DocumentChunk,
) *RequirementAnalysisPrompt {
	systemPrompt := `你是一名医疗器械技术参数审核专家，负责检查技术参数表的准确性和合规性。

## 检查维度
1. **完整性检查**: 必要参数是否齐全
2. **准确性检查**: 参数值是否符合标准规范
3. **合规性检查**: 是否符合相关国家标准和行业标准
4. **一致性检查**: 参数之间是否存在矛盾
5. **合理性检查**: 参数值是否合理（不过高或过低）

## 输出格式（严格JSON）
{
  "overall_assessment": {
    "score": 0-100分,
    "level": "优秀|良好|一般|较差",
    "summary": "总体评价"
  },
  "issues": [
    {
      "type": "缺失|错误|不合规|不一致|不合理",
      "severity": "严重|中等|轻微",
      "parameter": "参数名称",
      "problem": "问题描述",
      "suggestion": "修改建议",
      "standard_reference": "依据标准"
    }
  ],
  "optimization_suggestions": [
    {
      "parameter": "参数名称",
      "current_value": "当前值",
      "suggested_value": "建议值",
      "reason": "修改理由"
    }
  ]
}`

	// 构建参数JSON字符串
	paramsJSON, _ := formatJSON(parameters)

	userPrompt := fmt.Sprintf(`# 待校验的技术参数

设备类型: %s

技术参数表:
%s

# 参考标准

%s

# 任务

请对上述技术参数进行全面审核，找出所有问题并提供优化建议。
输出必须是严格的JSON格式。`, deviceType, paramsJSON, formatKnowledgeContext(knowledgeContext))

	return &RequirementAnalysisPrompt{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
	}
}

// formatJSON 格式化JSON（简化实现）
func formatJSON(data interface{}) (string, error) {
	return fmt.Sprintf("%+v", data), nil
}

// formatKnowledgeContext 格式化知识库上下文
func formatKnowledgeContext(chunks []*models.DocumentChunk) string {
	var builder strings.Builder
	for i, chunk := range chunks {
		if i >= 3 {
			break
		}
		builder.WriteString(fmt.Sprintf("\n## 标准文档 %d\n", i+1))
		if chunk.Title != "" {
			builder.WriteString(fmt.Sprintf("标题: %s\n", chunk.Title))
		}
		builder.WriteString(fmt.Sprintf("内容: %s\n", truncateText(chunk.Content, 300)))
	}
	return builder.String()
}
