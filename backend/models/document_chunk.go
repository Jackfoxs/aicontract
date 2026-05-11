package models

import (
	"time"
)

// DocumentChunk 文档块模型
type DocumentChunk struct {
	ID           uint64 `json:"id" xorm:"pk bigint 'id' comment('主键')"`
	ChunkID      uint64 `json:"chunk_id" xorm:"varchar(64) notnull unique 'chunk_id' comment('文档块唯一标识ID')"`
	ArticleID    uint64 `json:"article_id" xorm:"bigint notnull 'article_id' comment('关联的文章ID')"`
	Title        string `json:"title" xorm:"varchar(255) 'title' comment('文档块标题')"`
	Content      string `json:"content" xorm:"text 'content' comment('文档块正文内容，用于向量化与检索')"`
	IsAttachment bool   `json:"is_attachment" xorm:"bool 'is_attachment' comment('是否附件内容')"`
	// 规则级定位与检索增强字段（规范原子化）
	Page        int       `json:"page" xorm:"int 'page' comment('所在页码')"`
	CharStart   int       `json:"char_start" xorm:"int 'char_start' comment('rune起始偏移')"`
	CharEnd     int       `json:"char_end" xorm:"int 'char_end' comment('rune结束偏移')"`
	SectionPath string    `json:"section_path" xorm:"varchar(255) 'section_path' comment('章节路径，如 第3章>3.1>3.1.2')"`
	RuleID      string    `json:"rule_id" xorm:"varchar(64) 'rule_id' comment('规则单元ID')"`
	Anchors     string    `json:"anchors" xorm:"text 'anchors' comment('锚点短语JSON')"`
	Aliases     string    `json:"aliases" xorm:"text 'aliases' comment('别名/同义短语JSON')"`
	Fingerprint string    `json:"fingerprint" xorm:"varchar(128) index 'fingerprint' comment('归一化指纹')"`
	OrderIndex  int       `json:"order_index" xorm:"int 'order_index' comment('手工排序索引，默认0')"`
	CreatedAt   time.Time `json:"created_at" xorm:"created 'created_at' comment('创建时间')"`
	UpdatedAt   time.Time `json:"updated_at" xorm:"updated 'updated_at' comment('更新时间')"`
}
