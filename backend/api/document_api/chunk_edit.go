package document_api

import (
	"backend/document/splitter"
	"backend/global"
	"backend/models"
	"backend/models/res"
	"backend/utils"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	embedding "github.com/cloudwego/eino-ext/components/embedding/ark"
	"github.com/cloudwego/hertz/pkg/app"
)

type ListChunksReq struct {
	ArticleID string `query:"article_id" json:"article_id" binding:"required"`
}

type ChunkDTO struct {
	ID           string `json:"id"`
	ChunkID      string `json:"chunk_id"`
	ArticleID    string `json:"article_id"`
	Title        string `json:"title"`
	Content      string `json:"content"`
	IsAttachment bool   `json:"is_attachment"`
	Page         int    `json:"page"`
	CharStart    int    `json:"char_start"`
	CharEnd      int    `json:"char_end"`
	SectionPath  string `json:"section_path"`
	RuleID       string `json:"rule_id"`
	Anchors      string `json:"anchors"`
	Aliases      string `json:"aliases"`
	Fingerprint  string `json:"fingerprint"`
	OrderIndex   int    `json:"order_index"`
}

// ListChunks 列出指定文章的文档块
func (api *DocumentAPI) ListChunks(c context.Context, ctx *app.RequestContext) {
	var req ListChunksReq
	if err := ctx.BindAndValidate(&req); err != nil {
		res.FailWithMessage("参数错误: "+err.Error(), c, ctx)
		return
	}
	articleID, err := strconv.ParseUint(req.ArticleID, 10, 64)
	if err != nil {
		res.FailWithMessage("article_id 非法", c, ctx)
		return
	}
	var chunks []*models.DocumentChunk
	if err := global.DB.Where("article_id = ?", articleID).Asc("order_index").Asc("id").Find(&chunks); err != nil {
		res.FailWithMessage("查询文档块失败: "+err.Error(), c, ctx)
		return
	}
	out := make([]ChunkDTO, 0, len(chunks))
	for _, ch := range chunks {
		out = append(out, ChunkDTO{
			ID:           strconv.FormatUint(ch.ID, 10),
			ChunkID:      strconv.FormatUint(ch.ChunkID, 10),
			ArticleID:    strconv.FormatUint(ch.ArticleID, 10),
			Title:        ch.Title,
			Content:      ch.Content,
			IsAttachment: ch.IsAttachment,
			Page:         ch.Page,
			CharStart:    ch.CharStart,
			CharEnd:      ch.CharEnd,
			SectionPath:  ch.SectionPath,
			RuleID:       ch.RuleID,
			Anchors:      ch.Anchors,
			Aliases:      ch.Aliases,
			Fingerprint:  ch.Fingerprint,
			OrderIndex:   ch.OrderIndex,
		})
	}
	res.OkWithData(map[string]any{"list": out}, c, ctx)
}

type UpdateChunkReq struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// UpdateChunk 更新文档块（并重建向量）
func (api *DocumentAPI) UpdateChunk(c context.Context, ctx *app.RequestContext) {
	chunkIDStr := ctx.Param("chunk_id")
	if strings.TrimSpace(chunkIDStr) == "" {
		res.FailWithMessage("chunk_id 不能为空", c, ctx)
		return
	}
	chunkID, err := strconv.ParseUint(chunkIDStr, 10, 64)
	if err != nil {
		res.FailWithMessage("chunk_id 非法", c, ctx)
		return
	}
	var req UpdateChunkReq
	if err := json.Unmarshal(ctx.Request.Body(), &req); err != nil {
		res.FailWithMessage("请求体解析失败: "+err.Error(), c, ctx)
		return
	}
	var chunk models.DocumentChunk
	has, err := global.DB.Where("chunk_id = ?", chunkID).Get(&chunk)
	if err != nil || !has {
		res.FailWithMessage("未找到文档块", c, ctx)
		return
	}
	if req.Title != "" {
		chunk.Title = req.Title
	}
	if req.Content != "" {
		chunk.Content = req.Content
		chunk.Fingerprint = splitter.Fingerprint(req.Content)
	}
	if _, err := global.DB.ID(chunk.ID).Cols("title", "content", "fingerprint").Update(&chunk); err != nil {
		res.FailWithMessage("更新失败: "+err.Error(), c, ctx)
		return
	}
	if err := reembedChunk(c, &chunk); err != nil {
		res.FailWithMessage("重建向量失败: "+err.Error(), c, ctx)
		return
	}
	res.OkWithData(map[string]any{}, c, ctx)
}

type SplitChunkReq struct {
	ChunkID string `json:"chunk_id" binding:"required"`
	Mode    string `json:"mode"` // articles_only: 仅按“第X条”
}

