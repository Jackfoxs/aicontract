package procurement_api

import (
	"context"

	"backend/knowledge"
	"backend/knowledge/embeddings"
)

// ProcurementAPI 采购需求API
type ProcurementAPI struct {
	ProcurementService *knowledge.ProcurementService
}

// NewProcurementAPI 创建采购需求API实例
func NewProcurementAPI() (*ProcurementAPI, error) {
	ctx := context.Background()

	// 创建embedder和search engine
	embedder, err := embeddings.NewArkEmbedder(ctx)
	if err != nil {
		return nil, err
	}

	searchEngine := knowledge.NewVectorSearchEngine(embedder)
	procurementService := knowledge.NewProcurementService(searchEngine)

	return &ProcurementAPI{
		ProcurementService: procurementService,
	}, nil
}
