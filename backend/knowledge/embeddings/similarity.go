package embeddings

import (
	"math"
)

// ScoredItem 用于存储ID及其相似度分数
type ScoredItem struct {
	ID    uint64  // 项目ID
	Score float64 // 相似度分数
}

// CosineSimilarity 计算两个向量之间的余弦相似度
// 返回值范围为 [-1, 1]，值越大表示越相似
func CosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0.0 // 向量长度不匹配或为空
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	// 避免除以零
	denominator := math.Sqrt(normA) * math.Sqrt(normB)
	if denominator == 0.0 {
		return 0.0
	}

	similarity := dotProduct / denominator
	// 限制相似度值在 [-1, 1] 范围内，避免浮点数精度问题
	if similarity > 1.0 {
		return 1.0
	}
	if similarity < -1.0 {
		return -1.0
	}
	return similarity
}

// EuclideanDistance 计算两个向量之间的欧几里得距离
// 值越小表示越相似
func EuclideanDistance(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return math.MaxFloat64 // 向量长度不匹配或为空
	}

	var sumSquared float64
	for i := range a {
		diff := a[i] - b[i]
		sumSquared += diff * diff
	}

	return math.Sqrt(sumSquared)
}

// DotProduct 计算两个向量的点积
// 在归一化向量上，点积等同于余弦相似度
func DotProduct(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0.0 // 向量长度不匹配或为空
	}

	var dotProduct float64
	for i := range a {
		dotProduct += a[i] * b[i]
	}

	return dotProduct
}