package knowledge

import (
	"backend/global"
	"backend/knowledge/embeddings"
	"backend/models"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// KnowledgeServiceImpl 知识库服务实现，实现KnowledgeService接口
type KnowledgeServiceImpl struct {
	KnowledgeServiceBase // 嵌入基础实现结构
}

// NewKnowledgeServiceImpl 创建知识库服务实现
func NewKnowledgeServiceImpl() (*KnowledgeServiceImpl, error) {
	ctx := context.Background()

	// 使用embeddings包中的NewArkEmbedder函数创建嵌入器
	embedder, err := embeddings.NewArkEmbedder(ctx)
	if err != nil {
		return nil, fmt.Errorf("初始化嵌入模型失败: %w", err)
	}

	return &KnowledgeServiceImpl{
		KnowledgeServiceBase: KnowledgeServiceBase{
			Embedder: embedder,
		},
	}, nil
}

// UploadArticle 实现KnowledgeService接口的UploadArticle方法
func (s *KnowledgeServiceImpl) UploadArticle(title, articleType, content, categoryCode string, attachment *os.File, attachmentName string) (uint64, error) {
	// 现在ArticleService接受Embedder接口而不是具体实现，提高了代码的解耦性
	articleService := &ArticleService{
		Embedder:         s.Embedder,
		KnowledgeService: s,
	}

	return articleService.UploadArticle(title, articleType, content, categoryCode, attachment, attachmentName)
}

// GetArticleList 获取文章列表
func (s *KnowledgeServiceImpl) GetArticleList(page, pageSize int, keyword, articleType string) ([]*models.Article, int64, error) {
	var articles []*models.Article

	// 构建查询条件
	session := global.DB.NewSession()
	defer session.Close()

	// 添加搜索条件
	if keyword != "" {
		session = session.Where("title LIKE ? OR content LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if articleType != "" {
		session = session.Where("type = ?", articleType)
	}

	// 获取总数
	total, err := session.Count(&models.Article{})
	if err != nil {
		return nil, 0, fmt.Errorf("获取文章总数失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err = session.OrderBy("created_at DESC").Limit(pageSize, offset).Find(&articles)
	if err != nil {
		return nil, 0, fmt.Errorf("获取文章列表失败: %w", err)
	}

	return articles, total, nil
}

// GetArticleByID 根据ID获取文章详情
func (s *KnowledgeServiceImpl) GetArticleByID(id string) (*models.Article, error) {
	// 将string类型的ID转换为uint64
	articleID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("无效的文章ID: %w", err)
	}

	var article models.Article
	has, err := global.DB.Where("id = ?", articleID).Get(&article)
	if err != nil {
		return nil, fmt.Errorf("获取文章详情失败: %w", err)
	}
	if !has {
		return nil, nil
	}
	return &article, nil
}

// DeleteArticle 删除文章
func (s *KnowledgeServiceImpl) DeleteArticle(id string) error {
	// 将string类型的ID转换为uint64
	articleID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return fmt.Errorf("无效的文章ID: %w", err)
	}

	// 先获取文章信息，记录附件路径，用于后续删除物理文件
	var article models.Article
	has, err := global.DB.Where("id = ?", articleID).Get(&article)
	if err != nil {
		return fmt.Errorf("查询文章失败: %w", err)
	}
	if !has {
		return fmt.Errorf("文章不存在")
	}

	attachmentPath := article.Attachment // 保存附件路径

	// 开启事务
	session := global.DB.NewSession()
	defer session.Close()

	if err := session.Begin(); err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}

	// 先获取相关的文档块ID列表
	var chunks []models.DocumentChunk
	err = session.Where("article_id = ?", articleID).Find(&chunks)
	if err != nil {
		session.Rollback()
		return fmt.Errorf("获取文档块失败: %w", err)
	}

	global.Log.Info("准备删除文章相关数据", "article_id", articleID, "chunks_count", len(chunks), "has_attachment", attachmentPath != "")

	// 删除相关的向量（通过chunk_id）
	for _, chunk := range chunks {
		affected, err := session.Where("chunk_id = ?", chunk.ChunkID).Delete(&models.Vector{})
		if err != nil {
			session.Rollback()
			return fmt.Errorf("删除向量失败: %w", err)
		}
		global.Log.Info("删除向量", "chunk_id", chunk.ChunkID, "affected", affected)
	}

	// 删除相关的文档块
	affected, err := session.Where("article_id = ?", articleID).Delete(&models.DocumentChunk{})
	if err != nil {
		session.Rollback()
		return fmt.Errorf("删除文档块失败: %w", err)
	}
	global.Log.Info("删除文档块", "article_id", articleID, "affected", affected)

	// 删除文章记录
	affected, err = session.Where("id = ?", articleID).Delete(&models.Article{})
	if err != nil {
		session.Rollback()
		return fmt.Errorf("删除文章失败: %w", err)
	}
	global.Log.Info("删除文章记录", "article_id", articleID, "affected", affected)

	// 提交事务
	if err := session.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	// 事务成功后，删除物理文件
	if attachmentPath != "" {
		if err := os.Remove(attachmentPath); err != nil {
			// 文件删除失败只记录警告，不影响整体删除操作
			global.Log.Warn("删除附件文件失败", "path", attachmentPath, "error", err)
		} else {
			global.Log.Info("删除附件文件成功", "path", attachmentPath)
		}
	}

	global.Log.Info("文章删除成功", "article_id", articleID)
	return nil
}

// UpdateArticle 更新文章
func (s *KnowledgeServiceImpl) UpdateArticle(id, title, articleType, content string) error {
	// 将string类型的ID转换为uint64
	articleID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return fmt.Errorf("无效的文章ID: %w", err)
	}

	article := &models.Article{
		Title:   title,
		Type:    articleType,
		Content: content,
	}

	affected, err := global.DB.Where("id = ?", articleID).Update(article)
	if err != nil {
		return fmt.Errorf("更新文章失败: %w", err)
	}

	global.Log.Info("文章更新成功", "article_id", articleID, "affected", affected)
	return nil
}

