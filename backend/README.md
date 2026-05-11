# MySQL向量知识库系统

## 项目概述

本项目是一个基于Go语言开发的知识库系统，结合了MySQL数据库和向量检索技术，实现了文档管理、语义搜索和智能问答功能。系统支持文章上传、文档分块、向量化存储和基于语义的相似度检索，可用于构建企业知识库、智能客服等应用场景。

## 技术栈

- **后端框架**：Go + Gin
- **数据库**：MySQL
- **ORM框架**：XORM
- **向量嵌入**：ark Embedding Deepseek API
- **日志系统**：Logrus
- **配置管理**：YAML

## 系统架构

系统主要由以下几个核心模块组成：

1. **文章管理模块**：负责文章的上传、存储和管理
2. **文档处理模块**：将文章分割成文档块，便于精确检索
3. **向量存储模块**：将文档块转换为向量并存储
4. **语义搜索模块**：基于向量相似度的文档检索
5. **智能问答模块**：结合检索结果和大模型生成回答

## 目录结构

```
├── api                 # API接口层
│   ├── article_api     # 文章相关API
│   └── chat_api        # 聊天相关API
├── config              # 配置文件
├── core                # 核心组件
├── global              # 全局变量
├── knowledge           # 知识库服务
│   ├── service.go                # 知识库服务主接口定义
│   ├── embeddings/              # 向量嵌入相关功能
│   │   ├── embedder.go          # 向量嵌入器接口和实现
│   │   └── similarity.go        # 向量相似度计算工具
│   ├── document/                # 文档处理相关功能（计划中）
│   │   ├── processor.go         # 文档处理器接口和实现
│   │   ├── converter.go         # 文档格式转换工具
│   │   └── splitter.go          # 文档分割器
│   ├── search/                  # 搜索相关功能（计划中）
│   │   ├── engine.go            # 搜索引擎接口和实现
│   │   └── ranking.go           # 搜索结果排序工具
│   ├── answer/                  # 问答相关功能（计划中）
│   │   ├── generator.go         # 答案生成器接口和实现
│   │   └── formatter.go         # 答案格式化工具
│   └── storage/                 # 存储相关功能（计划中）
│       ├── article_repo.go      # 文章存储库
│       ├── chunk_repo.go        # 文档块存储库
│       └── vector_repo.go       # 向量存储库
├── models              # 数据模型
├── routers             # 路由配置
├── utils               # 工具函数
└── settings.yaml       # 配置文件
```

## 数据模型

### 文章模型 (Article)

```go
type Article struct {
	ID            uint64    `json:"id" xorm:"pk bigint 'id'"`
	Title         string    `json:"title" xorm:"varchar(255) notnull 'title'"`   // 文章标题
	Type          string    `json:"type" xorm:"varchar(50) notnull 'type'"`      // 文章类型
	Content       string    `json:"content" xorm:"text 'content'"`               // 文章内容
	Attachment    string    `json:"attachment" xorm:"varchar(255) 'attachment'"`  // 附件路径
	HasAttachment bool      `json:"has_attachment" xorm:"bool 'has_attachment'"`  // 是否有附件
	CreatedAt     time.Time `json:"created_at" xorm:"created 'created_at'"`      // 创建时间
	UpdatedAt     time.Time `json:"updated_at" xorm:"updated 'updated_at'"`      // 更新时间
}
```

### 文档块模型 (DocumentChunk)

```go
type DocumentChunk struct {
	ID        uint64    `json:"id" xorm:"pk bigint 'id'"`
	ChunkID   uint64    `json:"chunk_id" xorm:"bigint notnull unique 'chunk_id'"`  // 文档块ID
	ArticleID uint64    `json:"article_id" xorm:"bigint notnull 'article_id'"`          // 关联文章ID
	Title     string    `json:"title" xorm:"varchar(255) 'title'"`                      // 文档块标题
	Content   string    `json:"content" xorm:"text 'content'"`                          // 文档块内容
	CreatedAt time.Time `json:"created_at" xorm:"created 'created_at'"`                 // 创建时间
	UpdatedAt time.Time `json:"updated_at" xorm:"updated 'updated_at'"`                 // 更新时间
}
```

### 向量模型 (Vector)

