package knowledge

import (
	"backend/document/parser"
	"backend/document/splitter"
	"backend/global"
	"backend/models"
	"backend/utils"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/semantic"

	"github.com/bytedance/sonic"
	embedding "github.com/cloudwego/eino-ext/components/embedding/ark"
	"github.com/cloudwego/eino/schema"
)

// ProcessDocument 处理文档分割和向量化
// 将文档内容分割成多个块，并为每个块生成向量嵌入，存储到数据库中
func (s *KnowledgeServiceImpl) ProcessDocument(article *models.Article) error {
	if article == nil {
		return errors.New("无效的文章对象：article 不能为 nil")
	}

	// 检查是否为PDF附件，如果是则直接处理PDF文件
	if article.HasAttachment && strings.HasSuffix(strings.ToLower(article.Attachment), ".pdf") {
		global.Log.Info("检测到PDF附件，开始处理PDF文件", "attachment", article.Attachment)
		return s.processPDFDocument(article)
	}

	// 检查是否为DOCX附件，如果是则处理DOCX文件
	if article.HasAttachment && (strings.HasSuffix(strings.ToLower(article.Attachment), ".docx") || strings.HasSuffix(strings.ToLower(article.Attachment), ".doc")) {
		global.Log.Info("检测到DOCX附件，开始处理DOCX文件", "attachment", article.Attachment)
		return s.processDocxDocument(article)
	}

	if strings.TrimSpace(article.Content) == "" {
		global.Log.Warn("文章内容为空，跳过处理", "article_id", article.ID)
		return nil
	}

	ctx := context.Background()
	global.Log.Info("开始处理文章", "article_id", article.ID, "title", article.Title)

	// 创建文档对象
	doc := &schema.Document{
		ID:      fmt.Sprintf("article_%d", article.ID),
		Content: article.Content,
		MetaData: map[string]interface{}{
			"title": article.Title,
			"type":  article.Type,
		},
	}

	embedders, err := embedding.NewEmbedder(ctx, &embedding.EmbeddingConfig{
		APIKey: global.Config.Embeddings.APIKey,    // 使用配置中的API密钥
		Model:  global.Config.Embeddings.Embedding, // 使用配置指定的嵌入模型
	})
	if err != nil {
		global.Log.Error("初始化嵌入器失败", "error", err, "article_id", article.ID)
		return fmt.Errorf("初始化嵌入器失败: %w", err)
	}

	// 规范类文档处理
	if isNormativeArticle(article) {
		// 法律法规：仅按“第X条”切分；若没有“第X条”，则回退语义切分
		if isLegalArticle(article) {
			global.Log.Info("法律法规：按‘第X条’切分", "article_id", article.ID)
			units := splitter.SplitArticles(article.Content)
			if len(units) == 0 {
				global.Log.Info("未检测到‘第X条’，切换语义切分", "article_id", article.ID)
				splitterSemantic, err := semantic.NewSplitter(ctx, &semantic.Config{
					Embedding:    embedders,
					BufferSize:   2,
					MinChunkSize: 100,
					Separators:   []string{"\n", ".", "?", "!", "。", "<p>"},
					Percentile:   0.8,
				})
				if err != nil {
					return fmt.Errorf("初始化文本分割器失败: %w", err)
				}
				chunks, err := splitterSemantic.Transform(ctx, []*schema.Document{doc})
				if err != nil || len(chunks) == 0 {
					chunks = []*schema.Document{doc}
				}
				for i, chunk := range chunks {
					chunkID := utils.GenerateID()
					if strings.TrimSpace(chunk.Content) == "" {
						continue
					}
					documentChunk := &models.DocumentChunk{
						ID:        utils.GenerateID(),
						ChunkID:   chunkID,
						ArticleID: article.ID,
						Title:     article.Title,
						Content:   chunk.Content,
					}
					if _, err := global.DB.Insert(documentChunk); err != nil {
						return fmt.Errorf("保存文档块失败 (块 %d/%d): %w", i+1, len(chunks), err)
					}
					vectors, err := s.Embedder.EmbedStrings(ctx, []string{chunk.Content})
					if err != nil || len(vectors) == 0 {
						return fmt.Errorf("生成向量嵌入失败 (块 %d/%d): %w", i+1, len(chunks), err)
					}
					vectorData, err := json.Marshal(vectors[0])
					if err != nil {
						return fmt.Errorf("序列化向量数据失败 (块 %d/%d): %w", i+1, len(chunks), err)
					}
					vector := &models.Vector{ID: utils.GenerateID(), ChunkID: chunkID, VectorData: string(vectorData)}
					if _, err = global.DB.Insert(vector); err != nil {
						return fmt.Errorf("保存向量失败 (块 %d/%d): %w", i+1, len(chunks), err)
					}
				}
				global.Log.Info("语义分割完成(法律法规无‘第X条’)", "article_id", article.ID)
				return nil
			}

			for i, u := range units {
				chunkID := utils.GenerateID()
				anchorsJSON, _ := sonic.Marshal(u.Anchors)
				aliasesJSON, _ := sonic.Marshal(u.Aliases)
				documentChunk := &models.DocumentChunk{
					ID:          utils.GenerateID(),
					ChunkID:     chunkID,
					ArticleID:   article.ID,
					Title:       article.Title,
					Content:     u.Text,
					Page:        u.Page,
					CharStart:   u.CharStart,
					CharEnd:     u.CharEnd,
					SectionPath: u.SectionPath,
					RuleID:      u.RuleID,
					Anchors:     string(anchorsJSON),
					Aliases:     string(aliasesJSON),
					Fingerprint: splitter.Fingerprint(u.Text),
				}
				if _, err := global.DB.Insert(documentChunk); err != nil {
					return fmt.Errorf("保存法律法规文档块失败 (块 %d/%d): %w", i+1, len(units), err)
				}
				vectors, err := s.Embedder.EmbedStrings(ctx, []string{u.Text})
				if err != nil || len(vectors) == 0 {
					return fmt.Errorf("生成向量失败 (法律法规块 %d/%d): %w", i+1, len(units), err)
				}
				vectorData, err := json.Marshal(vectors[0])
				if err != nil {
					return fmt.Errorf("序列化向量失败 (法律法规块 %d/%d): %w", i+1, len(units), err)
				}
				vector := &models.Vector{ID: utils.GenerateID(), ChunkID: chunkID, VectorData: string(vectorData)}
				if _, err = global.DB.Insert(vector); err != nil {
					return fmt.Errorf("保存向量失败 (法律法规块 %d/%d): %w", i+1, len(units), err)
				}
			}
			global.Log.Info("法律法规‘第X条’切分完成", "article_id", article.ID)
			return nil
		}

		// 其他规范类：沿用原有规则级切分（章/条/编号），若无标题则语义分割
		if !splitter.HasRuleHeaders(article.Content) {
			global.Log.Info("规范类未检测到标题，切换为语义分割", "article_id", article.ID)
			splitterSemantic, err := semantic.NewSplitter(ctx, &semantic.Config{
				Embedding:    embedders,
				BufferSize:   2,
				MinChunkSize: 100,
				Separators:   []string{"\n", ".", "?", "!", "。", "<p>"},
				Percentile:   0.8,
			})
			if err != nil {
				return fmt.Errorf("初始化文本分割器失败: %w", err)
			}
			chunks, err := splitterSemantic.Transform(ctx, []*schema.Document{doc})
			if err != nil || len(chunks) == 0 {
				chunks = []*schema.Document{doc}
			}
			for i, chunk := range chunks {
				chunkID := utils.GenerateID()
				if strings.TrimSpace(chunk.Content) == "" {
					continue
				}
				documentChunk := &models.DocumentChunk{
					ID:        utils.GenerateID(),
					ChunkID:   chunkID,
					ArticleID: article.ID,
					Title:     article.Title,
					Content:   chunk.Content,
				}
				if _, err := global.DB.Insert(documentChunk); err != nil {
					return fmt.Errorf("保存文档块失败 (块 %d/%d): %w", i+1, len(chunks), err)
				}
				vectors, err := s.Embedder.EmbedStrings(ctx, []string{chunk.Content})
				if err != nil || len(vectors) == 0 {
					return fmt.Errorf("生成向量嵌入失败 (块 %d/%d): %w", i+1, len(chunks), err)
				}
				vectorData, err := json.Marshal(vectors[0])
				if err != nil {
					return fmt.Errorf("序列化向量数据失败 (块 %d/%d): %w", i+1, len(chunks), err)
				}
				vector := &models.Vector{ID: utils.GenerateID(), ChunkID: chunkID, VectorData: string(vectorData)}
				if _, err = global.DB.Insert(vector); err != nil {
					return fmt.Errorf("保存向量失败 (块 %d/%d): %w", i+1, len(chunks), err)
				}
			}
			global.Log.Info("规范类语义分割完成(无标题)", "article_id", article.ID)
			return nil
		}
		global.Log.Info("规范类：规则级切分", "article_id", article.ID)
		units := splitter.SplitRules(article.Content)
		for i, u := range units {
			chunkID := utils.GenerateID()
			anchorsJSON, _ := sonic.Marshal(u.Anchors)
			aliasesJSON, _ := sonic.Marshal(u.Aliases)
			documentChunk := &models.DocumentChunk{
				ID:          utils.GenerateID(),
				ChunkID:     chunkID,
				ArticleID:   article.ID,
				Title:       article.Title,
				Content:     u.Text,
				Page:        u.Page,
				CharStart:   u.CharStart,
				CharEnd:     u.CharEnd,
				SectionPath: u.SectionPath,
				RuleID:      u.RuleID,
				Anchors:     string(anchorsJSON),
				Aliases:     string(aliasesJSON),
				Fingerprint: splitter.Fingerprint(u.Text),
			}
			if _, err := global.DB.Insert(documentChunk); err != nil {
				return fmt.Errorf("保存规则文档块失败 (块 %d/%d): %w", i+1, len(units), err)
			}
			vectors, err := s.Embedder.EmbedStrings(ctx, []string{u.Text})
			if err != nil || len(vectors) == 0 {
				return fmt.Errorf("生成向量失败 (规则块 %d/%d): %w", i+1, len(units), err)
			}
			vectorData, err := json.Marshal(vectors[0])
			if err != nil {
				return fmt.Errorf("序列化向量失败 (规则块 %d/%d): %w", i+1, len(units), err)
			}
			vector := &models.Vector{ID: utils.GenerateID(), ChunkID: chunkID, VectorData: string(vectorData)}
			if _, err = global.DB.Insert(vector); err != nil {
				return fmt.Errorf("保存向量失败 (规则块 %d/%d): %w", i+1, len(units), err)
			}
		}
		global.Log.Info("规范类规则级分割完成", "article_id", article.ID, "chunks", len(units))
		return nil
	}

	// 非规范类：沿用语义分割
	splitterSemantic, err := semantic.NewSplitter(ctx, &semantic.Config{
		Embedding:    embedders,
		BufferSize:   2,
		MinChunkSize: 100,
		Separators:   []string{"\n", ".", "?", "!", "。", "<p>"},
		Percentile:   0.8,
	})
	if err != nil {
		global.Log.Error("初始化文本分割器失败", "error", err, "article_id", article.ID)
		return fmt.Errorf("初始化文本分割器失败: %w", err)
	}

	global.Log.Info("准备分割文档", "article_id", article.ID, "content_length", len(article.Content))

	chunks, err := splitterSemantic.Transform(ctx, []*schema.Document{doc})
	if err != nil {
		global.Log.Warn("语义分割器失败，使用降级策略（整体作为一个chunk）", "error", err, "article_id", article.ID)
		chunks = []*schema.Document{doc}
	}

	global.Log.Debug("使用文本分割器，成功分割文档", "chunks_count", len(chunks))
	if len(chunks) == 0 {
		return nil
	}
	for i, chunk := range chunks {
		chunkID := utils.GenerateID()
		if strings.TrimSpace(chunk.Content) == "" {
			continue
		}
		isAttachment := article.HasAttachment && strings.HasSuffix(strings.ToLower(article.Attachment), ".txt")
		documentChunk := &models.DocumentChunk{
			ID:           utils.GenerateID(),
			ChunkID:      chunkID,
			ArticleID:    article.ID,
			Title:        article.Title,
			Content:      chunk.Content,
			IsAttachment: isAttachment,
		}
		if _, err := global.DB.Insert(documentChunk); err != nil {
			return fmt.Errorf("保存文档块失败 (块 %d/%d): %w", i+1, len(chunks), err)
		}
		vectors, err := s.Embedder.EmbedStrings(ctx, []string{chunk.Content})
		if err != nil || len(vectors) == 0 {
			return fmt.Errorf("生成向量嵌入失败 (块 %d/%d): %w", i+1, len(chunks), err)
		}
		vectorData, err := json.Marshal(vectors[0])
		if err != nil {
			return fmt.Errorf("序列化向量数据失败 (块 %d/%d): %w", i+1, len(chunks), err)
		}
		vector := &models.Vector{ID: utils.GenerateID(), ChunkID: chunkID, VectorData: string(vectorData)}
		if _, err = global.DB.Insert(vector); err != nil {
			return fmt.Errorf("保存向量失败 (块 %d/%d): %w", i+1, len(chunks), err)
		}
	}

	global.Log.Info("文章处理完成", "article_id", article.ID, "title", article.Title)
	return nil
}

