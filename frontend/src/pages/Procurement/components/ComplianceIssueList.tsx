import { Card, Alert, Space } from '@arco-design/web-react'
import { IconExclamationCircleFill, IconCheckCircleFill } from '@arco-design/web-react/icon'
import type { ComplianceIssue } from '@/types'
import ComplianceTag from './ComplianceTag'

interface ComplianceIssueListProps {
  issues: ComplianceIssue[]
  className?: string
}

const ComplianceIssueList: React.FC<ComplianceIssueListProps> = ({
  issues,
  className,
}) => {
  if (!issues || issues.length === 0) {
    return (
      <Card className={className} title="合规性检查">
        <Alert
          type="success"
          icon={<IconCheckCircleFill />}
          content="恭喜！未发现合规性问题"
        />
      </Card>
    )
  }

  const severityTypeMap = {
    high: 'error',
    medium: 'warning',
    low: 'info',
  } as const

  return (
    <Card className={className} title="合规性检查">
      <Space direction="vertical" size="medium" style={{ width: '100%' }}>
        {issues.map((issue, index) => (
          <Alert
            key={index}
            type={severityTypeMap[issue.severity]}
            icon={<IconExclamationCircleFill />}
            title={
              <Space>
                <span>{issue.field}</span>
                <ComplianceTag severity={issue.severity} type={issue.type} />
              </Space>
            }
            content={
              <div>
                <p><strong>问题：</strong>{issue.description}</p>
                <p><strong>建议：</strong>{issue.suggestion}</p>
                {issue.reference && (
                  <p style={{ fontSize: 12, color: '#86909c' }}>
                    <strong>依据：</strong>{issue.reference}
                  </p>
                )}
              </div>
            }
          />
        ))}
      </Space>
    </Card>
  )
}

export default ComplianceIssueList

