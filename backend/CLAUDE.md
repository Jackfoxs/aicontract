# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

这是一个基于Go语言的向量知识库系统，使用MySQL存储向量数据，提供文档管理、语义搜索和智能问答功能。系统采用CloudWeGo Hertz框架构建REST API，使用XORM作为ORM框架。

## 核心架构

### 三层架构设计
- **API层** (`/api/*_api/`): 处理HTTP请求，参数验证，调用服务层
- **服务层** (`/knowledge/`): 核心业务逻辑，包含文档处理、向量存储、语义搜索等
- **数据层** (`/models/`): 数据模型定义，通过XORM与MySQL交互

### 关键服务组件
- **文章服务** (`knowledge/article_service.go`): 文章CRUD操作，附件处理
- **文档服务** (`knowledge/document_service.go`): 文档分块，管理文档块与文章关联
- **向量服务** (`knowledge/embeddings/`): 调用Deepseek API生成向量，计算相似度
- **搜索引擎** (`knowledge/search_engine.go`): 基于向量相似度的语义搜索
- **问答服务** (`knowledge/answer_service.go`): 整合检索结果生成智能回答

## 项目结构规范

### API层组织结构
API层严格按照模块进行目录划分，每个模块独立一个包：
```
/api/
  ├── article_api/        # 文章模块
  │   ├── article_create.go   # 创建文章
  │   ├── article_update.go   # 更新文章
  │   ├── article_delete.go   # 删除文章
  │   ├── article_query.go    # 查询文章
  │   └── article_list.go     # 文章列表
  ├── document_api/       # 文档模块
  │   ├── document_upload.go  # 上传文档
  │   ├── document_parse.go   # 解析文档
  │   └── document_chunk.go   # 文档分块
  └── search_api/         # 搜索模块
      ├── search_semantic.go # 语义搜索
      └── search_answer.go   # 智能问答
```

### 文件职责原则
- **单一职责**：每个文件只负责一个具体的功能点
- **功能独立**：一个API操作对应一个独立文件
- **职责明确**：文件名即功能描述，避免功能混杂

### 命名规则规范

#### 文件命名
文件命名必须遵循：`模块前缀_操作名.go`
- 模块前缀：对应API所属的业务模块（如 article、document、search）
- 操作名：描述具体的业务操作（如 create、update、delete、query）

示例：
- `article_create.go` - 文章创建
- `document_upload.go` - 文档上传
- `search_semantic.go` - 语义搜索

#### 函数命名
API处理函数命名规范：`模块名 + 操作动词`
```go
// 文章模块
func ArticleCreate(c *app.RequestContext)
func ArticleUpdate(c *app.RequestContext)
func ArticleDelete(c *app.RequestContext)
func ArticleQuery(c *app.RequestContext)

// 文档模块
func DocumentUpload(c *app.RequestContext)
func DocumentParse(c *app.RequestContext)

// 搜索模块
func SearchSemantic(c *app.RequestContext)
func SearchAnswer(c *app.RequestContext)
```

#### 路由命名
路由路径遵循RESTful规范，使用模块名作为前缀：
```go
// 文章模块路由
article := router.Group("/api/article")
{
    article.POST("/create", article_api.ArticleCreate)
    article.PUT("/update/:id", article_api.ArticleUpdate)
    article.DELETE("/delete/:id", article_api.ArticleDelete)
    article.GET("/query/:id", article_api.ArticleQuery)
}
```

## 常用开发命令

```bash
# 运行服务（监听6636端口）
go run main.go

# 构建项目
go build -o knowledge-server main.go

# 更新依赖
go mod tidy

# 下载依赖
go mod download

# 运行单个文件测试（如果存在）
go test ./knowledge/...

# 格式化代码
go fmt ./...

# 检查代码
go vet ./...
```

## 数据库管理

系统使用XORM自动同步数据库结构，启动时会自动创建/更新以下表：
- `article`: 文章表
- `document_chunk`: 文档块表
- `vector`: 向量存储表

数据库连接配置在 `settings.yaml` 中，默认连接本地MySQL（端口3307）。

## API开发流程

1. 在 `/api/` 对应目录创建API处理函数
2. �� `/routers/` 添加路由注册
3. 在 `/knowledge/` 实现具体业务逻辑
4. 必要时在 `/models/` 添加新的数据模型

## 配置管理

主配置文件：`settings.yaml`
- MySQL连接配置
- 日志配置
- 聊天模型配置（Deepseek）
- 向量嵌入配置（ARK Embedding）

## 外部API依赖

- **Deepseek Chat API**: 用于智能问答生成
- **ARK Embedding API**: 用于文本向量化（字节跳动火山引擎）

确保在 `settings.yaml` 中配置正确的API密钥。

## 文件上传处理

上传的附件存储在 `/uploads/` 目录，系统支持PDF解析（使用 `dslipak/pdf` 库）。

## 日志系统

使用 Go 标准库 `log/slog` 进行结构化日志记录，配合自定义的 ColorHandler 实现彩色日志输出：
- 日志配置在 `settings.yaml` 中
- 支持控制台彩色输出和文件输出（输出到 `/log/` 目录）
- 全局日志实例：`global.Log`
- 自定义处理器：`core.ColorHandler`，支持显示文件名、行号和彩色级别标识