// processPDFDocument 处理PDF文档，提取内容并进行分割处理
func (s *KnowledgeServiceImpl) processPDFDocument(article *models.Article) error {
	if article == nil || !article.HasAttachment {
		return errors.New("无效的文章对象或没有附件")
	}

	ctx := context.Background()
	global.Log.Info("开始处理PDF文章", "article_id", article.ID, "title", article.Title)

	// 使用PDF解析器解析PDF文件
	pdfParser := NewPDFParser()
	// 解析PDF文件并获取完整内容
	pdfContent, err := pdfParser.Parse(article.Attachment)
	if err != nil {
		global.Log.Error("解析PDF文件内容失败", "error", err)
		return fmt.Errorf("解析PDF文件内容失败: %w", err)
	}

	// 将PDF原始内容存储到attachment_content字段
	article.AttachmentContent = pdfContent
	// 更新文章记录
	_, err = global.DB.ID(article.ID).Update(article)
	if err != nil {
		global.Log.Error("更新文章附件内容失败", "error", err)
		return fmt.Errorf("更新文章附件内容失败: %w", err)
	}

	// 解析PDF文件为文档块
	docs, err := pdfParser.ParseToDocuments(article.Attachment)
	if err != nil {
		global.Log.Error("解析PDF文件失败", "error", err)
		return fmt.Errorf("解析PDF文件失败: %w", err)
	}

	if len(docs) == 0 {
		global.Log.Warn("PDF文件解析结果为空", "article_id", article.ID)
		return nil
	}

	global.Log.Info("PDF文件解析成功", "docs_count", len(docs))

	// 初始化嵌入器
	embedder, err := embedding.NewEmbedder(ctx, &embedding.EmbeddingConfig{
		APIKey: global.Config.Embeddings.APIKey,    // 使用配置中的API密钥
		Model:  global.Config.Embeddings.Embedding, // 使用配置指定的嵌入模型
	})
	if err != nil {
		global.Log.Error("初始化嵌入器失败", "error", err)
		return fmt.Errorf("初始化嵌入器失败: %w", err)
	}

	// 规范类：法律法规使用“第X条”切分；其他规范类维持规则级切分
	if isNormativeArticle(article) {
		if isLegalArticle(article) {
			units := splitter.SplitArticles(pdfContent)
			if len(units) == 0 {
				global.Log.Info("PDF法律法规未检测到‘第X条’，切换语义分割", "article_id", article.ID)
				doc := &schema.Document{
					ID:      fmt.Sprintf("article_%d", article.ID),
					Content: pdfContent,
					MetaData: map[string]interface{}{
						"title": article.Title,
						"type":  article.Type,
					},
				}
				splitterSemantic, err := semantic.NewSplitter(ctx, &semantic.Config{
					Embedding:    embedder,
					BufferSize:   2,
					MinChunkSize: 100,
					Separators:   []string{"\n", ".", "?", "!", "。", "<p>"},
					Percentile:   0.8,
				})
				if err != nil {
					return fmt.Errorf("初始化文本分割器失败: %w", err)
				}
				chunks, err := splitterSemantic.Transform(ctx, []*schema.Document{doc})
				if err != nil || len(chunks) == 0 {
					chunks = []*schema.Document{doc}
				}
				for i, chunk := range chunks {
					chunkID := utils.GenerateID()
					if strings.TrimSpace(chunk.Content) == "" {
						continue
					}
					documentChunk := &models.DocumentChunk{
						ID:           utils.GenerateID(),
						ChunkID:      chunkID,
						ArticleID:    article.ID,
						Title:        article.Title,
						Content:      chunk.Content,
						IsAttachment: true,
					}
					if _, err := global.DB.Insert(documentChunk); err != nil {
						return fmt.Errorf("保存PDF文档块失败 (块 %d/%d): %w", i+1, len(chunks), err)
					}
					vectors, err := s.Embedder.EmbedStrings(ctx, []string{chunk.Content})
					if err != nil || len(vectors) == 0 {
						return fmt.Errorf("生成向量失败 (PDF块 %d/%d): %w", i+1, len(chunks), err)
					}
					vectorData, err := json.Marshal(vectors[0])
					if err != nil {
						return fmt.Errorf("序列化向量失败 (PDF块 %d/%d): %w", i+1, len(chunks), err)
					}
					vector := &models.Vector{ID: utils.GenerateID(), ChunkID: chunkID, VectorData: string(vectorData)}
					if _, err = global.DB.Insert(vector); err != nil {
						return fmt.Errorf("保存向量失败 (PDF块 %d/%d): %w", i+1, len(chunks), err)
					}
				}
				global.Log.Info("PDF语义分割完成(法律法规无‘第X条’)", "article_id", article.ID)
				return nil
			}
			// 否则走规则写入（下方通用写入逻辑）
		} else {
			// 其他规范类：规则级切分
			if !splitter.HasRuleHeaders(pdfContent) {
				global.Log.Info("PDF规范类未检测到标题，切换语义分割", "article_id", article.ID)
				doc := &schema.Document{
					ID:      fmt.Sprintf("article_%d", article.ID),
					Content: pdfContent,
					MetaData: map[string]interface{}{
						"title": article.Title,
						"type":  article.Type,
					},
				}
				splitterSemantic, err := semantic.NewSplitter(ctx, &semantic.Config{
					Embedding:    embedder,
					BufferSize:   2,
					MinChunkSize: 100,
					Separators:   []string{"\n", ".", "?", "!", "。", "<p>"},
					Percentile:   0.8,
				})
				if err != nil {
					return fmt.Errorf("初始化文本分割器失败: %w", err)
				}
				chunks, err := splitterSemantic.Transform(ctx, []*schema.Document{doc})
				if err != nil || len(chunks) == 0 {
					chunks = []*schema.Document{doc}
				}
				for i, chunk := range chunks {
					chunkID := utils.GenerateID()
					if strings.TrimSpace(chunk.Content) == "" {
						continue
					}
					documentChunk := &models.DocumentChunk{
						ID:           utils.GenerateID(),
						ChunkID:      chunkID,
						ArticleID:    article.ID,
						Title:        article.Title,
						Content:      chunk.Content,
						IsAttachment: true,
					}
					if _, err := global.DB.Insert(documentChunk); err != nil {
						return fmt.Errorf("保存PDF文档块失败 (块 %d/%d): %w", i+1, len(chunks), err)
					}
					vectors, err := s.Embedder.EmbedStrings(ctx, []string{chunk.Content})
					if err != nil || len(vectors) == 0 {
						return fmt.Errorf("生成向量失败 (PDF块 %d/%d): %w", i+1, len(chunks), err)
					}
					vectorData, err := json.Marshal(vectors[0])
					if err != nil {
						return fmt.Errorf("序列化向量失败 (PDF块 %d/%d): %w", i+1, len(chunks), err)
					}
					vector := &models.Vector{ID: utils.GenerateID(), ChunkID: chunkID, VectorData: string(vectorData)}
					if _, err = global.DB.Insert(vector); err != nil {
						return fmt.Errorf("保存向量失败 (PDF块 %d/%d): %w", i+1, len(chunks), err)
					}
				}
				global.Log.Info("PDF规范类语义分割完成(无标题)", "article_id", article.ID)
				return nil
			}
		}

		// 到这里说明：要么是法律法规且已切出units，要么是其他规范类并且有标题，统一写入
		var units []splitter.RuleUnit
		if isLegalArticle(article) {
			units = splitter.SplitArticles(pdfContent)
		} else {
			units = splitter.SplitRules(pdfContent)
		}
		if len(units) == 0 {
			global.Log.Warn("PDF规则级分割无结果", "article_id", article.ID)
			return nil
		}
		// 构建页起始rune偏移，用于标注每个规则的所在页
		pageStarts := make([]int, 0, len(docs))
		accum := 0
		for _, d := range docs {
			pageStarts = append(pageStarts, accum)
			accum += len([]rune(d.Content)) + 2 // 与 Parse 合并时的 "\n\n" 对齐
		}

		// 写入规则块
		for i, u := range units {
			// 计算所在页（以起始rune为基准）
			pageNum := 0
			for pi := 0; pi < len(pageStarts); pi++ {
				if u.CharStart < accum && (pi == len(pageStarts)-1 || u.CharStart < pageStarts[pi+1]) {
					pageNum = pi + 1 // 页码从1开始
					break
				}
			}

			chunkID := utils.GenerateID()
			anchorsJSON, _ := sonic.Marshal(u.Anchors)
			aliasesJSON, _ := sonic.Marshal(u.Aliases)
			documentChunk := &models.DocumentChunk{
				ID:           utils.GenerateID(),
				ChunkID:      chunkID,
				ArticleID:    article.ID,
				Title:        article.Title,
				Content:      u.Text,
				Page:         pageNum,
				CharStart:    u.CharStart,
				CharEnd:      u.CharEnd,
				SectionPath:  u.SectionPath,
				RuleID:       u.RuleID,
				Anchors:      string(anchorsJSON),
				Aliases:      string(aliasesJSON),
				Fingerprint:  splitter.Fingerprint(u.Text),
				IsAttachment: true,
			}
			if _, err := global.DB.Insert(documentChunk); err != nil {
				return fmt.Errorf("保存PDF规则文档块失败 (块 %d/%d): %w", i+1, len(units), err)
			}
			vectors, err := s.Embedder.EmbedStrings(ctx, []string{u.Text})
			if err != nil || len(vectors) == 0 {
				return fmt.Errorf("生成向量失败 (PDF规则块 %d/%d): %w", i+1, len(units), err)
			}
			vectorData, err := json.Marshal(vectors[0])
			if err != nil {
				return fmt.Errorf("序列化向量失败 (PDF规则块 %d/%d): %w", i+1, len(units), err)
			}
			vector := &models.Vector{ID: utils.GenerateID(), ChunkID: chunkID, VectorData: string(vectorData)}
			if _, err = global.DB.Insert(vector); err != nil {
				return fmt.Errorf("保存向量失败 (PDF规则块 %d/%d): %w", i+1, len(units), err)
			}
		}
		global.Log.Info("PDF整文规则级分割完成", "article_id", article.ID, "chunks", len(units))
		return nil
	}

	// 非规范类：整文按语义/段落切分
	// 先尝试语义切分
	doc := &schema.Document{
		ID:      fmt.Sprintf("article_%d", article.ID),
		Content: pdfContent,
		MetaData: map[string]interface{}{
			"title": article.Title,
			"type":  article.Type,
		},
	}
	splitterSemantic, err := semantic.NewSplitter(ctx, &semantic.Config{
		Embedding:    embedder,
		BufferSize:   2,
		MinChunkSize: 100,
		Separators:   []string{"\n\n", "\n", ".", "?", "!", "。", "<p>"},
		Percentile:   0.8,
	})
	if err != nil {
		return fmt.Errorf("初始化文本分割器失败: %w", err)
	}
	chunks, err := splitterSemantic.Transform(ctx, []*schema.Document{doc})
	if err != nil || len(chunks) == 0 {
		// 回退：按双换行段落切分
		paras := strings.Split(strings.TrimSpace(pdfContent), "\n\n")
		chunks = make([]*schema.Document, 0, len(paras))
		for idx, p := range paras {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			chunks = append(chunks, &schema.Document{ID: fmt.Sprintf("para_%d", idx), Content: p, MetaData: map[string]interface{}{"title": article.Title}})
		}
	}
	for i, chunk := range chunks {
		if strings.TrimSpace(chunk.Content) == "" {
			continue
		}
		chunkID := utils.GenerateID()
		documentChunk := &models.DocumentChunk{
			ID:           utils.GenerateID(),
			ChunkID:      chunkID,
			ArticleID:    article.ID,
			Title:        article.Title,
			Content:      chunk.Content,
			IsAttachment: true,
		}
		if _, err := global.DB.Insert(documentChunk); err != nil {
			return fmt.Errorf("保存PDF文档块失败 (块 %d/%d): %w", i+1, len(chunks), err)
		}
		vectors, err := s.Embedder.EmbedStrings(ctx, []string{chunk.Content})
		if err != nil || len(vectors) == 0 {
			return fmt.Errorf("生成向量失败 (PDF块 %d/%d): %w", i+1, len(chunks), err)
		}
		vectorData, err := json.Marshal(vectors[0])
		if err != nil {
			return fmt.Errorf("序列化向量失败 (PDF块 %d/%d): %w", i+1, len(chunks), err)
		}
		vector := &models.Vector{ID: utils.GenerateID(), ChunkID: chunkID, VectorData: string(vectorData)}
		if _, err = global.DB.Insert(vector); err != nil {
			return fmt.Errorf("保存向量失败 (PDF块 %d/%d): %w", i+1, len(chunks), err)
		}
	}

	global.Log.Info("PDF文章处理完成(非规范类：按语义/段落)", "article_id", article.ID, "title", article.Title)
	return nil
}

