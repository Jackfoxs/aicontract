package knowledge

import (
	"backend/global"
	"backend/knowledge/embeddings"
	"backend/models"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort" // 导入 sort 包以替代手动快速排序
	"strings"
	"sync" // 导入 sync 包用于并发
)

// ScoredChunk 用于存储文档块ID及其相似度分数
type ScoredChunk struct {
	ChunkID uint64  // 文档块ID
	Score   float64 // 相似度分数
}

// VectorSearchEngine 实现了SearchEngine接口的向量搜索引擎
type VectorSearchEngine struct {
	Embedder embeddings.Embedder // 向量嵌入器
}

// NewVectorSearchEngine 创建一个新的向量搜索引擎
func NewVectorSearchEngine(embedder embeddings.Embedder) *VectorSearchEngine {
	return &VectorSearchEngine{
		Embedder: embedder,
	}
}

// Search 根据查询文本搜索相似的文档块
// 参数:
//   - query: 查询文本
//   - limit: 返回结果的最大数量
//
// 返回:
//   - 相似度排序后的文档块列表
//   - 可能的错误
//
// 注意: 强烈推荐使用向量数据库方案，以下代码仅作为现有逻辑的优化示例。
func (s *VectorSearchEngine) Search(query string, limit int) ([]*models.DocumentChunk, error) {
	ctx := context.Background()

	if limit <= 0 {
		limit = 5
	}

	cleanedQuery := strings.TrimSpace(query)
	if cleanedQuery == "" {
		return []*models.DocumentChunk{}, nil
	}

	// 生成查询向量
	vectors, err := s.Embedder.EmbedStrings(ctx, []string{cleanedQuery})
	if err != nil {
		return nil, fmt.Errorf("生成查询向量失败: %w", err)
	}
	if len(vectors) == 0 {
		return nil, errors.New("生成的查询向量为空")
	}
	queryVector := vectors[0]

	// 载入全部向量记录（可结合向量库替换）
	vectorRecords := make([]models.Vector, 0)
	err = global.DB.Find(&vectorRecords)
	if err != nil {
		return nil, fmt.Errorf("获取向量数据失败: %w", err)
	}
	if len(vectorRecords) == 0 {
		return []*models.DocumentChunk{}, nil
	}

	// 5) 并发计算余弦相似度
	var wg sync.WaitGroup
	resultsChan := make(chan ScoredChunk, len(vectorRecords))

	for _, vectorRecord := range vectorRecords {
		wg.Add(1)
		go func(rec models.Vector) {
			defer wg.Done()
			var docVector []float64
			// 解析存储的 JSON 向量数据
			var unmarshalErr error
			if unmarshalErr = json.Unmarshal([]byte(rec.VectorData), &docVector); unmarshalErr != nil {
				global.Log.Warn("解析向量数据失败", "chunk_id", rec.ChunkID, "error", unmarshalErr)
				return // 跳过无法解析的记录
			}
			// 计算余弦相似度，使用embeddings包中的实现
			score := embeddings.CosineSimilarity(queryVector, docVector)
			// 将结果发送到 channel
			resultsChan <- ScoredChunk{
				ChunkID: rec.ChunkID,
				Score:   score,
			}
		}(vectorRecord)
	}

	// 等待所有 goroutine 完成
	wg.Wait()
	// 关闭 channel，以便下面的 range 循环可以结束
	close(resultsChan)

	// 从 channel 收集所有计算结果
	scoredChunks := make([]ScoredChunk, 0, len(vectorRecords))
	for res := range resultsChan {
		scoredChunks = append(scoredChunks, res)
	}

	//按相似度降序排序
	sort.Slice(scoredChunks, func(i, j int) bool {
		return scoredChunks[i].Score > scoredChunks[j].Score // 降序
	})

	//获取 Top K 的 Chunk ID ---
	// 限制结果数量
	if limit > len(scoredChunks) {
		limit = len(scoredChunks)
	}
	if limit == 0 {
		return []*models.DocumentChunk{}, nil // 如果 limit 为 0 或没有结果，返回空
	}

	// 提取 Top K 的 Chunk ID（向量排序）
	topChunkIDs := make([]uint64, 0, limit)
	for i := 0; i < limit; i++ {
		topChunkIDs = append(topChunkIDs, scoredChunks[i].ChunkID)
	}

	// --- 6. 批量获取 DocumentChunk (优化方式 A：使用 IN 子句) ---
	result := make([]*models.DocumentChunk, 0, limit)
	// 使用 IN 子句一次性查询所有需要的 DocumentChunk
	err = global.DB.In("chunk_id", topChunkIDs).Find(&result)
	if err != nil {
		global.Log.Error("批量获取文档块失败", "error", err)
		// 根据策略决定是返回错误还是部分结果或空结果
		return nil, fmt.Errorf("批量获取文档块失败: %w", err)
	}

	// --- 7. 按原始相似度顺序重新排序结果 ---
	// 因为 IN 查询不保证返回顺序，需要根据 topChunkIDs 的顺序重新排列结果
	finalResult := make([]*models.DocumentChunk, 0, limit)
	chunkMap := make(map[uint64]*models.DocumentChunk, len(result))
	for _, chunk := range result {
		// 创建一个 map 以便快速查找
		chunkMap[chunk.ChunkID] = chunk
	}

	// 按照 topChunkIDs 的顺序构建最终结果列表
	for _, id := range topChunkIDs {
		if chunk, ok := chunkMap[id]; ok {
			finalResult = append(finalResult, chunk)
		} else {
			// 处理可能出现的 chunk 未找到的情况 (理论上不应发生，除非数据不一致)
			global.Log.Warn("未能找到预期的文档块", "chunk_id", id)
		}
	}

	// 6) 直接返回向量Top-K（AI问答召回以向量为主，去掉精确匹配优先）
	return finalResult, nil
}

