import { useState, useRef, useEffect } from 'react'
import {
  Card,
  Input,
  Button,
  Space,
  Empty,
  Spin,
  Divider
} from '@arco-design/web-react'
import { IconSend, IconDelete } from '@arco-design/web-react/icon'
import { useChatStore } from '@/store'
import ChatMessage from './components/ChatMessage'
import './index.css'

const { TextArea } = Input

export default function Chat() {
  const {
    messages,
    loading,
    streaming,
    currentAnswer,
    currentDocuments,
    sendMessage,
    clearMessages
  } = useChatStore()

  const [input, setInput] = useState('')
  // 始终流式输出
  const messagesEndRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    scrollToBottom()
  }, [messages, currentAnswer])

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }

  const handleSend = async () => {
    if (!input.trim() || loading) return

    const query = input.trim()
    setInput('')
    await sendMessage(query, true)
  }

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  const handleClear = () => {
    clearMessages()
  }

  return (
    <div className="chat-container">
      <Card className="chat-card">
        <div className="chat-header">
          <div className="chat-title">智能问答助手</div>
          <Space>
            <Button
              type="outline"
              status="danger"
              icon={<IconDelete />}
              onClick={handleClear}
              disabled={messages.length === 0}
            >
              清空对话
            </Button>
          </Space>
        </div>

        <Divider style={{ margin: '16px 0' }} />

        <div className="chat-messages">
          {messages.length === 0 && !loading ? (
            <Empty
              description="暂无对话，开始提问吧"
              style={{ paddingTop: 80 }}
            />
          ) : (
            <>
              {messages.map((message, index) => (
                <ChatMessage key={index} message={message} />
              ))}
              
              {/* 显示当前正在生成的回答 */}
              {streaming && currentAnswer && (
                <div className="message-wrapper assistant">
                  <div className="message-content">
                    <div className="message-text">{currentAnswer}</div>
                    {currentDocuments.length > 0 && (
                      <div className="message-documents">
                        <Divider orientation="left" style={{ fontSize: 12 }}>
                          参考文档
                        </Divider>
                        {currentDocuments.map((doc, idx) => (
                          <div key={idx} className="document-item">
                            <div className="document-title">{doc.title}</div>
                            <div className="document-content">{doc.content}</div>
                          </div>
                        ))}
                      </div>
                    )}
                    <div className="typing-indicator">
                      <span></span>
                      <span></span>
                      <span></span>
                    </div>
                  </div>
                </div>
              )}
              
              {loading && !streaming && (
                <div className="loading-wrapper">
                  <Spin />
                </div>
              )}
              
              <div ref={messagesEndRef} />
            </>
          )}
        </div>

        <div className="chat-input-wrapper">
          <TextArea
            placeholder="请输入您的问题... (Shift+Enter 换行，Enter 发送)"
            value={input}
            onChange={setInput}
            onKeyPress={handleKeyPress}
            autoSize={{ minRows: 2, maxRows: 6 }}
            disabled={loading}
          />
          <Button
            type="primary"
            icon={<IconSend />}
            onClick={handleSend}
            loading={loading}
            disabled={!input.trim()}
            style={{ marginTop: 12 }}
          >
            发送
          </Button>
        </div>
      </Card>
    </div>
  )
}