```go
type Vector struct {
	ID         uint64    `json:"id" xorm:"pk bigint 'id'"`
	ChunkID    uint64    `json:"chunk_id" xorm:"bigint notnull unique 'chunk_id'"`  // 关联文档块ID
	VectorData string    `json:"vector_data" xorm:"text 'vector_data'"`                  // 向量数据
	CreatedAt  time.Time `json:"created_at" xorm:"created 'created_at'"`                 // 创建时间
	UpdatedAt  time.Time `json:"updated_at" xorm:"updated 'updated_at'"`                 // 更新时间
}
```

## 主要功能

### 1. 文章管理

- 支持文章上传、存储和管理
- 支持附件上传和管理
- 文章分类管理

### 2. 文档处理

- 自动将文章分割成适当大小的文档块
- 为每个文档块生成唯一ID
- 维护文档块与原文章的关联关系

### 3. 向量存储

- 使用Deepseek API将文档块转换为向量
- 在MySQL中存储向量数据
- 支持向量数据的更新和管理

### 4. 语义搜索

- 基于向量相似度的文档检索
- 支持按相似度排序
- 支持限制返回结果数量

### 5. 智能问答

- 结合检索到的相关文档
- 使用大模型生成准确的回答
- 支持上下文理解和多轮对话

## API接口

### 文章API

- `POST /api/article/upload`：上传文章

### 聊天API

- `POST /api/chat/query`：发送聊天消息并获取回答

## 配置说明

系统配置在`settings.yaml`文件中，主要包括：

- MySQL数据库配置
- Redis配置
- 日志配置
- 聊天模型配置

## 启动方式

```bash
# 启动服务
go run main.go
```

默认服务端口为6636，可在代码中修改。

## 注意事项

- 确保MySQL服务已启动并创建了对应的数据库
- 确保已配置正确的Deepseek API密钥
- 首次运行时会自动创建数据表

## 知识库服务模块详细设计

### 模块概述

知识库服务模块是系统的核心组件，用于管理、检索和利用知识内容。它支持文档上传、内容分割、向量化存储、语义搜索和智能问答等功能。

### 模块职责划分

#### 1. 主服务接口 (service.go)

定义知识库服务的主要接口，作为整个模块的入口点。

#### 2. 向量嵌入 (embeddings/)

负责文本向量化和相似度计算：
- `embedder.go`: 提供向量嵌入功能，封装外部嵌入模型API
- `similarity.go`: 提供向量相似度计算工具

#### 3. 文档处理 (document/)

负责文档的处理、转换和分割：
- `processor.go`: 文档处理的主要逻辑
- `converter.go`: 不同格式文档的转换工具
- `splitter.go`: 文档分割器，将长文档分割成语义块

#### 4. 搜索功能 (search/)

负责知识检索和相似度搜索：
- `engine.go`: 搜索引擎的主要实现
- `ranking.go`: 搜索结果的排序和优化

#### 5. 问答功能 (answer/)

负责基于检索结果生成回答：
- `generator.go`: 答案生成器，集成大语言模型
- `formatter.go`: 答案格式化，添加引用和格式处理

#### 6. 存储功能 (storage/)

负责数据的持久化和检索：
- `article_repo.go`: 文章数据的存储和检索
- `chunk_repo.go`: 文档块数据的存储和检索
- `vector_repo.go`: 向量数据的存储和检索

### 设计原则

1. **单一职责原则**：每个包和文件只负责一个明确的功能
2. **接口隔离原则**：通过接口定义清晰的模块边界
3. **依赖倒置原则**：高层模块不依赖低层模块，都依赖于抽象
4. **开闭原则**：对扩展开放，对修改关闭
5. **组合优于继承**：通过组合不同的功能模块构建复杂系统

### 优化点

1. **模块化设计**：将大型服务拆分为小型、专注的组件
2. **接口定义**：通过接口实现松耦合设计
3. **依赖注入**：通过依赖注入实现组件的灵活组合
4. **错误处理**：统一的错误处理策略
5. **并发控制**：合理使用并发提高性能
6. **可测试性**：设计便于单元测试的组件结构

### 未来扩展

1. **向量数据库集成**：替换当前的全量加载方案
2. **多模态支持**：扩展到图像、音频等多模态内容
3. **分布式部署**：支持服务的水平扩展
4. **缓存机制**：添加多级缓存提高性能
5. **权限控制**：细粒度的访问控制机制