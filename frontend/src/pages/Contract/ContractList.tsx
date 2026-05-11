import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Card,
  Table,
  Button,
  Space,
  Message,
  Select,
  Popconfirm,
  Typography,
} from '@arco-design/web-react'
import type { ColumnProps } from '@arco-design/web-react/es/Table'
import {
  IconPlus,
  IconEye,
  IconDelete,
} from '@arco-design/web-react/icon'
import { getContractList, deleteContractReview } from '@/api/contract'
import type { ContractReview } from '@/types'
import RiskLevelBadge from './components/RiskLevelBadge'
import './ContractList.css'

const { Title } = Typography

const ContractList: React.FC = () => {
  const navigate = useNavigate()
  const [data, setData] = useState<ContractReview[]>([])
  const [loading, setLoading] = useState(false)
  const [pagination, setPagination] = useState({
    current: 1,
    pageSize: 10,
    total: 0,
  })
  const [filters, setFilters] = useState({
    review_status: '',
    overall_risk: '',
  })

  useEffect(() => {
    loadData()
  }, [pagination.current, pagination.pageSize, filters])

  const loadData = async () => {
    setLoading(true)
    try {
      const res = await getContractList({
        page: pagination.current,
        page_size: pagination.pageSize,
        review_status: filters.review_status || undefined,
        overall_risk: filters.overall_risk || undefined,
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
      const res = await deleteContractReview(id)
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
    pending: { text: '待审核', color: 'gray' },
    reviewing: { text: '审核中', color: 'blue' },
    completed: { text: '已完成', color: 'green' },
    failed: { text: '失败', color: 'red' },
  }

  const columns: ColumnProps<ContractReview>[] = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 80,
    },
    {
      title: '合同标题',
      dataIndex: 'contract_title',
      ellipsis: true,
      width: 250,
    },
    {
      title: '关联采购需求',
      dataIndex: 'procurement_id',
      width: 150,
      render: (id) =>
        id ? (
          <Button
            type="text"
            size="small"
            onClick={() => navigate(`/procurement/generate/${id}`)}
          >
            #{id}
          </Button>
        ) : (
          '-'
        ),
    },
    {
      title: '审核状态',
      dataIndex: 'review_status',
      width: 120,
      render: (status) => {
        const statusInfo = statusMap[status as keyof typeof statusMap]
        return <span style={{ color: statusInfo.color }}>{statusInfo.text}</span>
      },
    },
    {
      title: '总体风险',
      dataIndex: 'overall_risk',
      width: 120,
      render: (risk) => (risk ? <RiskLevelBadge level={risk} /> : '-'),
    },
    {
      title: '风险项',
      dataIndex: 'risk_items',
      width: 100,
      render: (items) => (items ? `${items.length}项` : '-'),
    },
    {
      title: '审核耗时',
      dataIndex: 'review_time',
      width: 120,
      render: (time) => (time ? `${time}ms` : '-'),
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
            onClick={() => navigate(`/contract/review/${record.id}`)}
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
    <div className="contract-list-page">
      <Card>
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <Title heading={3}>合同审核记录</Title>
            <Button
              type="primary"
              icon={<IconPlus />}
              onClick={() => navigate('/contract/upload')}
            >
              上传合同
            </Button>
          </div>

          {/* 筛选器 */}
          <Space>
            <Select
              placeholder="审核状态"
              allowClear
              style={{ width: 150 }}
              onChange={(value) =>
                setFilters((prev) => ({ ...prev, review_status: value }))
              }
            >
              <Select.Option value="">全部</Select.Option>
              <Select.Option value="pending">待审核</Select.Option>
              <Select.Option value="reviewing">审核中</Select.Option>
              <Select.Option value="completed">已完成</Select.Option>
              <Select.Option value="failed">失败</Select.Option>
            </Select>
            <Select
              placeholder="风险等级"
              allowClear
              style={{ width: 150 }}
              onChange={(value) =>
                setFilters((prev) => ({ ...prev, overall_risk: value }))
              }
            >
              <Select.Option value="">全部</Select.Option>
              <Select.Option value="High">高风险</Select.Option>
              <Select.Option value="Medium">中风险</Select.Option>
              <Select.Option value="Low">低风险</Select.Option>
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

export default ContractList