// processDocxDocument 处理DOCX文档，使用已经解析好的内容进行分割处理
func (s *KnowledgeServiceImpl) processDocxDocument(article *models.Article) error {
	if article == nil || !article.HasAttachment {
		return errors.New("无效的文章对象或没有附件")
	}

	ctx := context.Background()
	global.Log.Info("开始处理DOCX文章", "article_id", article.ID, "title", article.Title)

	// 检查文章内容是否已经在上传时解析；为空则本地兜底解析一次
	if strings.TrimSpace(article.Content) == "" {
		global.Log.Warn("DOCX文件内容为空，尝试本地兜底解析", "article_id", article.ID, "attachment", article.Attachment)
		factory := parser.NewParserFactory()
		ext := strings.ToLower(filepath.Ext(article.Attachment))
		p, err := factory.GetParser(ext)
		if err == nil && p != nil {
			if res, err2 := p.Parse(article.Attachment); err2 == nil && strings.TrimSpace(res) != "" {
				article.Content = res
				article.AttachmentContent = res
				if _, uerr := global.DB.ID(article.ID).Update(article); uerr != nil {
					global.Log.Warn("更新文章内容失败(兜底解析成功)", "article_id", article.ID, "error", uerr)
				}
			} else if err2 != nil {
				global.Log.Error("DOCX兜底解析失败", "error", err2)
				return fmt.Errorf("DOCX兜底解析失败: %w", err2)
			}
		} else if err != nil {
			global.Log.Warn("不支持的文档格式，无法兜底解析", "ext", ext, "error", err)
			return nil
		}
		if strings.TrimSpace(article.Content) == "" {
			global.Log.Warn("DOCX兜底解析后仍为空，终止处理", "article_id", article.ID)
			return nil
		}
	}

	// 规范类：优先规则切分，无编号则语义切分
	if isNormativeCategory(article.CategoryCode) {
		embedder, err := embedding.NewEmbedder(ctx, &embedding.EmbeddingConfig{
			APIKey: global.Config.Embeddings.APIKey,
			Model:  global.Config.Embeddings.Embedding,
		})
		if err != nil {
			return fmt.Errorf("初始化嵌入器失败: %w", err)
		}

		if isLegalArticle(article) {
			global.Log.Info("DOCX法律法规：按‘第X条’切分", "article_id", article.ID)
			units := splitter.SplitArticles(article.Content)
			if len(units) == 0 {
				// 语义切分回退
				doc := &schema.Document{
					ID:      fmt.Sprintf("article_%d", article.ID),
					Content: article.Content,
					MetaData: map[string]interface{}{
						"title": article.Title,
						"type":  article.Type,
					},
				}
				splitterSemantic, err := semantic.NewSplitter(ctx, &semantic.Config{
					Embedding:    embedder,
					BufferSize:   2,
					MinChunkSize: 100,
					Separators:   []string{"\n", ".", "?", "!", "。", "<p>"},
					Percentile:   0.8,
				})
				if err != nil {
					return fmt.Errorf("初始化文本分割器失败: %w", err)
				}
				chunks, err := splitterSemantic.Transform(ctx, []*schema.Document{doc})
				if err != nil || len(chunks) == 0 {
					chunks = []*schema.Document{doc}
				}
				for i, chunk := range chunks {
					chunkID := utils.GenerateID()
					if strings.TrimSpace(chunk.Content) == "" {
						continue
					}
					documentChunk := &models.DocumentChunk{
						ID:           utils.GenerateID(),
						ChunkID:      chunkID,
						ArticleID:    article.ID,
						Title:        article.Title,
						Content:      chunk.Content,
						IsAttachment: true,
					}
					if _, err := global.DB.Insert(documentChunk); err != nil {
						return fmt.Errorf("保存DOCX文档块失败 (块 %d/%d): %w", i+1, len(chunks), err)
					}
					vectors, err := s.Embedder.EmbedStrings(ctx, []string{chunk.Content})
					if err != nil || len(vectors) == 0 {
						return fmt.Errorf("生成向量失败 (DOCX块 %d/%d): %w", i+1, len(chunks), err)
					}
					vectorData, err := json.Marshal(vectors[0])
					if err != nil {
						return fmt.Errorf("序列化向量失败 (DOCX块 %d/%d): %w", i+1, len(chunks), err)
					}
					vector := &models.Vector{ID: utils.GenerateID(), ChunkID: chunkID, VectorData: string(vectorData)}
					if _, err = global.DB.Insert(vector); err != nil {
						return fmt.Errorf("保存向量失败 (DOCX块 %d/%d): %w", i+1, len(chunks), err)
					}
				}
				global.Log.Info("DOCX语义分割完成(法律法规无‘第X条’)", "article_id", article.ID)
				return nil
			}
			for i, u := range units {
				chunkID := utils.GenerateID()
				anchorsJSON, _ := sonic.Marshal(u.Anchors)
				aliasesJSON, _ := sonic.Marshal(u.Aliases)
				documentChunk := &models.DocumentChunk{
					ID:           utils.GenerateID(),
					ChunkID:      chunkID,
					ArticleID:    article.ID,
					Title:        article.Title,
					Content:      u.Text,
					Page:         u.Page,
					CharStart:    u.CharStart,
					CharEnd:      u.CharEnd,
					SectionPath:  u.SectionPath,
					RuleID:       u.RuleID,
					Anchors:      string(anchorsJSON),
					Aliases:      string(aliasesJSON),
					Fingerprint:  splitter.Fingerprint(u.Text),
					IsAttachment: true,
				}
				if _, err := global.DB.Insert(documentChunk); err != nil {
					return fmt.Errorf("保存DOCX法律法规文档块失败 (块 %d/%d): %w", i+1, len(units), err)
				}
				vectors, err := s.Embedder.EmbedStrings(ctx, []string{u.Text})
				if err != nil || len(vectors) == 0 {
					return fmt.Errorf("生成向量失败 (DOCX法律法规块 %d/%d): %w", i+1, len(units), err)
				}
				vectorData, err := json.Marshal(vectors[0])
				if err != nil {
					return fmt.Errorf("序列化向量失败 (DOCX法律法规块 %d/%d): %w", i+1, len(units), err)
				}
				vector := &models.Vector{ID: utils.GenerateID(), ChunkID: chunkID, VectorData: string(vectorData)}
				if _, err = global.DB.Insert(vector); err != nil {
					return fmt.Errorf("保存向量失败 (DOCX法律法规块 %d/%d): %w", i+1, len(units), err)
				}
			}
			global.Log.Info("DOCX法律法规‘第X条’切分完成", "article_id", article.ID)
			return nil
		}

		// 其他规范类：规则级切分，若无标题则语义切分
		if !splitter.HasRuleHeaders(article.Content) {
			doc := &schema.Document{
				ID:      fmt.Sprintf("article_%d", article.ID),
				Content: article.Content,
				MetaData: map[string]interface{}{
					"title": article.Title,
					"type":  article.Type,
				},
			}
			splitterSemantic, err := semantic.NewSplitter(ctx, &semantic.Config{
				Embedding:    embedder,
				BufferSize:   2,
				MinChunkSize: 100,
				Separators:   []string{"\n", ".", "?", "!", "。", "<p>"},
				Percentile:   0.8,
			})
			if err != nil {
				return fmt.Errorf("初始化文本分割器失败: %w", err)
			}
			chunks, err := splitterSemantic.Transform(ctx, []*schema.Document{doc})
			if err != nil || len(chunks) == 0 {
				chunks = []*schema.Document{doc}
			}
			for i, chunk := range chunks {
				chunkID := utils.GenerateID()
				if strings.TrimSpace(chunk.Content) == "" {
					continue
				}
				documentChunk := &models.DocumentChunk{
					ID:           utils.GenerateID(),
					ChunkID:      chunkID,
					ArticleID:    article.ID,
					Title:        article.Title,
					Content:      chunk.Content,
					IsAttachment: true,
				}
				if _, err := global.DB.Insert(documentChunk); err != nil {
					return fmt.Errorf("保存DOCX文档块失败 (块 %d/%d): %w", i+1, len(chunks), err)
				}
				vectors, err := embedder.EmbedStrings(ctx, []string{chunk.Content})
				if err != nil || len(vectors) == 0 {
					return fmt.Errorf("生成向量失败 (DOCX块 %d/%d): %w", i+1, len(chunks), err)
				}
				vectorData, err := json.Marshal(vectors[0])
				if err != nil {
					return fmt.Errorf("序列化向量失败 (DOCX块 %d/%d): %w", i+1, len(chunks), err)
				}
				vector := &models.Vector{ID: utils.GenerateID(), ChunkID: chunkID, VectorData: string(vectorData)}
				if _, err = global.DB.Insert(vector); err != nil {
					return fmt.Errorf("保存向量失败 (DOCX块 %d/%d): %w", i+1, len(chunks), err)
				}
			}
			global.Log.Info("DOCX规范类语义分割完成(无标题)", "article_id", article.ID)
			return nil
		}
		units := splitter.SplitRules(article.Content)
		for i, u := range units {
			chunkID := utils.GenerateID()
			anchorsJSON, _ := sonic.Marshal(u.Anchors)
			aliasesJSON, _ := sonic.Marshal(u.Aliases)
			documentChunk := &models.DocumentChunk{
				ID:           utils.GenerateID(),
				ChunkID:      chunkID,
				ArticleID:    article.ID,
				Title:        article.Title,
				Content:      u.Text,
				Page:         u.Page,
				CharStart:    u.CharStart,
				CharEnd:      u.CharEnd,
				SectionPath:  u.SectionPath,
				RuleID:       u.RuleID,
				Anchors:      string(anchorsJSON),
				Aliases:      string(aliasesJSON),
				Fingerprint:  splitter.Fingerprint(u.Text),
				IsAttachment: true,
			}
			if _, err := global.DB.Insert(documentChunk); err != nil {
				return fmt.Errorf("保存DOCX规则文档块失败 (块 %d/%d): %w", i+1, len(units), err)
			}
			vectors, err := s.Embedder.EmbedStrings(ctx, []string{u.Text})
			if err != nil || len(vectors) == 0 {
				return fmt.Errorf("生成向量失败 (DOCX块 %d/%d): %w", i+1, len(units), err)
			}
			vectorData, err := json.Marshal(vectors[0])
			if err != nil {
				return fmt.Errorf("序列化向量失败 (DOCX块 %d/%d): %w", i+1, len(units), err)
			}
			vector := &models.Vector{ID: utils.GenerateID(), ChunkID: chunkID, VectorData: string(vectorData)}
			if _, err = global.DB.Insert(vector); err != nil {
				return fmt.Errorf("保存向量失败 (DOCX块 %d/%d): %w", i+1, len(units), err)
			}
		}
		global.Log.Info("DOCX规则级分割完成", "article_id", article.ID, "chunks_count", len(units))
		return nil
	}

	// 非规范类：沿用语义分割
	doc := &schema.Document{
		ID:      fmt.Sprintf("article_%d", article.ID),
		Content: article.Content,
		MetaData: map[string]interface{}{
			"title": article.Title,
			"type":  article.Type,
		},
	}
	embedders, err := embedding.NewEmbedder(ctx, &embedding.EmbeddingConfig{
		APIKey: global.Config.Embeddings.APIKey,
		Model:  global.Config.Embeddings.Embedding,
	})
	if err != nil {
		return fmt.Errorf("初始化嵌入器失败: %w", err)
	}
	splitterSemantic, err := semantic.NewSplitter(ctx, &semantic.Config{
		Embedding:    embedders,
		BufferSize:   2,
		MinChunkSize: 100,
		Separators:   []string{"\n", ".", "?", "!", "。", "<p>"},
		Percentile:   0.8,
	})
	if err != nil {
		return fmt.Errorf("初始化文本分割器失败: %w", err)
	}
	chunks, err := splitterSemantic.Transform(ctx, []*schema.Document{doc})
	if err != nil || len(chunks) == 0 {
		chunks = []*schema.Document{doc}
	}
	for i, chunk := range chunks {
		chunkID := utils.GenerateID()
		if strings.TrimSpace(chunk.Content) == "" {
			continue
		}
		documentChunk := &models.DocumentChunk{
			ID:           utils.GenerateID(),
			ChunkID:      chunkID,
			ArticleID:    article.ID,
			Title:        article.Title,
			Content:      chunk.Content,
			IsAttachment: true,
		}
		if _, err := global.DB.Insert(documentChunk); err != nil {
			return fmt.Errorf("保存DOCX文档块失败 (块 %d/%d): %w", i+1, len(chunks), err)
		}
		vectors, err := embedders.EmbedStrings(ctx, []string{chunk.Content})
		if err != nil || len(vectors) == 0 {
			return fmt.Errorf("生成向量失败 (DOCX块 %d/%d): %w", i+1, len(chunks), err)
		}
		vectorData, err := json.Marshal(vectors[0])
		if err != nil {
			return fmt.Errorf("序列化向量失败 (DOCX块 %d/%d): %w", i+1, len(chunks), err)
		}
		vector := &models.Vector{ID: utils.GenerateID(), ChunkID: chunkID, VectorData: string(vectorData)}
		if _, err = global.DB.Insert(vector); err != nil {
			return fmt.Errorf("保存向量失败 (DOCX块 %d/%d): %w", i+1, len(chunks), err)
		}
	}

	global.Log.Info("DOCX文章处理完成", "article_id", article.ID, "title", article.Title)
	return nil
}

