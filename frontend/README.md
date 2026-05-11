# AI智能知识库前端

基于 React 18 + TypeScript + Arco Design 构建的智能知识库管理系统前端。

## 技术栈

- **框架**: React 18
- **语言**: TypeScript
- **UI库**: Arco Design (字节跳动)
- **状态管理**: Zustand
- **路由**: React Router v7
- **请求库**: Axios
- **工具库**: ahooks
- **构建工具**: Vite
- **包管理器**: npm

## 功能模块

### 1. 文章管理
- 文章列表查看（支持分页、搜索、类型筛选）
- 文章上传（支持文本内容和PDF/Word/TXT附件）
- 文章详情查看
- 文章编辑
- 文章删除

### 2. 智能问答
- 实时对话聊天界面
- 支持流式输出和普通输出两种模式
- 显示参考文档来源
- 对话历史记录
- 清空对话功能

### 3. 文档搜索
- 智能文档搜索
- 支持类型筛选
- 相关度评分展示
- 关键词高亮显示
- 分页浏览

## 快速开始

### 安装依赖

```bash
npm install
```

### 开发环境

```bash
npm run dev
```

访问 http://localhost:3000

### 构建生产版本

```bash
npm run build
```

### 预览生产版本

```bash
npm run preview
```

## 项目结构

```
frontend/
├── src/
│   ├── api/                  # API接口定义
│   │   ├── article.ts       # 文章相关接口
│   │   ├── chat.ts          # 聊天相关接口
│   │   └── search.ts        # 搜索相关接口
│   ├── components/          # 公共组件
│   ├── layouts/             # 布局组件
│   │   └── BasicLayout.tsx  # 基础布局
│   ├── pages/               # 页面组件
│   │   ├── Article/         # 文章管理页面
│   │   ├── Chat/            # 智能问答页面
│   │   └── Search/          # 文档搜索页面
│   ├── store/               # Zustand状态管理
│   │   ├── articleStore.ts  # 文章状态
│   │   ├── chatStore.ts     # 聊天状态
│   │   └── searchStore.ts   # 搜索状态
│   ├── types/               # TypeScript类型定义
│   ├── utils/               # 工具函数
│   │   └── request.ts       # Axios封装
│   ├── App.tsx              # 根组件
│   └── main.tsx             # 入口文件
├── package.json
├── vite.config.ts           # Vite配置
└── tsconfig.json            # TypeScript配置
```

## API 配置

前端通过 Vite 代理将 `/api` 请求转发到后端服务：

```typescript
// vite.config.ts
server: {
  port: 3000,
  proxy: {
    '/api': {
      target: 'http://localhost:6636',
      changeOrigin: true
    }
  }
}
```

确保后端服务运行在 `http://localhost:6636`

## 开发说明

### 添加新页面

1. 在 `src/pages` 下创建页面组件
2. 在 `src/App.tsx` 中添加路由配置
3. 如需要，在 `src/layouts/BasicLayout.tsx` 中添加菜单项

### 添加新接口

1. 在 `src/types` 中定义接口类型
2. 在 `src/api` 中添加接口函数
3. 如需要，在 `src/store` 中添加状态管理

### 状态管理

使用 Zustand 进行状态管理，每个模块独立管理状态：

```typescript
// 使用示例
import { useArticleStore } from '@/store'

function Component() {
  const { articles, loading, fetchArticles } = useArticleStore()
  
  useEffect(() => {
    fetchArticles()
  }, [])
  
  // ...
}
```

## 注意事项

1. 确保后端服务已启动
2. Node.js 版本建议 >= 18
3. 文件上传限制：10MB，支持 PDF、Word、TXT 格式
4. 开发时请遵循 ESLint 规范

## 浏览器支持

- Chrome >= 87
- Firefox >= 78
- Safari >= 14
- Edge >= 88

