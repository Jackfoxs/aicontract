package knowledge

import (
	"backend/models"
	"backend/knowledge/embeddings"
)

// SearchSimilarDocuments 根据查询文本搜索相似的文档块
// 该方法现在使用VectorSearchEngine实现，遵循接口分离原则
// 参数:
//   - query: 查询文本
//   - limit: 返回结果的最大数量
// 返回:
//   - 相似度排序后的文档块列表
//   - 可能的错误
func (s *KnowledgeServiceImpl) SearchSimilarDocuments(query string, limit int) ([]*models.DocumentChunk, error) {
	// 创建搜索引擎实例
	searchEngine := NewVectorSearchEngine(s.Embedder.(embeddings.Embedder))
	// 委托搜索任务给专门的搜索引擎
	return searchEngine.Search(query, limit)
}