// SearchWithCategories 根据查询文本和分类代码搜索相似的文档块
// 参数:
//   - query: 查询文本
//   - categories: 分类代码列表（空则不过滤）
//   - limit: 返回结果的最大数量
//
// 返回:
//   - 相似度排序后的文档块列表
//   - 可能的错误
func (s *VectorSearchEngine) SearchWithCategories(query string, categories []string, limit int) ([]*models.DocumentChunk, error) {
	ctx := context.Background()

	if limit <= 0 {
		limit = 5
	}

	cleanedQuery := strings.TrimSpace(query)
	if cleanedQuery == "" {
		return []*models.DocumentChunk{}, nil
	}

	// 生成查询向量
	vectors, err := s.Embedder.EmbedStrings(ctx, []string{cleanedQuery})
	if err != nil {
		return nil, fmt.Errorf("生成查询向量失败: %w", err)
	}
	if len(vectors) == 0 {
		return nil, errors.New("生成的查询向量为空")
	}
	queryVector := vectors[0]

	// 如果指定了分类，先筛选出符合分类的文章ID
	var allowedChunkIDs map[uint64]bool
	if len(categories) > 0 {
		// 查询符合分类的文章
		var articles []models.Article
		err = global.DB.In("category_code", categories).Cols("id").Find(&articles)
		if err != nil {
			return nil, fmt.Errorf("查询分类文章失败: %w", err)
		}

		if len(articles) == 0 {
			global.Log.Info("指定分类下没有文章", "categories", categories)
			return []*models.DocumentChunk{}, nil
		}

		// 提取文章ID列表
		articleIDs := make([]uint64, 0, len(articles))
		for _, article := range articles {
			articleIDs = append(articleIDs, article.ID)
		}

		// 查询这些文章对应的文档块
		var chunks []models.DocumentChunk
		err = global.DB.In("article_id", articleIDs).Cols("chunk_id").Find(&chunks)
		if err != nil {
			return nil, fmt.Errorf("查询文档块失败: %w", err)
		}

		if len(chunks) == 0 {
			global.Log.Info("指定分类的文章没有文档块", "categories", categories)
			return []*models.DocumentChunk{}, nil
		}

		// 构建允许的chunk_id集合
		allowedChunkIDs = make(map[uint64]bool, len(chunks))
		for _, chunk := range chunks {
			allowedChunkIDs[chunk.ChunkID] = true
		}

		global.Log.Info("分类过滤",
			"categories", categories,
			"articles", len(articleIDs),
			"chunks", len(allowedChunkIDs),
		)
	}

	// 构建向量加载的 chunkID 列表
	var chunkIDList []uint64
	if len(allowedChunkIDs) > 0 {
		chunkIDList = make([]uint64, 0, len(allowedChunkIDs))
		for id := range allowedChunkIDs {
			chunkIDList = append(chunkIDList, id)
		}
	}

	// 从数据库获取向量（若指定分类则仅载入对应chunk）
	vectorRecords := make([]models.Vector, 0)
	if len(chunkIDList) > 0 {
		err = global.DB.In("chunk_id", chunkIDList).Find(&vectorRecords)
	} else {
		err = global.DB.Find(&vectorRecords)
	}

	if err != nil {
		return nil, fmt.Errorf("获取向量数据失败: %w", err)
	}

	if len(vectorRecords) == 0 {
		return []*models.DocumentChunk{}, nil
	}

	// 并发计算余弦相似度
	var wg sync.WaitGroup
	resultsChan := make(chan ScoredChunk, len(vectorRecords))

	for _, vectorRecord := range vectorRecords {
		wg.Add(1)
		go func(rec models.Vector) {
			defer wg.Done()
			var docVector []float64
			if err := json.Unmarshal([]byte(rec.VectorData), &docVector); err != nil {
				global.Log.Warn("解析向量数据失败", "chunk_id", rec.ChunkID, "error", err)
				return
			}

			score := embeddings.CosineSimilarity(queryVector, docVector)
			resultsChan <- ScoredChunk{
				ChunkID: rec.ChunkID,
				Score:   score,
			}
		}(vectorRecord)
	}

	wg.Wait()
	close(resultsChan)

	// 收集结果
	scoredChunks := make([]ScoredChunk, 0, len(vectorRecords))
	for res := range resultsChan {
		scoredChunks = append(scoredChunks, res)
	}

	// 按相似度降序排序
	sort.Slice(scoredChunks, func(i, j int) bool {
		return scoredChunks[i].Score > scoredChunks[j].Score
	})

	// 限制结果数量
	if limit > len(scoredChunks) {
		limit = len(scoredChunks)
	}
	if limit == 0 {
		return []*models.DocumentChunk{}, nil
	}

	// 提取 Top K 的 Chunk ID
	topChunkIDs := make([]uint64, 0, limit)
	for i := 0; i < limit; i++ {
		topChunkIDs = append(topChunkIDs, scoredChunks[i].ChunkID)
	}

	// 批量获取 DocumentChunk
	result := make([]*models.DocumentChunk, 0, limit)
	err = global.DB.In("chunk_id", topChunkIDs).Find(&result)
	if err != nil {
		global.Log.Error("批量获取文档块失败", "error", err)
		return nil, fmt.Errorf("批量获取文档块失败: %w", err)
	}

	// 按相似度顺序重新排序
	finalResult := make([]*models.DocumentChunk, 0, limit)
	chunkMap := make(map[uint64]*models.DocumentChunk, len(result))
	for _, chunk := range result {
		chunkMap[chunk.ChunkID] = chunk
	}

	for _, id := range topChunkIDs {
		if chunk, ok := chunkMap[id]; ok {
			finalResult = append(finalResult, chunk)
		} else {
			global.Log.Warn("未能找到预期的文档块", "chunk_id", id)
		}
	}

	// 直接返回向量Top-K（AI问答召回以向量为主）
	return finalResult, nil
}
