/* ================= 右侧停靠聊天面板（悬浮球唤起，不跳转页面） =================
 * 会话与完整对话页共用（服务端存储）；支持左缘拖拽调整宽度（记忆宽度）。
 */
import { useEffect, useRef, useState } from 'react';
import type { PointerEvent as RPointerEvent } from 'react';
import { QUICK_QUESTIONS } from '../lib/mockAgent';
import { useChat } from '../lib/useChat';
import { loadDrawerWidth, saveDrawerWidth, loadActiveSession, saveActiveSession } from '../lib/chatSessions';
import { relTime } from '../lib/dashboards';
import { MessageView } from './chatParts';
import { ChatAnchorRail } from './chatAnchorRail';
import { IconHistory, IconPlus, IconClose, IconRobot } from './icons';

export function ChatPanel({ onClose }: { onClose: () => void }) {
  const { sessions, active, setActiveId, streaming, send, stop, createSession, removeSession } = useChat();
  const [input, setInput] = useState('');
  const [showHistory, setShowHistory] = useState(false);
  const [width, setWidth] = useState<number>(() => loadDrawerWidth());
  const bodyRef = useRef<HTMLDivElement>(null);

  useEffect(() => { if (active) saveActiveSession(active.id); }, [active?.id]);

  useEffect(() => {
    const el = bodyRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [active?.messages]);

  const doSend = (text?: string) => {
    const q = send(text);
    if (q !== undefined) setInput('');
  };

  /* 左缘拖拽调宽（320px ~ 视口内上限），松手后记忆宽度 */
  const onResizeStart = (e: RPointerEvent<HTMLDivElement>) => {
    e.preventDefault();
    const startX = e.clientX;
    const startW = width;
    const move = (ev: PointerEvent) => {
      const max = Math.min(920, window.innerWidth - 200);
      setWidth(Math.max(320, Math.min(max, startW + (startX - ev.clientX))));
    };
    const up = () => {
      document.removeEventListener('pointermove', move);
      document.removeEventListener('pointerup', up);
      document.body.style.userSelect = '';
      setWidth(w => { saveDrawerWidth(w); return w; });
    };
    document.body.style.userSelect = 'none';
    document.addEventListener('pointermove', move);
    document.addEventListener('pointerup', up);
  };

  if (!active) return null;
  const empty = active.messages.length === 0;
  const sortedSessions = sessions.slice().sort((a, b) => b.updatedAt - a.updatedAt);

  return (
    <div className="chat-drawer" style={{ width }}>
      <div className="chat-drawer-resizer" onPointerDown={onResizeStart} title="拖拽调整宽度"></div>
      <div className="chat-drawer-head">
        <div className="chat-drawer-title">
          <span className="chat-drawer-ava">AI</span>
          <div>
            <b>DB Cockpit 智能助手</b>
            <span className="chat-drawer-status"><i></i>在线 · 路由 / 诊断 / 问数</span>
          </div>
        </div>
        <div className="chat-drawer-ops">
          <button className={`chat-drawer-ic${showHistory ? ' on' : ''}`} title="聊天历史" onClick={() => setShowHistory(v => !v)}><IconHistory /></button>
          <button className="chat-drawer-ic" title="新会话" onClick={createSession}><IconPlus /></button>
          <button className="chat-drawer-close" onClick={onClose} title="收起面板"><IconClose /></button>
        </div>
      </div>

      {showHistory ? (
        <div className="chat-history">
          <div className="chat-history-head">
            <span>聊天历史 · {sessions.length} 个会话</span>
          </div>
          <div className="chat-history-list">
            {sortedSessions.map(s => (
              <div key={s.id} className={`chat-history-item${s.id === active.id ? ' active' : ''}`}
                onClick={() => { setActiveId(s.id); setShowHistory(false); }}>
                <div className="chi-title">{s.title}</div>
                <div className="chi-meta">{s.messages.length} 条消息 · {relTime(s.updatedAt)}</div>
                <button className="chi-del" title="删除会话"
                  onClick={e => { e.stopPropagation(); removeSession(s.id); }}>✕</button>
              </div>
            ))}
          </div>
        </div>
      ) : (
        <>
          <ChatAnchorRail scrollRef={bodyRef} messages={active.messages}>
            <div className="chat-drawer-body" ref={bodyRef}>
              {empty && (
                <div className="chatp-welcome">
                  <div className="chatp-welcome-ico"><IconRobot /></div>
                  <div className="chatp-welcome-title">你好，我是 DB Cockpit 智能运维助手</div>
                  <div className="chatp-welcome-desc">支持告警问数、指标问数、实例诊断。点击下方问题快速开始：</div>
                  <div className="chatp-quick">
                    {QUICK_QUESTIONS.slice(0, 4).map(q => <button key={q} onClick={() => doSend(q)}>{q}</button>)}
                  </div>
                </div>
              )}
              {active.messages.map(m => <MessageView key={m.id} msg={m} onAsk={q => doSend(q)} />)}
              {streaming && (
                <div className="chatp-stop-row"><button className="btn sm" onClick={stop}>■ 停止生成</button></div>
              )}
            </div>
          </ChatAnchorRail>
          <div className="chat-drawer-input">
            <input
              value={input}
              placeholder={streaming ? 'AI 正在回复…' : '向 AI 助手提问…'}
              onChange={e => setInput(e.target.value)}
              onKeyDown={e => { if (e.key === 'Enter') doSend(); }}
            />
            <button className="chat-drawer-send" onClick={() => doSend()} disabled={streaming || !input.trim()}>发送</button>
          </div>
        </>
      )}
    </div>
  );
}
