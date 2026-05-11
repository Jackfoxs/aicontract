# 知识库系统 API 文档

## 基本信息

- **服务地址**: `http://localhost:6636`
- **API前缀**: `/api`
- **响应格式**: JSON

## 通用响应格式

### 成功响应
```json
{
  "code": 0,
  "data": {},
  "msg": "成功"
}
```

### 失败响应
```json
{
  "code": 7,
  "data": {},
  "msg": "错误信息"
}
```

### 错误码说明
- `0`: 成功
- `7`: 通用错误
- `1001`: 系统错误
- `1002`: 参数错误
- `1003`: 数据库错误

---

## 文章管理 API

### 1. 上传文章
**接口地址**: `POST /api/article/upload`

**请求方式**: `multipart/form-data`

**请求参数**:
| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| title | string | 是 | 文章标题 |
| type | string | 是 | 文章类型 |
| content | string | 否 | 文章内容 |
| attachment | file | 否 | 附件文件 |

**请求示例**:
```bash
curl -X POST http://localhost:6636/api/article/upload \
  -F "title=测试文章" \
  -F "type=技术文档" \
  -F "content=这是文章内容" \
  -F "attachment=@document.pdf"
```

**响应示例**:
```json
{
  "code": 0,
  "data": {
    "article_id": 123456
  },
  "msg": "成功"
}
```

### 2. 获取文章列表
**接口地址**: `GET /api/article/list`

**请求参数**:
| 参数名 | 类型 | 必填 | 默认值 | 说明 |
|--------|------|------|--------|------|
| page | int | 否 | 1 | 页码 |
| pageSize | int | 否 | 10 | 每页数量 |
| keyword | string | 否 | - | 搜索关键词 |
| type | string | 否 | - | 文章类型 |

**请求示例**:
```bash
curl "http://localhost:6636/api/article/list?page=1&pageSize=10&keyword=测试&type=技术文档"
```

**响应示例**:
```json
{
  "code": 0,
  "data": {
    "list": [
      {
        "id": 123456,
        "title": "测试文章",
        "type": "技术文档",
        "content": "这是文章内容",
        "has_attachment": true,
        "created_at": "2025-01-16T10:30:00Z",
        "updated_at": "2025-01-16T10:30:00Z"
      }
    ],
    "total": 1,
    "page": 1,
    "pageSize": 10
  },
  "msg": "成功"
}
```

### 3. 获取文章详情
**接口地址**: `GET /api/article/{id}`

**路径参数**:
| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| id | string | 是 | 文章ID |

**请求示例**:
```bash
curl "http://localhost:6636/api/article/123456"
```

**响应示例**:
```json
{
  "code": 0,
  "data": {
    "id": 123456,
    "title": "测试文章",
    "type": "技术文档",
    "content": "这是文章内容",
    "attachment": "document.pdf",
    "attachment_content": "附件解析后的文本内容",
    "has_attachment": true,
    "created_at": "2025-01-16T10:30:00Z",
    "updated_at": "2025-01-16T10:30:00Z"
  },
  "msg": "成功"
}
```

### 4. 更新文章
**接口地址**: `PUT /api/article/{id}`

**请求方式**: `application/json`

**路径参数**:
| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| id | string | 是 | 文章ID |

**请求体**:
```json
{
  "title": "更新后的标题",
  "type": "技术文档",
  "content": "更新后的内容"
}
```

**请求示例**:
```bash
curl -X PUT http://localhost:6636/api/article/123456 \
  -H "Content-Type: application/json" \
  -d '{
    "title": "更新后的标题",
    "type": "技术文档", 
    "content": "更新后的内容"
  }'
```

**响应示例**:
```json
{
  "code": 0,
  "data": {},
  "msg": "成功"
}
```

### 5. 删除文章
**接口地址**: `DELETE /api/article/{id}`

**路径参数**:
| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| id | string | 是 | 文章ID |

