package knowledge

import (
	"backend/models"
	"context"
	"os"
)

// KnowledgeService 定义知识库服务的主要接口
type KnowledgeService interface {
	// UploadArticle 文档管理功能
	UploadArticle(title, articleType, content, categoryCode string, attachment *os.File, attachmentName string) (uint64, error)

	// GetArticleList 获取文章列表
	GetArticleList(page, pageSize int, keyword, articleType string) ([]*models.Article, int64, error)

	// GetArticleByID 根据ID获取文章详情
	GetArticleByID(id string) (*models.Article, error)

	// DeleteArticle 删除文章
	DeleteArticle(id string) error

	// UpdateArticle 更新文章
	UpdateArticle(id, title, articleType, content string) error

	// SearchDocuments 搜索文档
	SearchDocuments(query, docType string, page, pageSize int) ([]*models.DocumentChunk, int64, error)

	// ProcessDocument 文档处理功能
	ProcessDocument(article *models.Article) error

	// SearchSimilarDocuments 搜索功能
	SearchSimilarDocuments(query string, limit int) ([]*models.DocumentChunk, error)

	// GenerateAnswer 问答功能
	GenerateAnswer(ctx context.Context, query string, documents []*models.DocumentChunk) (string, error)

	// GenerateAnswerStream 流式问答功能
	GenerateAnswerStream(ctx context.Context, query string, documents []*models.DocumentChunk, onToken func(string)) error

	// ConvertToMarkdown 文档转换功能
	ConvertToMarkdown(filePath string, ext string) (string, error)
}

type KnowledgeServiceBase struct {
	// 依赖的子模块，将在重构后通过依赖注入方式设置
	Embedder        Embedder          // 向量嵌入器
	DocProcessor    DocumentProcessor // 文档处理器
	SearchEngine    SearchEngine      // 搜索引擎
	AnswerGenerator AnswerGenerator   // 答案生成器
	ArticleRepo     ArticleRepository // 文章存储库
	ChunkRepo       ChunkRepository   // 文档块存储库
	VectorRepo      VectorRepository  // 向量存储库
}

// Embedder 定义向量嵌入器接口
type Embedder interface {
	EmbedStrings(ctx context.Context, texts []string) ([][]float64, error)
}

// DocumentProcessor 定义文档处理器接口
type DocumentProcessor interface {
	Process(article *models.Article) error
	ConvertToMarkdown(filePath string, ext string) (string, error)
}

// SearchEngine 定义搜索引擎接口
type SearchEngine interface {
	Search(query string, limit int) ([]*models.DocumentChunk, error)
}

// AnswerGenerator 定义答案生成器接口
type AnswerGenerator interface {
	Generate(ctx context.Context, query string, documents []*models.DocumentChunk) (string, error)
}

// 存储库接口定义

// ArticleRepository 定义文章存储库接口
type ArticleRepository interface {
	Save(article *models.Article) error
	FindByID(id uint64) (*models.Article, error)
}

// ChunkRepository 定义文档块存储库接口
type ChunkRepository interface {
	Save(chunk *models.DocumentChunk) error
	FindByIDs(ids []uint64) ([]*models.DocumentChunk, error)
}

// VectorRepository 定义向量存储库接口
type VectorRepository interface {
	Save(vector *models.Vector) error
	FindAll() ([]models.Vector, error)
}

// NewKnowledgeService 创建知识库服务实例
// 在重构完成后，这里将使用依赖注入模式，接收各个子模块的实现
func NewKnowledgeService() (KnowledgeService, error) {
	// 创建KnowledgeServiceImpl实例
	impl, err := NewKnowledgeServiceImpl()
	if err != nil {
		return nil, err
	}
	// 返回接口实现
	return impl, nil
}

// ProcessDocument 实现KnowledgeService接口的ProcessDocument方法
func (s *KnowledgeServiceBase) ProcessDocument(article *models.Article) error {
	return s.DocProcessor.Process(article)
}
