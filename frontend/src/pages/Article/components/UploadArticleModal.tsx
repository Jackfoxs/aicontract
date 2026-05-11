import { useState, useEffect } from 'react'
import { Modal, Form, Input, Select, Upload, Message } from '@arco-design/web-react'
import { IconUpload } from '@arco-design/web-react/icon'
import { useArticleStore } from '@/store'
import { Editor, Toolbar } from '@wangeditor/editor-for-react'
import { IDomEditor, IEditorConfig, IToolbarConfig } from '@wangeditor/editor'
import '@wangeditor/editor/dist/css/style.css'

const FormItem = Form.Item
const { Option } = Select

interface UploadArticleModalProps {
  visible: boolean
  onClose: () => void
  onSuccess: () => void
}

export default function UploadArticleModal({ visible, onClose, onSuccess }: UploadArticleModalProps) {
  const [form] = Form.useForm()
  const { uploadArticle, loading } = useArticleStore()
  const [fileList, setFileList] = useState<any[]>([])
  const [editor, setEditor] = useState<IDomEditor | null>(null)
  const [html, setHtml] = useState('')

  // 编辑器工具栏配置
  const toolbarConfig: Partial<IToolbarConfig> = {
    toolbarKeys: [
      'headerSelect',
      'bold',
      'italic',
      'underline',
      'through',
      '|',
      'bulletedList',
      'numberedList',
      'todo',
      '|',
      'fontSize',
      'color',
      'bgColor',
      '|',
      'emotion',
      'insertLink',
      'insertTable',
      'codeBlock',
      '|',
      'undo',
      'redo'
    ]
  }

  // 编辑器配置
  const editorConfig: Partial<IEditorConfig> = {
    placeholder: '请输入规范内容...',
    MENU_CONF: {}
  }

  // 销毁编辑器
  useEffect(() => {
    return () => {
      if (editor == null) return
      editor.destroy()
      setEditor(null)
    }
  }, [editor])

  // 处理粘贴上传 - 仅在模态框打开时监听
  useEffect(() => {
    if (!visible) {
      // 模态框关闭时清空文件列表和编辑器内容
      setFileList([])
      setHtml('')
      if (editor) {
        editor.clear()
      }
      return
    }

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
          const allowedTypes = ['.pdf', '.doc', '.docx', '.txt']
          const fileExtension = '.' + file.name.split('.').pop()?.toLowerCase()
          
          if (!allowedTypes.includes(fileExtension)) {
            Message.warning('仅支持 PDF、Word、TXT 格式的文件')
            continue
          }

          // 检查文件大小 (10MB)
          if (file.size > 10 * 1024 * 1024) {
            Message.warning('文件大小不能超过 10MB')
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
  }, [visible])

  const handleSubmit = async () => {
    try {
      const values = await form.validate()
      
      // 获取富文本编辑器的HTML内容
      const content = html || ''
      
      const params = {
        title: values.title,
        type: values.type,
        content: content,
        attachment: fileList[0]?.originFile
      }

      const success = await uploadArticle(params)
      if (success) {
        form.resetFields()
        setFileList([])
        setHtml('')
        if (editor) {
          editor.clear()
        }
        onSuccess()
      }
    } catch (error) {
      console.error('表单验证失败:', error)
    }
  }

  const handleCancel = () => {
    form.resetFields()
    setFileList([])
    setHtml('')
    if (editor) {
      editor.clear()
    }
    onClose()
  }

  // beforeUpload 处理
  const handleBeforeUpload = (file: File) => {
    // 检查文件大小
    if (file.size > 10 * 1024 * 1024) {
      Message.warning('文件大小不能超过 10MB')
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
    <Modal
      title="上传规范"
      visible={visible}
      onOk={handleSubmit}
      onCancel={handleCancel}
      confirmLoading={loading}
      style={{ width: 600 }}
    >
      <Form form={form} layout="vertical">
        <FormItem
          label="标题"
          field="title"
          rules={[{ required: true, message: '请输入标题' }]}
        >
          <Input placeholder="请输入规范标题" />
        </FormItem>

        <FormItem
          label="类型"
          field="type"
          rules={[{ required: true, message: '请选择类型' }]}
        >
          <Select placeholder="请选择规范类型">
            <Option value="技术文档">技术文档</Option>
            <Option value="业务文档">业务文档</Option>
            <Option value="法律法规">法律法规</Option>
            <Option value="其他">其他</Option>
          </Select>
        </FormItem>

        <FormItem label="内容">
          <div style={{ border: '1px solid #e5e6eb', borderRadius: 4, overflow: 'hidden' }}>
            <Toolbar
              editor={editor}
              defaultConfig={toolbarConfig}
              mode="default"
              style={{ borderBottom: '1px solid #e5e6eb' }}
            />
            <Editor
              defaultConfig={editorConfig}
              value={html}
              onCreated={setEditor}
              onChange={editor => setHtml(editor.getHtml())}
              mode="default"
              style={{ minHeight: '300px', maxHeight: '500px', overflowY: 'auto' }}
            />
          </div>
        </FormItem>

        <FormItem label="附件">
          <Upload
            drag
            fileList={fileList}
            onChange={setFileList}
            limit={1}
            accept=".pdf,.doc,.docx,.txt"
            tip="支持 PDF、Word、TXT 格式，单个文件不超过 10MB。可使用 Ctrl+V 粘贴文件"
            beforeUpload={handleBeforeUpload}
          >
            <div style={{ 
              padding: '40px 20px',
              textAlign: 'center',
              backgroundColor: '#f7f8fa',
              border: '2px dashed #e5e6eb',
              borderRadius: 4,
              cursor: 'pointer',
              transition: 'all 0.3s'
            }}>
              <div>
                <IconUpload style={{ fontSize: 48, color: '#86909c' }} />
              </div>
              <div style={{ marginTop: 16, fontWeight: 600, fontSize: 14, color: '#1d2129' }}>
                点击、拖拽或粘贴文件到此处上传
              </div>
              <div style={{ marginTop: 8, color: '#86909c', fontSize: 12 }}>
                支持 PDF、Word、TXT 格式，单个文件不超过 10MB
              </div>
            </div>
          </Upload>
        </FormItem>
      </Form>
    </Modal>
  )
}

