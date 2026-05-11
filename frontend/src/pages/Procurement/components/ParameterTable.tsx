import { Table, Card, Typography, Badge } from '@arco-design/web-react'
import type { ColumnProps } from '@arco-design/web-react/es/Table'

const { Title, Text } = Typography

interface ParameterTableProps {
  parameters: any
  showTitle?: boolean
  className?: string
}

const ParameterTable: React.FC<ParameterTableProps> = ({
  parameters,
  showTitle = true,
  className,
}) => {
  // 将参数对象转换为表格数据
  const parseParameters = (params: any): any[] => {
    if (!params) return []

    const tableData: any[] = []
    
    // 处理设备信息
    if (params.device_info) {
      tableData.push({
        category: '设备信息',
        key: '设备名称',
        value: params.device_info.name || '-',
      })
      if (params.device_info.model) {
        tableData.push({
          category: '设备信息',
          key: '型号',
          value: params.device_info.model,
        })
      }
      if (params.device_info.quantity) {
        tableData.push({
          category: '设备信息',
          key: '数量',
          value: `${params.device_info.quantity} ${params.device_info.unit || '台'}`,
        })
      }
    }

    // 处理技术参数
    if (params.technical_params) {
      Object.entries(params.technical_params).forEach(([key, value]) => {
        tableData.push({
          category: '技术参数',
          key,
          value: String(value),
        })
      })
    }

    // 处理合规要求
    if (params.compliance_requirements) {
      if (Array.isArray(params.compliance_requirements)) {
        params.compliance_requirements.forEach((req: string, index: number) => {
          tableData.push({
            category: '合规要求',
            key: `要求 ${index + 1}`,
            value: req,
          })
        })
      } else {
        Object.entries(params.compliance_requirements).forEach(([key, value]) => {
          tableData.push({
            category: '合规要求',
            key,
            value: String(value),
          })
        })
      }
    }

    // 处理参考标准
    if (params.reference_standards && Array.isArray(params.reference_standards)) {
      params.reference_standards.forEach((std: string, index: number) => {
        tableData.push({
          category: '参考标准',
          key: `标准 ${index + 1}`,
          value: std,
        })
      })
    }

    return tableData
  }

  const columns: ColumnProps[] = [
    {
      title: '类别',
      dataIndex: 'category',
      width: 120,
      render: (text) => <Badge status="processing" text={text} />,
    },
    {
      title: '参数名',
      dataIndex: 'key',
      width: 200,
    },
    {
      title: '参数值',
      dataIndex: 'value',
      render: (text) => <Text copyable>{text}</Text>,
    },
  ]

  const tableData = parseParameters(parameters)

  return (
    <Card className={className}>
      {showTitle && <Title heading={6}>生成的技术参数</Title>}
      <Table
        columns={columns}
        data={tableData}
        pagination={false}
        border
        stripe
      />
    </Card>
  )
}

export default ParameterTable