**请求示例**:
```bash
curl -X DELETE http://localhost:6636/api/article/123456
```

**响应示例**:
```json
{
  "code": 0,
  "data": {},
  "msg": "成功"
}
```

---

## 聊天问答 API

### 1. 普通问答
**接口地址**: `POST /api/chat/query`

**请求方式**: `application/json`

**请求体**:
```json
{
  "query": "什么是人工智能？"
}
```

**请求示例**:
```bash
curl -X POST http://localhost:6636/api/chat/query \
  -H "Content-Type: application/json" \
  -d '{"query": "什么是人工智能？"}'
```

**响应示例**:
```json
{
  "code": 0,
  "data": {
    "answer": "人工智能是计算机科学的一个分支...",
    "documents": [
      {
        "id": 1,
        "article_id": 123456,
        "title": "AI基础知识",
        "content": "相关文档内容片段...",
        "created_at": "2025-01-16T10:30:00Z"
      }
    ]
  },
  "msg": "成功"
}
```

### 2. 流式问答
**接口地址**: `POST /api/chat/stream`

**请求方式**: `application/json`

**响应格式**: `text/event-stream` (SSE)

**请求体**:
```json
{
  "query": "什么是人工智能？"
}
```

**请求示例**:
```bash
curl -X POST http://localhost:6636/api/chat/stream \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d '{"query": "什么是人工智能？"}'
```

**响应示例**:
```
data: {"type":"documents","data":[{"id":1,"article_id":123456,"title":"AI基础知识","content":"相关文档内容片段..."}]}

data: {"type":"token","data":{"content":"人工"}}

data: {"type":"token","data":{"content":"智能"}}

data: {"type":"token","data":{"content":"是"}}

data: {"type":"done","data":{"message":"completed"}}
```

**流式事件类型**:
- `documents`: 返回相关文档
- `token`: 返回生成的文本片段
- `done`: 生成完成
- `error`: 发生错误

---

## 文档搜索 API

### 1. 搜索文档
**接口地址**: `POST /api/search/documents`

**请求方式**: `application/json`

**请求体**:
```json
{
  "query": "人工智能",
  "type": "技术文档",
  "page": 1,
  "pageSize": 10
}
```

**请求参数**:
| 参数名 | 类型 | 必填 | 默认值 | 说明 |
|--------|------|------|--------|------|
| query | string | 是 | - | 搜索关键词 |
| type | string | 否 | - | 文档类型 |
| page | int | 否 | 1 | 页码 |
| pageSize | int | 否 | 10 | 每页数量 |

**请求示例**:
```bash
curl -X POST http://localhost:6636/api/search/documents \
  -H "Content-Type: application/json" \
  -d '{
    "query": "人工智能",
    "type": "技术文档",
    "page": 1,
    "pageSize": 10
  }'
```

**响应示例**:
```json
{
  "code": 0,
  "data": {
    "list": [
      {
        "id": 1,
        "title": "AI基础知识",
        "type": "技术文档",
        "content": "人工智能相关内容片段...",
        "relevance": 0.8,
        "created_at": "2025-01-16T10:30:00Z"
      }
    ],
    "total": 1,
    "page": 1,
    "pageSize": 10
  },
  "msg": "成功"
}
```

---

## 前端集成示例

### JavaScript/TypeScript 示例

#### 1. 上传文章
```javascript
async function uploadArticle(title, type, content, file) {
  const formData = new FormData();
  formData.append('title', title);
  formData.append('type', type);
  formData.append('content', content);
  if (file) {
    formData.append('attachment', file);
  }

  const response = await fetch('/api/article/upload', {
    method: 'POST',
    body: formData
  });
  
  return await response.json();
}
```

#### 2. 获取文章列表
```javascript
async function getArticleList(page = 1, pageSize = 10, keyword = '', type = '') {
  const params = new URLSearchParams({
    page: page.toString(),
    pageSize: pageSize.toString(),
    ...(keyword && { keyword }),
    ...(type && { type })
  });

  const response = await fetch(`/api/article/list?${params}`);
  return await response.json();
}
```

