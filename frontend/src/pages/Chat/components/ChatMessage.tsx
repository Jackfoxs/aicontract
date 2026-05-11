import { Divider } from '@arco-design/web-react'
import dayjs from 'dayjs'
import type { ChatMessage as ChatMessageType } from '@/types'

interface ChatMessageProps {
  message: ChatMessageType
}

export default function ChatMessage({ message }: ChatMessageProps) {
  return (
    <div className={`message-wrapper ${message.type}`}>
      <div className="message-content">
        <div className="message-text">{message.content}</div>
        
        {message.documents && message.documents.length > 0 && (
          <div className="message-documents">
            <Divider orientation="left" style={{ fontSize: 12 }}>
              参考文档 ({message.documents.length})
            </Divider>
            {message.documents.map((doc, index) => (
              <div key={index} className="document-item">
                <div className="document-title">{doc.title}</div>
                <div className="document-content">{doc.content}</div>
              </div>
            ))}
          </div>
        )}
        
        {message.timestamp && (
          <div className="message-time">
            {dayjs(message.timestamp).format('HH:mm:ss')}
          </div>
        )}
      </div>
    </div>
  )
}

