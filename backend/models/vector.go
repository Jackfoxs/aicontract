package models

import (
	"time"
)

// Vector 向量模型
type Vector struct {
	ID         uint64    `json:"id" xorm:"pk bigint 'id' comment('主键')"`
	ChunkID    uint64    `json:"chunk_id" xorm:"varchar(64) notnull unique 'chunk_id' comment('关联的文档块ID')"`
	VectorData string    `json:"vector_data" xorm:"text 'vector_data' comment('向量数据JSON')"`
	CreatedAt  time.Time `json:"created_at" xorm:"created 'created_at' comment('创建时间')"`
	UpdatedAt  time.Time `json:"updated_at" xorm:"updated 'updated_at' comment('更新时间')"`
}
