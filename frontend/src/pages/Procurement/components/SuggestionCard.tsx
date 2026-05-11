import { Card, Alert, Space, Typography } from '@arco-design/web-react'
import { IconInfoCircle } from '@arco-design/web-react/icon'

const { Paragraph } = Typography

interface SuggestionCardProps {
  suggestions: string[]
  className?: string
}

const SuggestionCard: React.FC<SuggestionCardProps> = ({
  suggestions,
  className,
}) => {
  if (!suggestions || suggestions.length === 0) {
    return null
  }

  return (
    <Card className={className} title="优化建议">
      <Space direction="vertical" size="medium" style={{ width: '100%' }}>
        {suggestions.map((suggestion, index) => (
          <Alert
            key={index}
            type="info"
            icon={<IconInfoCircle />}
            content={
              <Paragraph style={{ margin: 0 }}>
                {index + 1}. {suggestion}
              </Paragraph>
            }
          />
        ))}
      </Space>
    </Card>
  )
}

export default SuggestionCard

