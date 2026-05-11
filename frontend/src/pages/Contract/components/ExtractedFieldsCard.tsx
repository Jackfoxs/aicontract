import { Card, Descriptions, Divider, Typography } from '@arco-design/web-react'
import type { ExtractedFields } from '@/types'

const { Title } = Typography

interface ExtractedFieldsCardProps {
  fields: ExtractedFields
  className?: string
}

const ExtractedFieldsCard: React.FC<ExtractedFieldsCardProps> = ({
  fields,
  className,
}) => {
  if (!fields) {
    return null
  }

  return (
    <Card className={className} title="合同关键信息提取">
      {/* 参与方信息 */}
      {fields.parties_info && (
        <>
          <Title heading={6}>参与方信息</Title>
          <Descriptions
            column={1}
            data={[
              {
                label: '甲方名称',
                value: fields.parties_info.party_a?.name || '-',
              },
              {
                label: '甲方统一社会信用代码',
                value: fields.parties_info.party_a?.unified_social_credit_code || '-',
              },
              {
                label: '甲方地址',
                value: fields.parties_info.party_a?.address || '-',
              },
              {
                label: '乙方名称',
                value: fields.parties_info.party_b?.name || '-',
              },
              {
                label: '乙方统一社会信用代码',
                value: fields.parties_info.party_b?.unified_social_credit_code || '-',
              },
              {
                label: '乙方地址',
                value: fields.parties_info.party_b?.address || '-',
              },
            ]}
            labelStyle={{ fontWeight: 600, width: 200 }}
          />
          <Divider />
        </>
      )}

      {/* 金额信息 */}
      {fields.amount_info && (
        <>
          <Title heading={6}>金额信息</Title>
          <Descriptions
            column={2}
            data={[
              {
                label: '合同总金额',
                value: fields.amount_info.total_amount
                  ? `¥${fields.amount_info.total_amount.toLocaleString()}`
                  : '-',
              },
              {
                label: '货币单位',
                value: fields.amount_info.currency || '-',
              },
              {
                label: '大写金额',
                value: fields.amount_info.amount_in_words || '-',
                span: 2,
              },
            ]}
            labelStyle={{ fontWeight: 600 }}
          />
          <Divider />
        </>
      )}

      {/* 设备信息 */}
      {fields.device_info && (
        <>
          <Title heading={6}>设备信息</Title>
          <Descriptions
            column={2}
            data={[
              {
                label: '设备名称',
                value: fields.device_info.name || '-',
              },
              {
                label: '型号',
                value: fields.device_info.model || '-',
              },
              {
                label: '数量',
                value: fields.device_info.quantity
                  ? `${fields.device_info.quantity} ${fields.device_info.unit || '台'}`
                  : '-',
              },
            ]}
            labelStyle={{ fontWeight: 600 }}
          />
          <Divider />
        </>
      )}

      {/* 日期信息 */}
      {fields.contract_dates && (
        <>
          <Title heading={6}>日期信息</Title>
          <Descriptions
            column={2}
            data={[
              {
                label: '签订日期',
                value: fields.contract_dates.signing_date || '-',
              },
              {
                label: '生效日期',
                value: fields.contract_dates.effective_date || '-',
              },
              {
                label: '到期日期',
                value: fields.contract_dates.expiration_date || '-',
              },
              {
                label: '交付日期',
                value: fields.contract_dates.delivery_date || '-',
              },
            ]}
            labelStyle={{ fontWeight: 600 }}
          />
        </>
      )}

      {/* 置信度 */}
      <Divider />
      <div style={{ textAlign: 'right', color: '#86909c', fontSize: 12 }}>
        提取置信度：{(fields.confidence * 100).toFixed(1)}%
      </div>
    </Card>
  )
}

export default ExtractedFieldsCard

