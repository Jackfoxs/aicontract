import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import {
  Card,
  Space,
  Button,
  Message,
  Spin,
  Typography,
  Statistic,
  Grid,
  Divider,
  Alert,
  Progress,
} from '@arco-design/web-react'
import {
  IconCheckCircleFill,
  IconClockCircle,
  IconRefresh,
  IconDownload,
  IconExclamationCircleFill,
} from '@arco-design/web-react/icon'
import {
  getContractDetail,
  reviewContract,
  getContractReport,
} from '@/api/contract'
import useContractStore from '@/store/contractStore'
import RiskLevelBadge from './components/RiskLevelBadge'
import RiskItemCard from './components/RiskItemCard'
import ReviewSuggestionCard from './components/ReviewSuggestionCard'
import ExtractedFieldsCard from './components/ExtractedFieldsCard'
import './ContractReview.css'

const { Title } = Typography
const { Row, Col } = Grid

const ContractReview: React.FC = () => {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { currentReview, setCurrentReview, reviewing, setReviewing } =
    useContractStore()
  const [loading, setLoading] = useState(false)
  const [reviewTriggered, setReviewTriggered] = useState(false)

  useEffect(() => {
    if (id) {
      loadContractDetail(Number(id))
    }
  }, [id])

  const loadContractDetail = async (reviewId: number) => {
    setLoading(true)
    try {
      const res = await getContractDetail(reviewId)
      if (res.data.code === 0) {
        setCurrentReview(res.data.data)
        
        // 如果状态是pending且未触发审核，自动触发审核
        if (res.data.data.review_status === 'pending' && !reviewTriggered) {
          handleStartReview(reviewId)
        }
      } else {
        Message.error(res.data.msg || '加载失败')
      }
    } catch (error: any) {
      Message.error(error.message || '网络错误')
    } finally {
      setLoading(false)
    }
  }

  const handleStartReview = async (reviewId: number) => {
    setReviewTriggered(true)
    setReviewing(true)
    Message.info('开始智能审核，请稍候...')

    try {
      const res = await reviewContract(reviewId)
      if (res.data.code === 0) {
        Message.success('审核完成！')
        // 重新加载详情
        await loadContractDetail(reviewId)
      } else {
        Message.error(res.data.msg || '审核失败')
      }
    } catch (error: any) {
      Message.error(error.message || '网络错误')
    } finally {
      setReviewing(false)
    }
  }

  const handleDownloadReport = async () => {
    if (!id) return

    try {
      const res = await getContractReport(Number(id))
      if (res.data.code === 0) {
        // 创建Markdown文件下载
        const blob = new Blob([res.data.data.report_markdown], {
          type: 'text/markdown',
        })
        const url = URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        a.download = `合同审核报告_${id}.md`
        a.click()
        URL.revokeObjectURL(url)
        Message.success('报告下载成功')
      } else {
        Message.error(res.data.msg || '下载失败')
      }
    } catch (error: any) {
      Message.error(error.message || '网络错误')
    }
  }

  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: 100 }}>
        <Spin size={40} />
      </div>
    )
  }

  if (!currentReview) {
    return (
      <Card>
        <div style={{ textAlign: 'center', padding: 50 }}>
          <p>未找到合同数据</p>
          <Button type="primary" onClick={() => navigate('/contract/upload')}>
            上传新合同
          </Button>
        </div>
      </Card>
    )
  }

  const statusMap = {
    pending: { text: '待审核', color: 'gray' },
    reviewing: { text: '审核中', color: 'blue' },
    completed: { text: '已完成', color: 'green' },
    failed: { text: '失败', color: 'red' },
  }

  const statusInfo = statusMap[currentReview.review_status]

  return (
    <div className="contract-review-page">
      <Card>
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
          {/* 标题和状态 */}
          <div>
            <Title heading={3}>
              <IconCheckCircleFill
                style={{
                  color:
                    currentReview.review_status === 'completed'
                      ? '#00b42a'
                      : '#86909c',
                  marginRight: 8,
                }}
              />
              合同审核报告
            </Title>
            <div style={{ marginTop: 8 }}>
              <Space>
                <span>状态：{statusInfo.text}</span>
                {currentReview.review_status === 'completed' && (
                  <>
                    <Divider type="vertical" />
                    <span>总体风险：</span>
                    <RiskLevelBadge level={currentReview.overall_risk} />
                  </>
                )}
              </Space>
            </div>
          </div>

          {/* 审核中进度 */}
          {reviewing && (
            <Alert
              type="info"
              icon={<IconClockCircle />}
              content={
                <div>
                  <p>AI正在审核合同，这可能需要1-2分钟...</p>
                  <Progress percent={30} animation style={{ marginTop: 12 }} />
                </div>
              }
            />
          )}

          {/* 统计信息 */}
          {currentReview.review_status === 'completed' && (
            <Row gutter={16}>
              <Col span={6}>
                <Card>
                  <Statistic
                    title="风险项数量"
                    value={currentReview.risk_items?.length || 0}
                    suffix="项"
                  />
                  <Typography.Text
                    type="secondary"
                    style={{
                      color:
                        (currentReview.risk_items?.length || 0) > 0 ? '#ff7d00' : '#00b42a',
                    }}
                  >
                    {(currentReview.risk_items?.length || 0) > 0 ? '存在风险' : '风险较低'}
                  </Typography.Text>
                </Card>
              </Col>
              <Col span={6}>
                <Card>
                  <Statistic
                    title="优化建议"
                    value={currentReview.suggestions?.length || 0}
                    suffix="项"
                    prefix={<IconExclamationCircleFill />}
                  />
                </Card>
              </Col>
              <Col span={6}>
                <Card>
                  <Statistic
                    title="审核耗时"
                    value={currentReview.review_time || 0}
                    suffix="ms"
                    prefix={<IconClockCircle />}
                  />
                </Card>
              </Col>
              <Col span={6}>
                <Card>
                  <Statistic
                    title="AI成本"
                    value={currentReview.llm_cost || 0}
                    suffix="元"
                    precision={4}
                  />
                </Card>
              </Col>
            </Row>
          )}

          <Divider />

          {/* 合同基本信息 */}
          <Card title="合同基本信息" size="small">
            <p>
              <strong>合同标题：</strong>
              {currentReview.contract_title}
            </p>
            <p>
              <strong>合同文件：</strong>
              {currentReview.contract_path}
            </p>
            {currentReview.procurement_id && (
              <p>
                <strong>关联采购需求：</strong>
                <Button
                  type="text"
                  onClick={() =>
                    navigate(
                      `/procurement/generate/${currentReview.procurement_id}`
                    )
                  }
                >
                  #{currentReview.procurement_id}
                </Button>
              </p>
            )}
            <p>
              <strong>创建时间：</strong>
              {new Date(currentReview.created_at).toLocaleString('zh-CN')}
            </p>
          </Card>

          {/* 提取的关键字段 */}
          {currentReview.extracted_fields && (
            <ExtractedFieldsCard fields={currentReview.extracted_fields} />
          )}

          {/* 风险项 */}
          {currentReview.risk_items && currentReview.risk_items.length > 0 && (
            <RiskItemCard riskItems={currentReview.risk_items} />
          )}

          {/* 优化建议 */}
          {currentReview.suggestions && currentReview.suggestions.length > 0 && (
            <ReviewSuggestionCard suggestions={currentReview.suggestions} />
          )}

          {/* 操作按钮 */}
          <Space>
            {currentReview.review_status === 'completed' && (
              <Button
                type="primary"
                icon={<IconDownload />}
                onClick={handleDownloadReport}
              >
                下载报告
              </Button>
            )}
            {currentReview.review_status === 'pending' && !reviewing && (
              <Button
                type="primary"
                icon={<IconRefresh />}
                onClick={() => handleStartReview(Number(id))}
              >
                开始审核
              </Button>
            )}
            <Button onClick={() => navigate('/contract/list')}>返回列表</Button>
            <Button onClick={() => navigate('/contract/upload')}>
              上传新合同
            </Button>
          </Space>
        </Space>
      </Card>
    </div>
  )
}

export default ContractReview

