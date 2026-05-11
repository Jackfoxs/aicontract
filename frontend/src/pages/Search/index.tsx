import React, { useState } from 'react'
import {
  Card,
  Input,
  Space,
  Select,
  Empty,
  List,
  Tag,
  Pagination
} from '@arco-design/web-react'
import { useSearchStore } from '@/store'
import dayjs from 'dayjs'
import type { DocumentChunk } from '@/types'
import './index.css'

const { Option } = Select

export default function Search() {
  const { results, total, loading, search } = useSearchStore()
  
  const [query, setQuery] = useState('')
  const [type, setType] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize] = useState(10)

  const handleSearch = async () => {
    if (!query.trim()) return
    
    setPage(1)
    await search({
      query: query.trim(),
      type,
      page: 1,
      pageSize
    })
  }

  const handlePageChange = async (newPage: number) => {
    setPage(newPage)
    await search({
      query: query.trim(),
      type,
      page: newPage,
      pageSize
    })
  }

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleSearch()
    }
  }

  const renderRelevanceTag = (relevance?: number) => {
    if (!relevance) return null
    
    const score = Math.round(relevance * 100)
    let color = 'gray'
    
    if (score >= 80) color = 'green'
    else if (score >= 60) color = 'arcoblue'
    else if (score >= 40) color = 'orange'
    else color = 'red'
    
    return (
      <Tag color={color} size="small">
        相关度: {score}%
      </Tag>
    )
  }

  const suggestedKeywords = ['采购需求', '合同审核', '技术参数', '合规检查']

  const handleSuggestedClick = (keyword: string) => {
    setQuery(keyword)
  }

  return (
    <div className="search-container">
      <Card>
        <div className="search-header">
          <div className="search-input-wrapper">
            <Input.Search
              placeholder="搜索文档内容、合同条款、采购参数..."
              value={query}
              onChange={setQuery}
              onSearch={handleSearch}
              onPressEnter={handleKeyPress}
              loading={loading}
              searchButton="搜索"
              size="large"
            />
            <div className="search-filters">
              <Select
                style={{ width: 150 }}
                placeholder="文档类型"
                allowClear
                value={type}
                onChange={setType}
              >
                <Option value="技术文档">技术文档</Option>
                <Option value="业务文档">业务文档</Option>
                <Option value="法律法规">法律法规</Option>
                <Option value="其他">其他</Option>
              </Select>
            </div>
          </div>
        </div>

        {!query && results.length === 0 && (
          <div className="search-suggestions">
            <div className="suggestions-label">热门搜索：</div>
            <Space wrap>
              {suggestedKeywords.map((keyword) => (
                <Tag
                  key={keyword}
                  color="arcoblue"
                  style={{ cursor: 'pointer' }}
                  onClick={() => handleSuggestedClick(keyword)}
                >
                  {keyword}
                </Tag>
              ))}
            </Space>
          </div>
        )}

        {results.length > 0 && (
          <div className="search-info">
            找到 <span className="highlight">{total}</span> 个相关结果
          </div>
        )}

        <div className="search-results">
          {results.length === 0 ? (
            <Empty
              description={query ? '未找到相关文档，请尝试其他关键词' : '请输入关键词开始搜索'}
              style={{ paddingTop: 80 }}
            />
          ) : (
            <>
              <List
                dataSource={results}
                render={(item: DocumentChunk) => (
                  <List.Item key={item.id} className="search-result-item">
                    <div className="result-content">
                      <div className="result-header">
                        <div className="result-title">{item.title}</div>
                        <Space>
                          {renderRelevanceTag(item.relevance)}
                          <span className="result-time">
                            {dayjs(item.created_at).format('YYYY-MM-DD')}
                          </span>
                        </Space>
                      </div>
                      <div className="result-text">
                        {highlightKeyword(item.content, query)}
                      </div>
                    </div>
                  </List.Item>
                )}
              />
              
              {total > pageSize && (
                <div className="search-pagination">
                  <Pagination
                    current={page}
                    pageSize={pageSize}
                    total={total}
                    onChange={handlePageChange}
                    showTotal
                  />
                </div>
              )}
            </>
          )}
        </div>
      </Card>
    </div>
  )
}

// 高亮关键词
function highlightKeyword(text: string, keyword: string): React.ReactNode {
  if (!keyword) return text
  
  const parts = text.split(new RegExp(`(${keyword})`, 'gi'))
  
  return (
    <>
      {parts.map((part, index) => 
        part.toLowerCase() === keyword.toLowerCase() ? (
          <span key={index} className="highlight-text">
            {part}
          </span>
        ) : (
          <span key={index}>{part}</span>
        )
      )}
    </>
  )
}

