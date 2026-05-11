import { Tag } from '@arco-design/web-react'
import type { ComplianceIssue } from '@/types'

interface ComplianceTagProps {
  severity: ComplianceIssue['severity']
  type?: ComplianceIssue['type']
}

const ComplianceTag: React.FC<ComplianceTagProps> = ({ severity, type }) => {
  const severityColorMap = {
    high: 'red',
    medium: 'orangered',
    low: 'orange',
  }

  const typeTextMap = {
    missing: '缺失',
    invalid: '无效',
    suboptimal: '待优化',
  }

  const severityTextMap = {
    high: '高',
    medium: '中',
    low: '低',
  }

  return (
    <Tag color={severityColorMap[severity]}>
      {type && `${typeTextMap[type]} - `}严重程度: {severityTextMap[severity]}
    </Tag>
  )
}

export default ComplianceTag

