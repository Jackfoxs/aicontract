package core

import (
	"backend/global"
	"backend/models"
	"fmt"
)

// InitDefaultCategories 初始化默认文档分类
func InitDefaultCategories() error {
	// 检查是否已有数据
	count, err := global.DB.Count(&models.DocumentCategory{})
	if err != nil {
		return err
	}

	// 如果已有数据，跳过初始化
	if count > 0 {
		global.Log.Info("文档分类已存在，跳过初始化", "count", count)
		return nil
	}

	// 预置分类数据
	defaultCategories := []models.DocumentCategory{
		{
			Code:        "medical_std",
			Name:        "医疗器械标准",
			Description: "GB、YY等标准文档，包括医用电气设备通用标准等",
			Icon:        "📋",
			SortOrder:   1,
		},
		{
			Code:        "legal",
			Name:        "法律法规",
			Description: "合同法、医疗器械管理条例等法律法规文档",
			Icon:        "⚖️",
			SortOrder:   2,
		},
		{
			Code:        "clinical",
			Name:        "临床需求",
			Description: "临床使用规范和需求文档",
			Icon:        "🏥",
			SortOrder:   3,
		},
		{
			Code:        "procurement_case",
			Name:        "历史采购案例",
			Description: "匿名历史采购数据和案例",
			Icon:        "📦",
			SortOrder:   4,
		},
		{
			Code:        "contract_template",
			Name:        "合同模板",
			Description: "标准合同模板和范本",
			Icon:        "📄",
			SortOrder:   5,
		},
	}

	// 批量插入
	_, err = global.DB.Insert(&defaultCategories)
	if err != nil {
		return err
	}

	global.Log.Info("成功初始化默认文档分类", "count", len(defaultCategories))
	return nil
}

// EnsureFullTextIndex 确保 document_chunk 上存在 FULLTEXT 索引（ngram）
func EnsureFullTextIndex() {
	// 检查是否已存在
	type row struct {
		Cnt int `xorm:"cnt"`
	}
	var r []row
	err := global.DB.SQL(`
SELECT COUNT(1) AS cnt
FROM information_schema.statistics
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME = 'document_chunk'
  AND INDEX_NAME = 'ft_doc_chunk';
`).Find(&r)
	if err != nil || len(r) == 0 {
		global.Log.Warn("检查FULLTEXT索引失败，跳过自动创建", "error", err)
		return
	}
	if r[0].Cnt > 0 {
		return
	}
	// 创建FULLTEXT索引（需要 ngram 插件）
	_, execErr := global.DB.Exec(`ALTER TABLE document_chunk ADD FULLTEXT INDEX ft_doc_chunk (title, content, anchors, aliases) WITH PARSER ngram;`)
	if execErr != nil {
		global.Log.Warn("创建FULLTEXT索引失败", "error", fmt.Sprintf("%v", execErr))
		return
	}
	global.Log.Info("已创建FULLTEXT索引 ft_doc_chunk")
}
