package embeddings

import (
	"context"
	"fmt"
	"backend/global"

	"github.com/cloudwego/eino-ext/components/embedding/ark"
)

// Embedder 定义向量嵌入器接口
type Embedder interface {
	// EmbedStrings 将文本转换为向量表示
	EmbedStrings(ctx context.Context, texts []string) ([][]float64, error)
}

// ArkEmbedder 是基于Ark的向量嵌入器实现
type ArkEmbedder struct {
	embedder *ark.Embedder
}

// NewArkEmbedder 创建一个新的Ark向量嵌入器
func NewArkEmbedder(ctx context.Context) (*ArkEmbedder, error) {
	// 初始化嵌入模型
	embedder, err := ark.NewEmbedder(ctx, &ark.EmbeddingConfig{
		APIKey: global.Config.Embeddings.APIKey,
		Model:  global.Config.Embeddings.Embedding,
	})

	if err != nil {
		return nil, fmt.Errorf("初始化嵌入模型失败: %w", err)
	}

	return &ArkEmbedder{
		embedder: embedder,
	}, nil
}

// EmbedStrings 实现Embedder接口，将文本转换为向量表示
func (e *ArkEmbedder) EmbedStrings(ctx context.Context, texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("输入文本为空")
	}

	// 调用底层嵌入器进行向量化
	vectors, err := e.embedder.EmbedStrings(ctx, texts)
	if err != nil {
		return nil, fmt.Errorf("向量嵌入失败: %w", err)
	}

	if vectors == nil || len(vectors) == 0 {
		return nil, fmt.Errorf("生成的向量为空")
	}

	return vectors, nil
}