import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import {
  Card,
  Button,
  Space,
  Spin,
  Tag,
  Descriptions,
  Divider,
  Modal,
  Form,
  Input,
  Select,
} from '@arco-design/web-react'
import { IconLeft, IconEdit } from '@arco-design/web-react/icon'
import { useArticleStore } from '@/store'
import dayjs from 'dayjs'
import './ArticleDetail.css'

const FormItem = Form.Item
const TextArea = Input.TextArea
const { Option } = Select

export default function ArticleDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { currentArticle, loading, fetchArticleDetail, updateArticle } = useArticleStore()
  
  const [editModalVisible, setEditModalVisible] = useState(false)
  const [form] = Form.useForm()

  useEffect(() => {
    if (id) {
      fetchArticleDetail(id)
    }
  }, [id])

  useEffect(() => {
    if (currentArticle && editModalVisible) {
      form.setFieldsValue({
        title: currentArticle.title,
        type: currentArticle.type,
        content: currentArticle.content
      })
    }
  }, [currentArticle, editModalVisible])

  const handleEdit = () => {
    setEditModalVisible(true)
  }

  const handleUpdate = async () => {
    try {
      const values = await form.validate()
      if (id) {
        const success = await updateArticle(id, values)
        if (success) {
          setEditModalVisible(false)
          fetchArticleDetail(id)
        }
      }
    } catch (error) {
      console.error('表单验证失败:', error)
    }
  }

  if (loading || !currentArticle) {
    return (
      <div className="article-detail-loading">
        <Spin size={40} />
      </div>
    )
  }

  return (
    <div className="article-detail-container">
      <Card>
        <div className="detail-header">
          <Button
            icon={<IconLeft />}
            onClick={() => navigate('/articles')}
          >
            返回列表
          </Button>
          <Button
            type="primary"
            icon={<IconEdit />}
            onClick={handleEdit}
          >
            编辑
          </Button>
          {id && (
            <Button
              type="outline"
              onClick={() => navigate(`/articles/${id}/chunks`)}
            >
              编辑切片
            </Button>
          )}
        </div>

        <div className="detail-title">{currentArticle.title}</div>

        <Space size="large" style={{ marginBottom: 24 }}>
          <Tag color="arcoblue">{currentArticle.type}</Tag>
          <span className="detail-meta">
            创建时间: {dayjs(currentArticle.created_at).format('YYYY-MM-DD HH:mm:ss')}
          </span>
          <span className="detail-meta">
            更新时间: {dayjs(currentArticle.updated_at).format('YYYY-MM-DD HH:mm:ss')}
          </span>
        </Space>

        {currentArticle.has_attachment && (
          <>
            <Divider orientation="left">附件信息</Divider>
            <Descriptions
              column={1}
              data={[
                {
                  label: '附件名称',
                  value: currentArticle.attachment || '-'
                }
              ]}
              style={{ marginBottom: 24 }}
            />
          </>
        )}

        <Divider orientation="left">规范内容</Divider>
        <div 
          className="detail-content"
          dangerouslySetInnerHTML={{ __html: currentArticle.content || '暂无内容' }}
        />

        {currentArticle.attachment_content && (
          <>
            <Divider orientation="left">附件内容</Divider>
            <div 
              className="detail-attachment-content"
              dangerouslySetInnerHTML={{ __html: currentArticle.attachment_content }}
            />
          </>
        )}
      </Card>

      <Modal
        title="编辑规范"
        visible={editModalVisible}
        onOk={handleUpdate}
        onCancel={() => setEditModalVisible(false)}
        style={{ width: 600 }}
      >
        <Form form={form} layout="vertical">
          <FormItem label="标题" field="title" rules={[{ required: true, message: '请输入标题' }]}> 
            <Input placeholder="请输入规范标题" />
          </FormItem>
          <FormItem label="类型" field="type" rules={[{ required: true, message: '请选择类型' }]}> 
            <Select placeholder="请选择规范类型">
              <Option value="技术文档">技术文档</Option>
              <Option value="业务文档">业务文档</Option>
              <Option value="法律法规">法律法规</Option>
              <Option value="其他">其他</Option>
            </Select>
          </FormItem>
          <FormItem label="内容" field="content">
            <TextArea
              placeholder="请输入规范内容"
              rows={10}
              autoSize={{ minRows: 10, maxRows: 20 }}
            />
          </FormItem>
        </Form>
      </Modal>
    </div>
  )
}