// SearchDocuments 搜索文档
func (s *KnowledgeServiceImpl) SearchDocuments(query, docType string, page, pageSize int) ([]*models.DocumentChunk, int64, error) {
	var chunks []*models.DocumentChunk

	// 构建查询条件
	session := global.DB.NewSession()
	defer session.Close()

	// 添加搜索条件
	if query != "" {
		session = session.Where("document_chunk.title LIKE ? OR document_chunk.content LIKE ?", "%"+query+"%", "%"+query+"%")
	}

	// 如果指定了文档类型，需要通过article表进行关联查询
	if docType != "" {
		session = session.Join("INNER", "article", "document_chunk.article_id = article.id").
			Where("article.type = ?", docType)
	}

	// 获取总数 - 需要重新构建session因为之前的session可能被修改
	countSession := global.DB.NewSession()
	defer countSession.Close()

	if query != "" {
		countSession = countSession.Where("document_chunk.title LIKE ? OR document_chunk.content LIKE ?", "%"+query+"%", "%"+query+"%")
	}
	if docType != "" {
		countSession = countSession.Join("INNER", "article", "document_chunk.article_id = article.id").
			Where("article.type = ?", docType)
	}

	total, err := countSession.Count(&models.DocumentChunk{})
	if err != nil {
		return nil, 0, fmt.Errorf("获取文档块总数失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err = session.OrderBy("document_chunk.created_at DESC").Limit(pageSize, offset).Find(&chunks)
	if err != nil {
		return nil, 0, fmt.Errorf("搜索文档失败: %w", err)
	}

	// 当关键词存在且无结果时，回退到向量检索（Top-K，再做分页截断）
	if query != "" && len(chunks) == 0 {
		engine := NewVectorSearchEngine(s.Embedder.(embeddings.Embedder))
		vecRes, vErr := engine.Search(query, pageSize)
		if vErr != nil {
			global.Log.Warn("向量检索失败，回退空结果", "error", vErr)
			return chunks, total, nil
		}
		// 直接返回向量Top-K，total 以Top-K数量呈现
		return vecRes, int64(len(vecRes)), nil
	}

	return chunks, total, nil
}

// ConvertToMarkdown 将文档转换为Markdown格式
func (s *KnowledgeServiceImpl) ConvertToMarkdown(filePath string, ext string) (string, error) {
	// 根据文件扩展名选择合适的转换器
	switch strings.ToLower(ext) {
	case ".pdf":
		// 使用PDF解析器处理PDF文件
		pdfParser := NewPDFParser()
		return pdfParser.Parse(filePath)
	case ".docx", ".doc":
		// TODO: 实现Word文档转换逻辑
		return "# Word文档内容\n\n这是从Word文档转换的内容。", nil
	default:
		return "", fmt.Errorf("不支持的文件类型: %s", ext)
	}
}
