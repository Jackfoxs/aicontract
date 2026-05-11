package models

import (
	"time"
)

// Article 文章模型
type Article struct {
	ID                uint64    `json:"id" xorm:"pk bigint 'id' comment('主键')"`
	Title             string    `json:"title" xorm:"varchar(255) notnull 'title' comment('文章标题')"`
	Type              string    `json:"type" xorm:"varchar(50) notnull 'type' comment('文章类型')"`
	Content           string    `json:"content" xorm:"text 'content' comment('文章正文内容')"`
	CategoryCode      string    `json:"category_code" xorm:"varchar(50) index 'category_code' comment('文档分类代码')"` // 文档分类代码，关联document_categories表
	Metadata          string    `json:"metadata" xorm:"json 'metadata' comment('文档元数据JSON')"`                     // 文档元数据（JSON格式）
	Attachment        string    `json:"attachment" xorm:"varchar(255) 'attachment' comment('附件存储路径')"`
	AttachmentContent string    `json:"attachment_content" xorm:"text 'attachment_content' comment('附件原始内容')"`
	HasAttachment     bool      `json:"has_attachment" xorm:"bool 'has_attachment' comment('是否有附件')"`
	CreatedAt         time.Time `json:"created_at" xorm:"created 'created_at' comment('创建时间')"`
	UpdatedAt         time.Time `json:"updated_at" xorm:"updated 'updated_at' comment('更新时间')"`
}
