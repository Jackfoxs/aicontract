package knowledge

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"backend/document/parser"
	"backend/global"
	"backend/models"
	"backend/utils"
)

type ArticleService struct {
	Embedder         Embedder
	KnowledgeService KnowledgeService
}

// UploadArticle 上传文章
func (s *ArticleService) UploadArticle(title, articleType, content, categoryCode string, attachment *os.File, attachmentName string) (uint64, error) {
	// 创建文章记录
	id := utils.GenerateID()
	article := &models.Article{
		ID:           id,
		Title:        title,
		Type:         articleType,
		Content:      content,
		CategoryCode: categoryCode,
	}

	parseStartTime := time.Now()

	// 处理附件
	if attachment != nil && attachmentName != "" {
		// 设置附件标志
		article.HasAttachment = true

		// 获取文件扩展名
		ext := strings.ToLower(filepath.Ext(attachmentName))

		// 创建上传目录
		uploadDir := "uploads"
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			return 0, fmt.Errorf("创建上传目录失败: %w", err)
		}

		// 新代码（获取格式化时间）
		timestamp := time.Now().Unix()
		fileName := fmt.Sprintf("%d_%s", timestamp, attachmentName)
		filePath := filepath.Join(uploadDir, fileName)

		// 保存附件
		destFile, err := os.Create(filePath)
		if err != nil {
			return 0, fmt.Errorf("创建文件失败: %w", err)
		}
		defer destFile.Close()

		// 复制文件内容
		if _, err := io.Copy(destFile, attachment); err != nil {
			return 0, fmt.Errorf("保存文件失败: %w", err)
		}

		// 设置附件路径
		article.Attachment = filePath

		// 使用新的文档解析器解析 PDF/DOC/DOCX/TXT/MD/HTML
		if ext == ".pdf" || ext == ".doc" || ext == ".docx" || ext == ".txt" || ext == ".md" || ext == ".html" || ext == ".htm" {
			// 创建解析器
			factory := parser.NewParserFactory()
			docParser, err := factory.GetParser(ext)
			if err != nil {
				global.Log.Warn("获取解析器失败，跳过文档解析", "error", err)
			} else {
				// 解析文档
				parseResult, err := docParser.ParseWithMetadata(filePath)

				// 记录解析日志
				parseLog := &models.DocumentParseLog{
					FilePath:  filePath,
					FileName:  attachmentName,
					FileType:  strings.TrimPrefix(ext, "."),
					ParseTime: int(time.Since(parseStartTime).Milliseconds()),
					Success:   err == nil,
				}

				if err != nil {
					parseLog.ErrorMsg = err.Error()
					global.Log.Error("文档解析失败", "file", filePath, "error", err)
				} else if parseResult != nil {
					// 填充解析结果
					parseLog.FileSize = parseResult.Metadata.FileSize
					parseLog.ParseMethod = parseResult.ParseMethod
					parseLog.QualityScore = parseResult.QualityScore
					parseLog.TextLength = parseResult.Metadata.TextLength
					parseLog.TableCount = parseResult.Metadata.TableCount

					// 若正文为空或仅为占位符，使用解析后的内容
					if content == "" || isTrivialContent(content) {
						article.Content = parseResult.Content
					}
					// 保存附件原始内容，便于前端查看/调试
					article.AttachmentContent = parseResult.Content

					// 保存元数据
					metadataJSON, _ := json.Marshal(parseResult.Metadata)
					article.Metadata = string(metadataJSON)

					global.Log.Info("文档解析成功",
						"file", filePath,
						"quality_score", parseResult.QualityScore,
						"text_length", parseResult.Metadata.TextLength,
						"parse_method", parseResult.ParseMethod,
					)
				}

				// 保存解析日志
				if _, logErr := global.DB.Insert(parseLog); logErr != nil {
					global.Log.Warn("保存解析日志失败", "error", logErr)
				}
			}
		}
	}

	// 开启事务，确保文章插入和文档处理的原子性
	session := global.DB.NewSession()
	defer session.Close()

	if err := session.Begin(); err != nil {
		return 0, fmt.Errorf("开启事务失败: %w", err)
	}

	// 保存文章到数据库
	if _, err := global.DB.Insert(article); err != nil {
		session.Rollback()
		return 0, fmt.Errorf("保存文章失败: %w", err)
	}

	// 提交事务（文章已保存）
	if err := session.Commit(); err != nil {
		return 0, fmt.Errorf("提交事务失败: %w", err)
	}

	// 处理文档分割和向量化（如果失败，会删除已保存的文章）
	if err := s.KnowledgeService.ProcessDocument(article); err != nil {
		global.Log.Error("处理文档失败，删除已保存的文章", "article_id", article.ID, "error", err)
		// 删除已插入的文章
		if _, delErr := global.DB.ID(article.ID).Delete(&models.Article{}); delErr != nil {
			global.Log.Error("删除文章失败", "article_id", article.ID, "error", delErr)
		}
		return 0, fmt.Errorf("处理文档失败: %w", err)
	}

	return article.ID, nil
}

// isTrivialContent 判断内容是否仅为空或编辑器占位符，无有效文本
func isTrivialContent(s string) bool {
	t := strings.TrimSpace(s)
	if t == "" {
		return true
	}
	switch t {
	case "<p><br></p>", "<p></p>", "<br/>", "<br>":
		return true
	}
	// 如果不包含任何字母/数字/非ASCII字符（中文等），视为无效
	for _, r := range t {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r > 127 {
			return false
		}
	}
	return true
}
