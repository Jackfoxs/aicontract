import { useState, useEffect } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import {
  Card,
  Upload,
  Form,
  Input,
  InputNumber,
  Button,
  Message,
  Space,
  Typography,
  Alert,
} from '@arco-design/web-react'
import { IconUpload, IconFile } from '@arco-design/web-react/icon'
import { uploadContract } from '@/api/contract'
import useContractStore from '@/store/contractStore'
import './ContractUpload.css'

const { Title, Paragraph } = Typography

const ContractUpload: React.FC = () => {
  const [form] = Form.useForm()
  const navigate = useNavigate()
  const location = useLocation()
  const { setUploadResponse, setUploading } = useContractStore()
  const [fileList, setFileList] = useState<any[]>([])
  const [uploading, setUploadingState] = useState(false)

  // 从路由state获取采购需求ID
  const procurementId = (location.state as any)?.procurementId

  // 处理粘贴上传
  useEffect(() => {
    const handlePaste = (e: ClipboardEvent) => {
      // 如果焦点在输入框内，不处理文件粘贴
      const target = e.target as HTMLElement
      if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA') {
        return
      }

      const items = e.clipboardData?.items
      if (!items) return

      for (let i = 0; i < items.length; i++) {
        const item = items[i]
        
        if (item.kind === 'file') {
          const file = item.getAsFile()
          if (!file) continue

          // 检查文件类型
          const allowedTypes = ['.pdf', '.doc', '.docx']
          const fileExtension = '.' + file.name.split('.').pop()?.toLowerCase()
          
          if (!allowedTypes.includes(fileExtension)) {
            Message.warning('仅支持 PDF、Word 格式的文件')
            continue
          }

          // 检查文件大小 (20MB)
          if (file.size > 20 * 1024 * 1024) {
            Message.warning('文件大小不能超过 20MB')
            continue
          }

          const newFile = {
            uid: Date.now().toString(),
            name: file.name,
            originFile: file,
            status: 'done'
          }

          // 添加到文件列表
          setFileList([newFile])
          Message.success(`已粘贴文件：${file.name}`)
          e.preventDefault()
          break
        }
      }
    }

    document.addEventListener('paste', handlePaste)
    return () => {
      document.removeEventListener('paste', handlePaste)
    }
  }, [])

  const handleSubmit = async (values: any) => {
    if (fileList.length === 0) {
      Message.warning('请先上传合同文件')
      return
    }

    setUploadingState(true)
    setUploading(true)

    const formData = new FormData()
    formData.append('file', fileList[0].originFile)
    
    if (values.contract_title) {
      formData.append('contract_title', values.contract_title)
    }
    
    if (values.procurement_id || procurementId) {
      formData.append('procurement_id', String(values.procurement_id || procurementId))
    }

    try {
      const res = await uploadContract(formData)

      if (res.data.code === 0) {
        Message.success('合同上传成功！')
        setUploadResponse(res.data.data)
        // 跳转到审核页面
        navigate(`/contract/review/${res.data.data.review_id}`)
      } else {
        Message.error(res.data.msg || '上传失败')
      }
    } catch (error: any) {
      Message.error(error.message || '网络错误')
    } finally {
      setUploadingState(false)
      setUploading(false)
    }
  }

  // beforeUpload 处理
  const handleBeforeUpload = (file: File) => {
    const isLt20M = file.size / 1024 / 1024 < 20
    if (!isLt20M) {
      Message.error('文件大小不能超过 20MB')
      return false
    }
    
    // 手动添加文件到列表（因为返回 false 不会触发 onChange）
    const newFile = {
      uid: Date.now().toString(),
      name: file.name,
      originFile: file,
      status: 'done'
    }
    
    setFileList([newFile])
    
    // 返回 false 阻止自动上传
    return false
  }

  return (
    <div className="contract-upload-page">
      <Card>
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
          <div>
            <Title heading={3}>
              <IconFile style={{ marginRight: 8 }} />
              合同上传与审核
            </Title>
            <Paragraph type="secondary">
              上传合同文件（支持PDF、Word格式），AI将自动进行合同解析和风险审核。
            </Paragraph>
          </div>

          {procurementId && (
            <Alert
              type="info"
              content={`已关联采购需求 ID: ${procurementId}，系统将进行一致性检查`}
            />
          )}

          <Form
            form={form}
            layout="vertical"
            onSubmit={handleSubmit}
            autoComplete="off"
            initialValues={{
              procurement_id: procurementId,
            }}
          >
            <Form.Item
              label="合同文件"
              required
              extra="支持PDF、DOCX格式，文件大小不超过20MB。可使用 Ctrl+V 粘贴文件"
            >
              <Upload
                drag
                accept=".pdf,.docx,.doc"
                fileList={fileList}
                onChange={setFileList}
                limit={1}
                onExceedLimit={() => {
                  Message.warning('只能上传一个文件')
                }}
                beforeUpload={handleBeforeUpload}
              >
                <div style={{ padding: 40, textAlign: 'center' }}>
                  <IconUpload style={{ fontSize: 48, color: '#165dff' }} />
                  <div style={{ marginTop: 16, fontSize: 16 }}>
                    点击、拖拽或粘贴文件到此区域上传
                  </div>
                  <div style={{ marginTop: 8, color: '#86909c', fontSize: 12 }}>
                    支持 PDF、Word 格式
                  </div>
                </div>
              </Upload>
            </Form.Item>

            <Form.Item
              label="合同标题（可选）"
              field="contract_title"
              tooltip="如果不填写，将使用文件名作为标题"
            >
              <Input placeholder="例如：XX设备采购合同" />
            </Form.Item>

            <Form.Item
              label="关联采购需求ID（可选）"
              field="procurement_id"
              tooltip="关联已有采购需求，系统将进行一致性检查"
            >
              <InputNumber
                placeholder="请输入采购需求ID"
                min={1}
                style={{ width: '100%' }}
              />
            </Form.Item>

            <Form.Item>
              <Space>
                <Button
                  type="primary"
                  htmlType="submit"
                  loading={uploading}
                  icon={<IconUpload />}
                  size="large"
                  disabled={fileList.length === 0}
                >
                  {uploading ? '上传中...' : '上传并开始审核'}
                </Button>
                <Button
                  size="large"
                  onClick={() => navigate('/contract/list')}
                >
                  查看审核记录
                </Button>
              </Space>
            </Form.Item>
          </Form>
        </Space>
      </Card>
    </div>
  )
}

export default ContractUpload

