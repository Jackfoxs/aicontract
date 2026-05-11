import { Card, Alert, Space, Tag, Typography } from '@arco-design/web-react'
import { IconExclamationCircleFill } from '@arco-design/web-react/icon'
import type { RiskItem } from '@/types'

const { Paragraph } = Typography

interface RiskItemCardProps {
  riskItems: RiskItem[]
  className?: string
}

const RiskItemCard: React.FC<RiskItemCardProps> = ({ riskItems, className }) => {
  if (!riskItems || riskItems.length === 0) {
    return (
      <Card className={className} title="风险项分析">
        <Alert type="success" content="恭喜！未发现风险项" />
      </Card>
    )
  }

  const severityColorMap = {
    High: 'red',
    Medium: 'orangered',
    Low: 'orange',
  }

  const typeTextMap = {
    Compliance: '合规性',
    Consistency: '一致性',
    MissingClause: '缺失条款',
    AbnormalClause: '异常条款',
    LogicalConflict: '逻辑冲突',
  }

  const severityAlertMap = {
    High: 'error',
    Medium: 'warning',
    Low: 'info',
  } as const

  return (
    <Card className={className} title={`风险项分析 (${riskItems.length}项)`}>
      <Space direction="vertical" size="medium" style={{ width: '100%' }}>
        {riskItems.map((item, index) => (
          <Alert
            key={index}
            type={severityAlertMap[item.severity]}
            icon={<IconExclamationCircleFill />}
            title={
              <Space>
                <Tag color={severityColorMap[item.severity]}>
                  {item.severity} 风险
                </Tag>
                <Tag>{typeTextMap[item.type]}</Tag>
                {item.location && <span style={{ fontSize: 12 }}>位置：{item.location}</span>}
              </Space>
            }
            content={
              <div>
                <Paragraph style={{ marginBottom: 8 }}>
                  <strong>问题描述：</strong>
                  {item.description}
                </Paragraph>
                <Paragraph style={{ marginBottom: 8 }}>
                  <strong>修改建议：</strong>
                  {item.suggestion}
                </Paragraph>
                <Paragraph style={{ marginBottom: 0, fontSize: 12, color: '#86909c' }}>
                  <strong>依据：</strong>
                  {item.basis}
                </Paragraph>
              </div>
            }
          />
        ))}
      </Space>
    </Card>
  )
}

export default RiskItemCard

