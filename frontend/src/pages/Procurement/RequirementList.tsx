import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Card,
  Table,
  Button,
  Space,
  Message,
  Input,
  Select,
  Popconfirm,
  Badge,
  Typography,
} from '@arco-design/web-react'
import type { ColumnProps } from '@arco-design/web-react/es/Table'
import {
  IconPlus,
  IconEye,
  IconDelete,
  IconSearch,
} from '@arco-design/web-react/icon'
import { getProcurementList, deleteProcurementRequirement } from '@/api/procurement'
import type { ProcurementRequirement } from '@/types'
import './RequirementList.css'

const { Title } = Typography

const RequirementList: React.FC = () => {
  const navigate = useNavigate()
  const [data, setData] = useState<ProcurementRequirement[]>([])
  const [loading, setLoading] = useState(false)
  const [pagination, setPagination] = useState({
    current: 1,
    pageSize: 10,
    total: 0,
  })
  const [filters, setFilters] = useState({
    status: '',
    device_type: '',
  })

  useEffect(() => {
    loadData()
  }, [pagination.current, pagination.pageSize, filters])

  const loadData = async () => {
    setLoading(true)
    try {
      const res = await getProcurementList({
        page: pagination.current,
        page_size: pagination.pageSize,
        status: filters.status || undefined,
        device_type: filters.device_type || undefined,
      })

      if (res.data.code === 0) {
        setData(res.data.data.list || [])
        setPagination((prev) => ({
          ...prev,
          total: res.data.data.total,
        }))
      } else {
        Message.error(res.data.msg || '加载失败')
      }
    } catch (error: any) {
      Message.error(error.message || '网络错误')
    } finally {
      setLoading(false)
    }
  }

  const handleDelete = async (id: number) => {
    try {
      const res = await deleteProcurementRequirement(id)
      if (res.data.code === 0) {
        Message.success('删除成功')
        loadData()
      } else {
        Message.error(res.data.msg || '删除失败')
      }
    } catch (error: any) {
      Message.error(error.message || '网络错误')
    }
  }

  const statusMap = {
    draft: { text: '草稿', color: 'gray' },
    analyzing: { text: '分析中', color: 'blue' },
    completed: { text: '已完成', color: 'green' },
    failed: { text: '失败', color: 'red' },
  }

  const columns: ColumnProps<ProcurementRequirement>[] = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 80,
    },
    {
      title: '需求描述',
      dataIndex: 'requirement_text',
      ellipsis: true,
      width: 300,
      render: (text) => (
        <div style={{ maxWidth: 300, overflow: 'hidden', textOverflow: 'ellipsis' }}>
          {text}
        </div>
      ),
    },
    {
      title: '设备类型',
      dataIndex: 'device_type',
      width: 120,
      render: (text) => text || '-',
    },
    {
      title: '科室',
      dataIndex: 'department',
      width: 120,
      render: (text) => text || '-',
    },
    {
      title: '预算（万元）',
      dataIndex: 'budget',
      width: 120,
      render: (value) => (value ? (value / 10000).toFixed(2) : '-'),
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (status) => {
        const statusInfo = statusMap[status as keyof typeof statusMap]
        return (
          <Badge status={statusInfo.color as any} text={statusInfo.text} />
        )
      },
    },
    {
      title: '分析质量',
      dataIndex: 'analysis_quality',
      width: 100,
      render: (value) => (value ? `${(value * 100).toFixed(1)}%` : '-'),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      width: 180,
      render: (text) => new Date(text).toLocaleString('zh-CN'),
    },
    {
      title: '操作',
      width: 180,
      fixed: 'right',
      render: (_, record) => (
        <Space>
          <Button
            type="text"
            size="small"
            icon={<IconEye />}
            onClick={() => navigate(`/procurement/generate/${record.id}`)}
          >
            查看
          </Button>
          <Popconfirm
            title="确定要删除这条记录吗？"
            onOk={() => handleDelete(Number(record.id))}
          >
            <Button type="text" size="small" status="danger" icon={<IconDelete />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div className="requirement-list-page">
      <Card>
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <Title heading={3}>采购需求列表</Title>
            <Button
              type="primary"
              icon={<IconPlus />}
              onClick={() => navigate('/procurement/input')}
            >
              新建需求
            </Button>
          </div>

          {/* 筛选器 */}
          <Space>
            <Input
              placeholder="设备类型"
              prefix={<IconSearch />}
              allowClear
              style={{ width: 200 }}
              onChange={(value) =>
                setFilters((prev) => ({ ...prev, device_type: value }))
              }
            />
            <Select
              placeholder="状态"
              allowClear
              style={{ width: 150 }}
              onChange={(value) =>
                setFilters((prev) => ({ ...prev, status: value }))
              }
            >
              <Select.Option value="">全部</Select.Option>
              <Select.Option value="draft">草稿</Select.Option>
              <Select.Option value="analyzing">分析中</Select.Option>
              <Select.Option value="completed">已完成</Select.Option>
              <Select.Option value="failed">失败</Select.Option>
            </Select>
            <Button onClick={loadData}>搜索</Button>
          </Space>

          {/* 表格 */}
          <Table
            columns={columns}
            data={data}
            loading={loading}
            pagination={{
              ...pagination,
              showTotal: true,
              showJumper: true,
              sizeCanChange: true,
            }}
            onChange={(pagination) => {
              setPagination({
                current: Number(pagination.current) || 1,
                pageSize: Number(pagination.pageSize) || 10,
                total: Number(pagination.total) || 0,
              })
            }}
            scroll={{ x: 1400 }}
          />
        </Space>
      </Card>
    </div>
  )
}

export default RequirementList

