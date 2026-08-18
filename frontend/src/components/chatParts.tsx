/* ================= 聊天消息渲染（ChatPage 与悬浮球面板共用） ================= */
import { useState } from 'react';
import { CardContainer, ensureCardRenderers } from '../cards';
import type { ChatMessage } from '../lib/types';

ensureCardRenderers();

/* 单条消息：用户气泡（右对齐）/ 助手（左对齐，推理轨迹 + 文本 + 卡片，无头像） */
export function MessageView({ msg, onAsk }: { msg: ChatMessage; onAsk: (q: string) => void }) {
  const [traceOpen, setTraceOpen] = useState(false);
  if (msg.role === 'user') {
    return (
      <div className="cmsg user" data-mid={msg.id} data-role="user">
        <div className="cmsg-bubble">{msg.text}</div>
      </div>
    );
  }
  const thoughts = msg.thoughts || [];
  const running = thoughts.some(t => t.status === 'running');
  return (
    <div className="cmsg bot" data-mid={msg.id} data-role="assistant">
      <div className="cmsg-content">
        {thoughts.length > 0 && (
          <div className={`trace ${traceOpen ? 'open' : ''}`}>
            <div className="trace-head" onClick={() => setTraceOpen(o => !o)}>
              <span className={`trace-arrow ${traceOpen ? 'open' : ''}`}>▸</span>
              Agent 推理轨迹（{thoughts.length} 次工具调用）
              {running && <span className="trace-running"><i></i><i></i><i></i></span>}
            </div>
            {traceOpen && (
              <div className="trace-body">
                {thoughts.map((t, i) => (
                  <div key={i} className="trace-row">
                    <span className={`trace-dot trace-dot-${t.status}`}>{t.status === 'success' ? '✓' : t.status === 'failed' ? '✕' : ''}</span>
                    <span className="trace-tool">{t.tool_name}</span>
                    <span className={`trace-status trace-status-${t.status}`}>{t.status === 'running' ? '调用中' : t.status === 'success' ? '完成' : '失败'}</span>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}
        {(msg.text || msg.status === 'streaming') && (
          <div className="cmsg-bubble">
            {msg.text}
            {msg.status === 'streaming' && <span className="typing"><i></i><i></i><i></i></span>}
          </div>
        )}
        {(msg.cards || []).map(c => <CardContainer key={c.card_id} card={c} onAsk={onAsk} />)}
      </div>
    </div>
  );
}