#### 3. 聊天问答
```javascript
async function chat(query) {
  const response = await fetch('/api/chat/query', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({ query })
  });
  
  return await response.json();
}
```

#### 4. 流式聊天
```javascript
async function chatStream(query, onToken, onDocuments, onDone, onError) {
  const response = await fetch('/api/chat/stream', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Accept': 'text/event-stream'
    },
    body: JSON.stringify({ query })
  });

  const reader = response.body.getReader();
  const decoder = new TextDecoder();

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;

    const chunk = decoder.decode(value);
    const lines = chunk.split('\n');
    
    for (const line of lines) {
      if (line.startsWith('data: ')) {
        try {
          const event = JSON.parse(line.slice(6));
          
          switch (event.type) {
            case 'documents':
              onDocuments?.(event.data);
              break;
            case 'token':
              onToken?.(event.data.content);
              break;
            case 'done':
              onDone?.();
              return;
            case 'error':
              onError?.(event.data.message);
              return;
          }
        } catch (e) {
          console.error('解析SSE事件失败:', e);
        }
      }
    }
  }
}
```

#### 5. 搜索文档
```javascript
async function searchDocuments(query, type = '', page = 1, pageSize = 10) {
  const response = await fetch('/api/search/documents', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      query,
      type,
      page,
      pageSize
    })
  });
  
  return await response.json();
}
```

### React Hook 示例

```typescript
import { useState, useCallback } from 'react';

// 文章管理 Hook
export function useArticles() {
  const [loading, setLoading] = useState(false);
  const [articles, setArticles] = useState([]);
  const [total, setTotal] = useState(0);

  const fetchArticles = useCallback(async (params) => {
    setLoading(true);
    try {
      const result = await getArticleList(params.page, params.pageSize, params.keyword, params.type);
      if (result.code === 0) {
        setArticles(result.data.list);
        setTotal(result.data.total);
      }
    } finally {
      setLoading(false);
    }
  }, []);

  const uploadArticle = useCallback(async (data) => {
    setLoading(true);
    try {
      const result = await uploadArticle(data.title, data.type, data.content, data.file);
      return result;
    } finally {
      setLoading(false);
    }
  }, []);

  return {
    loading,
    articles,
    total,
    fetchArticles,
    uploadArticle
  };
}

// 聊天 Hook
export function useChat() {
  const [loading, setLoading] = useState(false);
  const [messages, setMessages] = useState([]);

  const sendMessage = useCallback(async (query) => {
    setLoading(true);
    try {
      const result = await chat(query);
      if (result.code === 0) {
        setMessages(prev => [...prev, {
          type: 'user',
          content: query
        }, {
          type: 'assistant',
          content: result.data.answer,
          documents: result.data.documents
        }]);
      }
    } finally {
      setLoading(false);
    }
  }, []);

  return {
    loading,
    messages,
    sendMessage
  };
}
```

---

## 注意事项

1. **文件上传**: 支持PDF等文档格式，系统会自动解析文档内容
2. **流式响应**: 使用SSE (Server-Sent Events) 实现实时流式输出
3. **错误处理**: 所有接口都遵循统一的错误响应格式
4. **分页**: 列表接口支持分页，默认每页10条记录
5. **搜索**: 支持关键词搜索和类型筛选
6. **CORS**: 流式接口已配置CORS支持跨域访问

## 联调建议

1. **开发环境**: 确保后端服务运行在 `http://localhost:6636`
2. **测试工具**: 推荐使用 Postman 或 curl 进行接口测试
3. **文件上传测试**: 准备一些PDF文档进行上传测试
4. **流式接口测试**: 注意处理SSE事件流的解析
5. **错误处理**: 前端需要根据返回的code字段判断请求是否成功