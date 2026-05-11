package models

import "time"

// DocumentCategory 文档分类表
type DocumentCategory struct {
	ID          uint64    `json:"id" xorm:"pk autoincr bigint 'id'"`
	Code        string    `json:"code" xorm:"varchar(50) unique notnull 'code' comment('分类代码')"`
	Name        string    `json:"name" xorm:"varchar(100) notnull 'name' comment('分类名称')"`
	Description string    `json:"description" xorm:"text 'description' comment('分类描述')"`
	Icon        string    `json:"icon" xorm:"varchar(50) 'icon' comment('图标')"`
	SortOrder   int       `json:"sort_order" xorm:"int default 0 'sort_order' comment('排序')"`
	CreatedAt   time.Time `json:"created_at" xorm:"created 'created_at'"`
	UpdatedAt   time.Time `json:"updated_at" xorm:"updated 'updated_at'"`
}

// TableName 指定表名
func (DocumentCategory) TableName() string {
	return "document_categories"
}