// SplitChunk 将单个文档块切分为多个（默认按“第X条”）
func (api *DocumentAPI) SplitChunk(c context.Context, ctx *app.RequestContext) {
	var req SplitChunkReq
	if err := json.Unmarshal(ctx.Request.Body(), &req); err != nil {
		res.FailWithMessage("请求体解析失败: "+err.Error(), c, ctx)
		return
	}
	chunkID, err := strconv.ParseUint(req.ChunkID, 10, 64)
	if err != nil {
		res.FailWithMessage("chunk_id 非法", c, ctx)
		return
	}
	var old models.DocumentChunk
	has, err := global.DB.Where("chunk_id = ?", chunkID).Get(&old)
	if err != nil || !has {
		res.FailWithMessage("未找到文档块", c, ctx)
		return
	}
	// 仅按“第X条”分割
	units := splitter.SplitArticles(old.Content)
	if len(units) <= 1 {
		res.FailWithMessage("未检测到‘第X条’，无法拆分", c, ctx)
		return
	}
	// 计算插入顺序，从 old.OrderIndex 开始
	baseOrder := old.OrderIndex
	// 删除旧向量与旧块
	if _, err := global.DB.Where("chunk_id = ?", old.ChunkID).Delete(&models.Vector{}); err != nil {
		res.FailWithMessage("删除旧向量失败: "+err.Error(), c, ctx)
		return
	}
	if _, err := global.DB.ID(old.ID).Delete(&models.DocumentChunk{}); err != nil {
		res.FailWithMessage("删除旧文档块失败: "+err.Error(), c, ctx)
		return
	}
	// 嵌入器
	emb, err := embedding.NewEmbedder(c, &embedding.EmbeddingConfig{APIKey: global.Config.Embeddings.APIKey, Model: global.Config.Embeddings.Embedding})
	if err != nil {
		res.FailWithMessage("初始化嵌入器失败: "+err.Error(), c, ctx)
		return
	}
	// 逐个插入新块
	for i, u := range units {
		newChunkID := utils.GenerateID()
		anchors := u.Anchors
		aliases := u.Aliases
		anchorsJSON, _ := json.Marshal(anchors)
		aliasesJSON, _ := json.Marshal(aliases)
		ch := &models.DocumentChunk{
			ID:           utils.GenerateID(),
			ChunkID:      newChunkID,
			ArticleID:    old.ArticleID,
			Title:        u.Title,
			Content:      u.Text,
			IsAttachment: old.IsAttachment,
			Page:         0,
			CharStart:    0,
			CharEnd:      0,
			SectionPath:  u.SectionPath,
			RuleID:       u.RuleID,
			Anchors:      string(anchorsJSON),
			Aliases:      string(aliasesJSON),
			Fingerprint:  splitter.Fingerprint(u.Text),
			OrderIndex:   baseOrder + i,
		}
		if _, err := global.DB.Insert(ch); err != nil {
			res.FailWithMessage(fmt.Sprintf("保存新块失败(%d/%d): %v", i+1, len(units), err), c, ctx)
			return
		}
		// 向量
		vectors, err := emb.EmbedStrings(c, []string{u.Text})
		if err != nil || len(vectors) == 0 {
			res.FailWithMessage(fmt.Sprintf("生成向量失败(%d/%d): %v", i+1, len(units), err), c, ctx)
			return
		}
		vecData, _ := json.Marshal(vectors[0])
		vec := &models.Vector{ID: utils.GenerateID(), ChunkID: newChunkID, VectorData: string(vecData)}
		if _, err := global.DB.Insert(vec); err != nil {
			res.FailWithMessage(fmt.Sprintf("保存向量失败(%d/%d): %v", i+1, len(units), err), c, ctx)
			return
		}
	}
	res.OkWithData(map[string]any{"count": len(units)}, c, ctx)
}

type MergeChunksReq struct {
	ChunkIDs []string `json:"chunk_ids" binding:"required"`
}

