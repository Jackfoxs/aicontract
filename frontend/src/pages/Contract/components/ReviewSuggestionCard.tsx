import { Card, Alert, Space, Tag } from '@arco-design/web-react'
import { IconInfoCircle } from '@arco-design/web-react/icon'
import type { ReviewSuggestion } from '@/types'

interface ReviewSuggestionCardProps {
  suggestions: ReviewSuggestion[]
  className?: string
}

const ReviewSuggestionCard: React.FC<ReviewSuggestionCardProps> = ({
  suggestions,
  className,
}) => {
  if (!suggestions || suggestions.length === 0) {
    return null
  }

  const typeTextMap = {
    Optimization: '优化建议',
    Clarification: '澄清建议',
    Standardize: '标准化建议',
  }

  const typeColorMap = {
    Optimization: 'blue',
    Clarification: 'cyan',
    Standardize: 'purple',
  }

  return (
    <Card className={className} title={`优化建议 (${suggestions.length}项)`}>
      <Space direction="vertical" size="medium" style={{ width: '100%' }}>
        {suggestions.map((item, index) => (
          <Alert
            key={index}
            type="info"
            icon={<IconInfoCircle />}
            title={
              <Space>
                <Tag color={typeColorMap[item.type]}>
                  {typeTextMap[item.type]}
                </Tag>
                {item.target_clause && (
                  <span style={{ fontSize: 12 }}>目标条款：{item.target_clause}</span>
                )}
              </Space>
            }
            content={item.description}
          />
        ))}
      </Space>
    </Card>
  )
}

export default ReviewSuggestionCard

