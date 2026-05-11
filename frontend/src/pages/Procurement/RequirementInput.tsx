import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Card,
  Form,
  Input,
  InputNumber,
  Button,
  Message,
  Space,
  Typography,
} from '@arco-design/web-react'
import { IconRobot } from '@arco-design/web-react/icon'
import { analyzeProcurementRequirement } from '@/api/procurement'
import useProcurementStore from '@/store/procurementStore'
import './RequirementInput.css'

const { TextArea } = Input
const { Title, Paragraph } = Typography

const RequirementInput: React.FC = () => {
  const [form] = Form.useForm()
  const navigate = useNavigate()
  const { setAnalysisResult, setAnalyzing } = useProcurementStore()
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (values: any) => {
    setLoading(true)
    setAnalyzing(true)

    try {
      const res = await analyzeProcurementRequirement({
        requirement_text: values.requirement_text,
        device_type: values.device_type,
        department: values.department,
        budget: values.budget,
      })

      if (res.data.code === 0) {
        Message.success('需求分析完成！')
        setAnalysisResult(res.data.data)
        // 跳转到参数展示页
        navigate(`/procurement/generate/${res.data.data.requirement_id}`)
      } else {
        Message.error(res.data.msg || '分析失败')
      }
    } catch (error: any) {
      Message.error(error.message || '网络错误')
    } finally {
      setLoading(false)
      setAnalyzing(false)
    }
  }

  return (
    <div className="requirement-input-page">
      <Card>
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
          <div>
            <Title heading={3}>
              <IconRobot style={{ marginRight: 8 }} />
              AI 采购需求分析
            </Title>
            <Paragraph type="secondary">
              请描述您的采购需求，AI将自动生成详细的技术参数表，并进行合规性检查。
            </Paragraph>
          </div>

          <Form
            form={form}
            layout="vertical"
            onSubmit={handleSubmit}
            autoComplete="off"
          >
            <Form.Item
              label="需求描述"
              field="requirement_text"
              rules={[
                { required: true, message: '请输入需求描述' },
                { minLength: 20, message: '需求描述至少20个字符' },
              ]}
            >
              <TextArea
                placeholder="请详细描述您的采购需求，例如：&#10;需要采购一台3.0T核磁共振设备，用于神经外科临床诊断。要求：&#10;1. 磁场强度：3.0T&#10;2. 梯度场强度：≥50mT/m&#10;3. 支持功能性磁共振成像(fMRI)&#10;4. 配备专用头颅线圈&#10;5. 符合YY 0505-2012标准"
                style={{ minHeight: 200 }}
                showWordLimit
                maxLength={2000}
              />
            </Form.Item>

            <Form.Item
              label="设备类型（可选）"
              field="device_type"
              tooltip="例如：MRI、CT、超声等"
            >
              <Input placeholder="例如：MRI、CT、超声等" />
            </Form.Item>

            <Form.Item
              label="使用科室（可选）"
              field="department"
            >
              <Input placeholder="例如：神经外科、心内科等" />
            </Form.Item>

            <Form.Item
              label="预算金额（可选）"
              field="budget"
              tooltip="单位：元"
            >
              <InputNumber
                placeholder="请输入预算金额"
                min={0}
                step={100000}
                style={{ width: '100%' }}
                formatter={(value) =>
                  value ? `¥ ${value}`.replace(/\B(?=(\d{3})+(?!\d))/g, ',') : ''
                }
              />
            </Form.Item>

            <Form.Item>
              <Space>
                <Button
                  type="primary"
                  htmlType="submit"
                  loading={loading}
                  icon={<IconRobot />}
                  size="large"
                >
                  {loading ? 'AI分析中...' : '开始AI分析'}
                </Button>
                <Button
                  size="large"
                  onClick={() => navigate('/procurement/list')}
                >
                  查看历史记录
                </Button>
              </Space>
            </Form.Item>
          </Form>
        </Space>
      </Card>
    </div>
  )
}

export default RequirementInput