// MergeChunks 合并多个相邻块（按 OrderIndex 排序），结果覆盖第一个块，其余删除
func (api *DocumentAPI) MergeChunks(c context.Context, ctx *app.RequestContext) {
	var req MergeChunksReq
	if err := json.Unmarshal(ctx.Request.Body(), &req); err != nil {
		res.FailWithMessage("请求体解析失败: "+err.Error(), c, ctx)
		return
	}
	if len(req.ChunkIDs) < 2 {
		res.FailWithMessage("至少选择两条进行合并", c, ctx)
		return
	}
	// 读取块
	var chunks []models.DocumentChunk
	for _, idStr := range req.ChunkIDs {
		cid, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			res.FailWithMessage("chunk_id 非法: "+idStr, c, ctx)
			return
		}
		var ch models.DocumentChunk
		has, err := global.DB.Where("chunk_id = ?", cid).Get(&ch)
		if err != nil || !has {
			res.FailWithMessage("未找到文档块: "+idStr, c, ctx)
			return
		}
		chunks = append(chunks, ch)
	}
	sort.Slice(chunks, func(i, j int) bool { return chunks[i].OrderIndex < chunks[j].OrderIndex })
	base := chunks[0]
	combined := strings.Builder{}
	newTitle := strings.TrimSpace(base.Title)
	for idx, ch := range chunks {
		if idx == 0 {
			combined.WriteString(ch.Content)
			continue
		}
		combined.WriteString("\n\n")
		combined.WriteString(ch.Content)
		if newTitle == "" {
			newTitle = ch.Title
		}
	}
	// 删除其余块与向量
	for i := 1; i < len(chunks); i++ {
		_ = deleteChunkAndVector(&chunks[i])
	}
	// 更新base块
	base.Content = combined.String()
	base.Title = newTitle
	base.Fingerprint = splitter.Fingerprint(base.Content)
	if _, err := global.DB.ID(base.ID).Cols("content", "title", "fingerprint").Update(&base); err != nil {
		res.FailWithMessage("更新合并后块失败: "+err.Error(), c, ctx)
		return
	}
	if err := reembedChunk(c, &base); err != nil {
		res.FailWithMessage("重建向量失败: "+err.Error(), c, ctx)
		return
	}
	res.OkWithData(map[string]any{"chunk_id": strconv.FormatUint(base.ChunkID, 10)}, c, ctx)
}

type ReorderReq struct {
	Orders []struct {
		ChunkID    string `json:"chunk_id"`
		OrderIndex int    `json:"order_index"`
	} `json:"orders" binding:"required"`
}

// ReorderChunks 批量更新排序
func (api *DocumentAPI) ReorderChunks(c context.Context, ctx *app.RequestContext) {
	var req ReorderReq
	if err := json.Unmarshal(ctx.Request.Body(), &req); err != nil {
		res.FailWithMessage("请求体解析失败: "+err.Error(), c, ctx)
		return
	}
	for _, item := range req.Orders {
		cid, err := strconv.ParseUint(item.ChunkID, 10, 64)
		if err != nil {
			res.FailWithMessage("chunk_id 非法: "+item.ChunkID, c, ctx)
			return
		}
		_, err = global.DB.Where("chunk_id = ?", cid).Cols("order_index").Update(&models.DocumentChunk{OrderIndex: item.OrderIndex})
		if err != nil {
			res.FailWithMessage("更新排序失败: "+err.Error(), c, ctx)
			return
		}
	}
	res.OkWithData(map[string]any{}, c, ctx)
}

// DeleteChunk 删除单个块与其向量
func (api *DocumentAPI) DeleteChunk(c context.Context, ctx *app.RequestContext) {
	chunkIDStr := ctx.Param("chunk_id")
	if strings.TrimSpace(chunkIDStr) == "" {
		res.FailWithMessage("chunk_id 不能为空", c, ctx)
		return
	}
	cid, err := strconv.ParseUint(chunkIDStr, 10, 64)
	if err != nil {
		res.FailWithMessage("chunk_id 非法", c, ctx)
		return
	}
	var ch models.DocumentChunk
	has, err := global.DB.Where("chunk_id = ?", cid).Get(&ch)
	if err != nil || !has {
		res.FailWithMessage("未找到文档块", c, ctx)
		return
	}
	if err := deleteChunkAndVector(&ch); err != nil {
		res.FailWithMessage("删除失败: "+err.Error(), c, ctx)
		return
	}
	res.OkWithData(map[string]any{}, c, ctx)
}

// reembedChunk 重新生成并保存向量（覆盖原向量）
func reembedChunk(ctx context.Context, chunk *models.DocumentChunk) error {
	// 删除原向量
	if _, err := global.DB.Where("chunk_id = ?", chunk.ChunkID).Delete(&models.Vector{}); err != nil {
		return err
	}
	// 嵌入
	emb, err := embedding.NewEmbedder(ctx, &embedding.EmbeddingConfig{APIKey: global.Config.Embeddings.APIKey, Model: global.Config.Embeddings.Embedding})
	if err != nil {
		return err
	}
	vectors, err := emb.EmbedStrings(ctx, []string{chunk.Content})
	if err != nil || len(vectors) == 0 {
		return fmt.Errorf("生成向量失败: %w", err)
	}
	vecJSON, _ := json.Marshal(vectors[0])
	vec := &models.Vector{ID: utils.GenerateID(), ChunkID: chunk.ChunkID, VectorData: string(vecJSON)}
	if _, err := global.DB.Insert(vec); err != nil {
		return err
	}
	return nil
}

func deleteChunkAndVector(ch *models.DocumentChunk) error {
	if _, err := global.DB.Where("chunk_id = ?", ch.ChunkID).Delete(&models.Vector{}); err != nil {
		return err
	}
	if _, err := global.DB.ID(ch.ID).Delete(&models.DocumentChunk{}); err != nil {
		return err
	}
	return nil
}
