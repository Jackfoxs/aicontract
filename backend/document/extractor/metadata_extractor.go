package extractor

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"backend/global"
	"backend/llm"
	"backend/models"

	"github.com/bytedance/sonic"
)

// MetadataExtractor 合同元数据提取器
type MetadataExtractor struct {
	llmClient *llm.DeepSeekClient // DeepSeek Chat客户端
}

// NewMetadataExtractor 创建元数据提取器
func NewMetadataExtractor() *MetadataExtractor {
	return &MetadataExtractor{
		llmClient: llm.NewDeepSeekClient(),
	}
}

// ExtractedFields 提取的字段集合
type ExtractedFields struct {
	PartiesInfo *PartiesInfo            `json:"parties_info"`
	AmountInfo  *AmountInfo             `json:"amount_info"`
	DeviceInfo  *DeviceInfo             `json:"device_info"`
	DateInfo    *DateInfo               `json:"date_info"`
	OtherFields []models.ExtractedField `json:"other_fields"`
}

// PartiesInfo 甲乙方信息
type PartiesInfo struct {
	PartyA     string  `json:"party_a"`      // 甲方名称
	PartyAAddr string  `json:"party_a_addr"` // 甲方地址
	PartyB     string  `json:"party_b"`      // 乙方名称
	PartyBAddr string  `json:"party_b_addr"` // 乙方地址
	Confidence float64 `json:"confidence"`   // 置信度
}

// AmountInfo 金额信息
type AmountInfo struct {
	TotalAmount   string  `json:"total_amount"`    // 总金额（数字）
	TotalAmountCN string  `json:"total_amount_cn"` // 总金额（中文大写）
	Currency      string  `json:"currency"`        // 币种
	Consistent    bool    `json:"consistent"`      // 数字和大写是否一致
	Confidence    float64 `json:"confidence"`      // 置信度
}

// DeviceInfo 设备信息
type DeviceInfo struct {
	DeviceName   string  `json:"device_name"`  // 设备名称
	Model        string  `json:"model"`        // 型号
	Quantity     int     `json:"quantity"`     // 数量
	Manufacturer string  `json:"manufacturer"` // 制造商
	Confidence   float64 `json:"confidence"`   // 置信度
}

// DateInfo 日期信息
type DateInfo struct {
	ContractDate   string  `json:"contract_date"`   // 合同签订日期
	DeliveryDate   string  `json:"delivery_date"`   // 交付日期
	WarrantyPeriod string  `json:"warranty_period"` // 质保期
	Confidence     float64 `json:"confidence"`      // 置信度
}

// ExtractAll 提取所有关键字段
func (e *MetadataExtractor) ExtractAll(ctx context.Context, content string) (*ExtractedFields, error) {
	fields := &ExtractedFields{
		OtherFields: []models.ExtractedField{},
	}

	// 1. 提取甲乙方信息（规则+LLM）
	parties, err := e.ExtractParties(ctx, content)
	if err != nil {
		global.Log.Warn("提取甲乙方信息失败", "error", err)
	} else {
		fields.PartiesInfo = parties
	}

	// 2. 提取金额信息（规则+验证）
	amount, err := e.ExtractAmount(content)
	if err != nil {
		global.Log.Warn("提取金额信息失败", "error", err)
	} else {
		fields.AmountInfo = amount
	}

	// 3. 提取设备信息（规则+LLM）
	device, err := e.ExtractDevice(ctx, content)
	if err != nil {
		global.Log.Warn("提取设备信息失败", "error", err)
	} else {
		fields.DeviceInfo = device
	}

	// 4. 提取日期信息（规则）
	dates, err := e.ExtractDates(content)
	if err != nil {
		global.Log.Warn("提取日期信息失败", "error", err)
	} else {
		fields.DateInfo = dates
	}

	return fields, nil
}

// ExtractParties 提取甲乙方信息
func (e *MetadataExtractor) ExtractParties(ctx context.Context, content string) (*PartiesInfo, error) {
	// 先尝试规则提取
	parties := e.extractPartiesByRules(content)

	// 如果规则提取不完整，使用LLM辅助
	if parties.PartyA == "" || parties.PartyB == "" {
		llmParties, err := e.extractPartiesByLLM(ctx, content)
		if err == nil && llmParties != nil {
			if parties.PartyA == "" {
				parties.PartyA = llmParties.PartyA
			}
			if parties.PartyB == "" {
				parties.PartyB = llmParties.PartyB
			}
			parties.Confidence = llmParties.Confidence
		}
	}

	return parties, nil
}

// extractPartiesByRules 规则提取甲乙方
func (e *MetadataExtractor) extractPartiesByRules(content string) *PartiesInfo {
	parties := &PartiesInfo{
		Confidence: 0.7, // 规则提取置信度较低
	}

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 匹配"甲方："后的内容
		if strings.Contains(line, "甲方") && (strings.Contains(line, "：") || strings.Contains(line, ":")) {
			re := regexp.MustCompile(`甲方[：:]\s*([^\n\r，,。.]+)`)
			if matches := re.FindStringSubmatch(line); len(matches) > 1 {
				parties.PartyA = strings.TrimSpace(matches[1])
			}
		}

		// 匹配"乙方："后的内容
		if strings.Contains(line, "乙方") && (strings.Contains(line, "：") || strings.Contains(line, ":")) {
			re := regexp.MustCompile(`乙方[：:]\s*([^\n\r，,。.]+)`)
			if matches := re.FindStringSubmatch(line); len(matches) > 1 {
				parties.PartyB = strings.TrimSpace(matches[1])
			}
		}

		// 匹配地址
		if strings.Contains(line, "地址") && strings.Contains(parties.PartyAAddr, "") {
			re := regexp.MustCompile(`地址[：:]\s*([^\n\r]+)`)
			if matches := re.FindStringSubmatch(line); len(matches) > 1 {
				if parties.PartyA != "" && parties.PartyAAddr == "" {
					parties.PartyAAddr = strings.TrimSpace(matches[1])
				} else if parties.PartyB != "" && parties.PartyBAddr == "" {
					parties.PartyBAddr = strings.TrimSpace(matches[1])
				}
			}
		}
	}

	return parties
}

