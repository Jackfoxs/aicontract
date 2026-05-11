import { Card, Descriptions, Space, Empty, Typography } from '@arco-design/web-react'

const { Title } = Typography

interface HistoryCaseCardProps {
  cases: any[]
  className?: string
}

const HistoryCaseCard: React.FC<HistoryCaseCardProps> = ({
  cases,
  className,
}) => {
  if (!cases || cases.length === 0) {
    return (
      <Card className={className} title="历史案例参考">
        <Empty description="暂无相关历史案例" />
      </Card>
    )
  }

  return (
    <Card className={className} title="历史案例参考">
      <Space direction="vertical" size="large" style={{ width: '100%' }}>
        {cases.map((caseItem, index) => (
          <Card key={index} size="small" hoverable>
            <Title heading={6}>案例 {index + 1}</Title>
            <Descriptions
              column={2}
              data={[
                {
                  label: '设备类型',
                  value: caseItem.device_type || '-',
                },
                {
                  label: '型号',
                  value: caseItem.model || '-',
                },
                {
                  label: '供应商',
                  value: caseItem.supplier || '-',
                },
                {
                  label: '采购单位',
                  value: caseItem.hospital || '-',
                },
                {
                  label: '科室',
                  value: caseItem.department || '-',
                },
                {
                  label: '年份',
                  value: caseItem.year || '-',
                },
                {
                  label: '采购金额',
                  value: caseItem.price ? `¥${caseItem.price.toLocaleString()}` : '-',
                },
                {
                  label: '相关性',
                  value: caseItem.relevance ? `${(caseItem.relevance * 100).toFixed(1)}%` : '-',
                },
              ]}
              labelStyle={{ fontWeight: 600 }}
            />
            {caseItem.parameters && (
              <div style={{ marginTop: 12 }}>
                <strong>关键参数：</strong>
                <pre style={{ 
                  background: '#f7f8fa', 
                  padding: 8, 
                  borderRadius: 4,
                  marginTop: 8,
                  fontSize: 12,
                }}>
                  {JSON.stringify(caseItem.parameters, null, 2)}
                </pre>
              </div>
            )}
          </Card>
        ))}
      </Space>
    </Card>
  )
}

export default HistoryCaseCard

