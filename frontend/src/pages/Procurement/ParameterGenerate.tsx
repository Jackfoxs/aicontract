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
} from '@arco-design/web-react'
import {
  IconCheckCircleFill,
  IconClockCircle,
  IconRefresh,
} from '@arco-design/web-react/icon'
import { getProcurementDetail } from '@/api/procurement'
import useProcurementStore from '@/store/procurementStore'
import ParameterTable from './components/ParameterTable'
import ComplianceIssueList from './components/ComplianceIssueList'
import SuggestionCard from './components/SuggestionCard'
import HistoryCaseCard from './components/HistoryCaseCard'
import './ParameterGenerate.css'

const { Title } = Typography
const { Row, Col } = Grid

const ParameterGenerate: React.FC = () => {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { currentRequirement, setCurrentRequirement } =
    useProcurementStore()
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (id) {
      loadRequirementDetail(Number(id))
    }
  }, [id])

  const loadRequirementDetail = async (requirementId: number) => {
    setLoading(true)
    try {
      const res = await getProcurementDetail(requirementId)
      if (res.data.code === 0) {
        setCurrentRequirement(res.data.data)
      } else {
        Message.error(res.data.msg || '加载失败')
      }
    } catch (error: any) {
      Message.error(error.message || '网络错误')
    } finally {
      setLoading(false)
    }
  }

  const handleRegenerate = () => {
    navigate('/procurement/input')
  }

  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: 100 }}>
        <Spin size={40} />
      </div>
    )
  }

  if (!currentRequirement) {
    return (
      <Card>
        <div style={{ textAlign: 'center', padding: 50 }}>
          <p>未找到需求数据</p>
          <Button type="primary" onClick={() => navigate('/procurement/input')}>
            创建新需求
          </Button>
        </div>
      </Card>
    )
  }

  const qualityColor =
    currentRequirement.analysis_quality >= 0.8
      ? 'green'
      : currentRequirement.analysis_quality >= 0.6
      ? 'orange'
      : 'red'

  return (
    <div className="parameter-generate-page">
      <Card>
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
          {/* 标题和统计 */}
          <div>
            <Title heading={3}>
              <IconCheckCircleFill style={{ color: '#00b42a', marginRight: 8 }} />
              需求分析结果
            </Title>
          </div>

          {/* 统计信息 */}
          <Row gutter={16}>
            <Col span={8}>
              <Card>
                <Statistic
                  title="分析质量评分"
                  value={(currentRequirement.analysis_quality * 100).toFixed(1)}
                  suffix="%"
                  precision={1}
                />
                <Typography.Text type="secondary" style={{ color: qualityColor }}>
                  {qualityColor === 'green'
                    ? '优秀'
                    : qualityColor === 'orange'
                    ? '良好'
                    : '需关注'}
                </Typography.Text>
              </Card>
            </Col>
            <Col span={8}>
              <Card>
                <Statistic
                  title="处理耗时"
                  value={currentRequirement.processing_time}
                  suffix="ms"
                  prefix={<IconClockCircle />}
                />
              </Card>
            </Col>
            <Col span={8}>
              <Card>
                <Statistic
                  title="AI成本"
                  value={currentRequirement.llm_cost}
                  suffix="元"
                  precision={4}
                />
              </Card>
            </Col>
          </Row>

          <Divider />

          {/* 原始需求 */}
          <Card title="原始需求描述" size="small">
            <Typography.Paragraph style={{ whiteSpace: 'pre-wrap' }}>
              {currentRequirement.requirement_text}
            </Typography.Paragraph>
            {currentRequirement.device_type && (
              <p>
                <strong>设备类型：</strong>
                {currentRequirement.device_type}
              </p>
            )}
            {currentRequirement.department && (
              <p>
                <strong>使用科室：</strong>
                {currentRequirement.department}
              </p>
            )}
            {currentRequirement.budget && (
              <p>
                <strong>预算金额：</strong>¥
                {currentRequirement.budget.toLocaleString()}
              </p>
            )}
          </Card>

          {/* 参数表格 */}
          <ParameterTable parameters={currentRequirement.generated_params} />

          {/* 合规性问题 */}
          <ComplianceIssueList issues={currentRequirement.compliance_issues} />

          {/* 优化建议 */}
          {currentRequirement.suggestions &&
            currentRequirement.suggestions.length > 0 && (
              <SuggestionCard suggestions={currentRequirement.suggestions} />
            )}

          {/* 历史案例 */}
          {currentRequirement.historical_cases &&
            currentRequirement.historical_cases.length > 0 && (
              <HistoryCaseCard cases={currentRequirement.historical_cases} />
            )}

          {/* 操作按钮 */}
          <Space>
            <Button
              type="primary"
              icon={<IconRefresh />}
              onClick={handleRegenerate}
            >
              重新生成
            </Button>
            <Button onClick={() => navigate('/procurement/list')}>
              返回列表
            </Button>
            <Button
              type="outline"
              onClick={() => navigate('/contract/upload', {
                state: { procurementId: currentRequirement.id }
              })}
            >
              上传合同审核
            </Button>
          </Space>
        </Space>
      </Card>
    </div>
  )
}

export default ParameterGenerate