// extractPartiesByLLM LLM提取甲乙方
func (e *MetadataExtractor) extractPartiesByLLM(ctx context.Context, content string) (*PartiesInfo, error) {
	systemPrompt := `你是一名合同信息提取专家。请从合同文本中准确提取甲乙方信息。

输出格式（严格JSON）：
{
  "party_a": "甲方完整名称",
  "party_a_addr": "甲方地址",
  "party_b": "乙方完整名称",
  "party_b_addr": "乙方地址",
  "confidence": 0.9
}`

	userPrompt := fmt.Sprintf("请提取以下合同中的甲乙方信息：\n\n%s", truncateText(content, 2000))

	response, _, err := e.llmClient.Chat(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, err
	}

	var parties PartiesInfo
	if err := sonic.UnmarshalString(response, &parties); err != nil {
		return nil, fmt.Errorf("解析LLM响应失败: %w", err)
	}

	return &parties, nil
}

// ExtractAmount 提取金额信息
func (e *MetadataExtractor) ExtractAmount(content string) (*AmountInfo, error) {
	amount := &AmountInfo{
		Currency:   "人民币",
		Confidence: 0.7,
	}

	// 匹配金额（数字）
	re := regexp.MustCompile(`(?:总金额|合同金额|价格)[：:]\s*([0-9,，]+(?:\.[0-9]+)?)\s*元`)
	if matches := re.FindStringSubmatch(content); len(matches) > 1 {
		amount.TotalAmount = strings.ReplaceAll(strings.ReplaceAll(matches[1], ",", ""), "，", "")
	}

	// 匹配金额（中文大写）
	reCN := regexp.MustCompile(`[（(]大写[）)][：:]\s*([壹贰叁肆伍陆柒捌玖拾佰仟万亿]+元)`)
	if matches := reCN.FindStringSubmatch(content); len(matches) > 1 {
		amount.TotalAmountCN = matches[1]
	}

	// 验证一致性（简化实现）
	if amount.TotalAmount != "" && amount.TotalAmountCN != "" {
		// TODO: 实现数字和中文大写的转换验证
		amount.Consistent = true
		amount.Confidence = 0.9
	}

	return amount, nil
}

// ExtractDevice 提取设备信息
func (e *MetadataExtractor) ExtractDevice(ctx context.Context, content string) (*DeviceInfo, error) {
	device := &DeviceInfo{
		Confidence: 0.6,
	}

	// 规则提取设备名称
	if re := regexp.MustCompile(`(?:设备名称|产品名称)[：:]\s*([^\n\r]+)`); re.MatchString(content) {
		matches := re.FindStringSubmatch(content)
		if len(matches) > 1 {
			device.DeviceName = strings.TrimSpace(matches[1])
		}
	}

	// 规则提取型号
	if re := regexp.MustCompile(`(?:型号|规格)[：:]\s*([^\n\r]+)`); re.MatchString(content) {
		matches := re.FindStringSubmatch(content)
		if len(matches) > 1 {
			device.Model = strings.TrimSpace(matches[1])
		}
	}

	// 规则提取数量
	if re := regexp.MustCompile(`数量[：:]\s*([0-9]+)`); re.MatchString(content) {
		matches := re.FindStringSubmatch(content)
		if len(matches) > 1 {
			fmt.Sscanf(matches[1], "%d", &device.Quantity)
		}
	}

	return device, nil
}

// ExtractDates 提取日期信息
func (e *MetadataExtractor) ExtractDates(content string) (*DateInfo, error) {
	dates := &DateInfo{
		Confidence: 0.7,
	}

	// 匹配合同日期
	datePattern := `([0-9]{4})\s*年\s*([0-9]{1,2})\s*月\s*([0-9]{1,2})\s*日`
	re := regexp.MustCompile(datePattern)
	if matches := re.FindAllStringSubmatch(content, -1); len(matches) > 0 {
		// 取第一个日期作为合同日期
		dates.ContractDate = fmt.Sprintf("%s-%s-%s", matches[0][1], matches[0][2], matches[0][3])
	}

	// 匹配交付日期
	if strings.Contains(content, "交付") || strings.Contains(content, "交货") {
		reDelivery := regexp.MustCompile(`(?:交付|交货)[^0-9]*` + datePattern)
		if matches := reDelivery.FindStringSubmatch(content); len(matches) > 3 {
			dates.DeliveryDate = fmt.Sprintf("%s-%s-%s", matches[1], matches[2], matches[3])
		}
	}

	// 匹配质保期
	if re := regexp.MustCompile(`质保期[：:]\s*([0-9]+)\s*([年月])`); re.MatchString(content) {
		matches := re.FindStringSubmatch(content)
		if len(matches) > 2 {
			dates.WarrantyPeriod = matches[1] + matches[2]
		}
	}

	return dates, nil
}

// truncateText 截断文本
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "...(已截断)"
}
