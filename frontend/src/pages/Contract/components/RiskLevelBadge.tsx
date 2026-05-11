import { Badge } from '@arco-design/web-react'

interface RiskLevelBadgeProps {
  level: 'High' | 'Medium' | 'Low'
  showText?: boolean
}

const RiskLevelBadge: React.FC<RiskLevelBadgeProps> = ({ level, showText = true }) => {
  const levelMap = {
    High: { color: 'red', text: '高风险' },
    Medium: { color: 'orangered', text: '中风险' },
    Low: { color: 'orange', text: '低风险' },
  }

  const config = levelMap[level]

  return showText ? (
    <Badge status={config.color as any} text={config.text} />
  ) : (
    <Badge status={config.color as any} />
  )
}

export default RiskLevelBadge

