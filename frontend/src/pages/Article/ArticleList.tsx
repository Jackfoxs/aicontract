import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Card,
  Table,
  Button,
  Space,
  Input,
  Select,
  Popconfirm,
  Tag
} from '@arco-design/web-react'
import { IconPlus, IconSearch, IconEye, IconDelete } from '@arco-design/web-react/icon'
import { useArticleStore } from '@/store'
import UploadArticleModal from './components/UploadArticleModal'
import dayjs from 'dayjs'
import type { Article } from '@/types'
import './ArticleList.css'

const { Option } = Select

export default function ArticleList() {
  const navigate = useNavigate()
  const { articles, total, loading, fetchArticles, deleteArticle } = useArticleStore()
  
  const [keyword, setKeyword] = useState('')
  const [type, setType] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [uploadModalVisible, setUploadModalVisible] = useState(false)

  useEffect(() => {
    loadArticles()
  }, [page, pageSize])

  const loadArticles = () => {
    fetchArticles({ page, pageSize, keyword, type })
  }

  const handleSearch = () => {
    setPage(1)
    loadArticles()
  }

  const handleView = (record: Article) => {
    navigate(`/articles/${record.id}`)
  }

  const handleDelete = async (id: string) => {
    const success = await deleteArticle(id)
    if (success) {
      loadArticles()
    }
  }

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 100
    },
    {
      title: '标题',
      dataIndex: 'title',
      render: (title: string) => (
        <div className="article-title">{title}</div>
      )
    },
    {
      title: '类型',
      dataIndex: 'type',
      width: 150,
      render: (type: string) => (
        <Tag color="arcoblue">{type}</Tag>
      )
    },
    {
      title: '附件',
      dataIndex: 'has_attachment',
      width: 100,
      render: (hasAttachment: boolean) => (
        <Tag color={hasAttachment ? 'green' : 'gray'}>
          {hasAttachment ? '有' : '无'}
        </Tag>
      )
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      width: 180,
      render: (time: string) => dayjs(time).format('YYYY-MM-DD HH:mm:ss')
    },
    {
      title: '操作',
      width: 200,
      render: (_: any, record: Article) => (
        <Space>
          <Button
            type="text"
            size="small"
            icon={<IconEye />}
            onClick={() => handleView(record)}
          >
            查看
          </Button>
          <Popconfirm
            title="确认删除？"
            onOk={() => handleDelete(record.id)}
          >
            <Button
              type="text"
              status="danger"
              size="small"
              icon={<IconDelete />}
            >
              删除
            </Button>
          </Popconfirm>
        </Space>
      )
    }
  ]

  return (
    <div className="article-list-container">
      <Card>
        <div className="search-bar">
          <Space>
            <Input
              style={{ width: 300 }}
              placeholder="搜索标题或内容"
              value={keyword}
              onChange={setKeyword}
              onPressEnter={handleSearch}
              prefix={<IconSearch />}
            />
            <Select
              style={{ width: 150 }}
              placeholder="规范类型"
              allowClear
              value={type}
              onChange={setType}
            >
              <Option value="技术文档">技术文档</Option>
              <Option value="业务文档">业务文档</Option>
              <Option value="法律法规">法律法规</Option>
              <Option value="其他">其他</Option>
            </Select>
            <Button type="primary" onClick={handleSearch}>
              搜索
            </Button>
          </Space>
          <Button
            type="primary"
            icon={<IconPlus />}
            onClick={() => setUploadModalVisible(true)}
          >
            上传规范
          </Button>
        </div>

        <Table
          loading={loading}
          columns={columns}
          data={articles}
          pagination={{
            current: page,
            pageSize,
            total,
            showTotal: true,
            sizeCanChange: true,
            onChange: (pageNumber, size) => {
              setPage(Number(pageNumber))
              setPageSize(Number(size))
            }
          }}
          rowKey="id"
        />
      </Card>

      <UploadArticleModal
        visible={uploadModalVisible}
        onClose={() => setUploadModalVisible(false)}
        onSuccess={() => {
          setUploadModalVisible(false)
          loadArticles()
        }}
      />
    </div>
  )
}