// isNormativeCategory 判断是否为规范/法律/合同模板类文档
func isNormativeCategory(code string) bool {
	c := strings.TrimSpace(strings.ToLower(code))
	switch c {
	case "medical_std", "legal", "contract_template":
		return true
	// 兼容常见别名/中文
	case "法规", "法律", "法律法规", "法", "law", "laws", "regulation", "regulations":
		return true
	case "合同", "合同模板", "contract", "template":
		return true
	default:
		return false
	}
}

// isLegalCategory 仅判断是否为法律法规类
func isLegalCategory(code string) bool {
	switch strings.TrimSpace(strings.ToLower(code)) {
	case "legal", "laws", "law", "法规", "法律", "法律法规", "fa", "fagui", "regulation", "regulations":
		return true
	default:
		return false
	}
}

// isNormativeArticle 结合 CategoryCode 与 Type 判断是否属于规范类
func isNormativeArticle(a *models.Article) bool {
	if a == nil {
		return false
	}
	c := strings.TrimSpace(strings.ToLower(a.CategoryCode))
	t := strings.TrimSpace(strings.ToLower(a.Type))
	if isNormativeCategory(c) {
		return true
	}
	switch t {
	case "medical_std", "legal", "contract_template", "法规", "法律", "法律法规", "合同", "合同模板", "law", "regulation", "regulations":
		return true
	default:
		return false
	}
}

// isLegalArticle 结合 CategoryCode 与 Type 判断是否为法律法规
func isLegalArticle(a *models.Article) bool {
	if a == nil {
		return false
	}
	if isLegalCategory(a.CategoryCode) {
		return true
	}
	t := strings.TrimSpace(strings.ToLower(a.Type))
	switch t {
	case "legal", "law", "laws", "法规", "法律", "法律法规", "fa", "fagui", "regulation", "regulations":
		return true
	default:
		return false
	}
}
