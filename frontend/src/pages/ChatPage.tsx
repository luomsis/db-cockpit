import { useEffect, useRef, useState } from 'react';
import { MessageView } from '../components/chatParts';
import { QUICK_QUESTIONS } from '../lib/mockAgent';
import { useChat } from '../lib/useChat';
import { IconRobot } from '../components/icons';
import { useBreadcrumb } from '../App';

/* 完整对话页：会话与流式回复走 apiserver SSE（离线自动回退本地 mock） */
export default function ChatPage() {
  useBreadcrumb([{ label: '首页' }, { label: '智能对话' }]);
  const { sessions, active, setActiveId, streaming, send, stop, createSession, removeSession } = useChat();
  const [input, setInput] = useState('');
  const bodyRef = useRef<HTMLDivElement>(null);

  /* 自动滚到底部 */
  useEffect(() => {
    const el = bodyRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [active?.messages]);

  const doSend = (text?: string) => {
    const q = send(text);
    if (q !== undefined) setInput('');
  };

  /* 告警页「问 AI」入口：sessionStorage 预填草稿，进入即自动发送 */
  useEffect(() => {
    const draft = sessionStorage.getItem('alertAskDraft');
    if (draft) {
      sessionStorage.removeItem('alertAskDraft');
      const timer = setTimeout(() => doSend(draft), 300);
      return () => clearTimeout(timer);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  if (!active) return null;
  const empty = active.messages.length === 0;

  return (
    <div className="chatp">
      {/* 会话列表 */}
      <aside className="chatp-sessions">
        <button className="btn sm primary chatp-new" onClick={createSession}>＋ 新会话</button>
        <div className="chatp-sess-list">
          {sessions.map(s => (
            <div key={s.id} className={`chatp-sess ${s.id === active.id ? 'active' : ''}`} onClick={() => setActiveId(s.id)}>
              <div className="chatp-sess-title">{s.title}</div>
              <div className="chatp-sess-meta">{s.messages.length} 条消息</div>
              <button className="chatp-sess-del" title="删除会话"
                onClick={e => { e.stopPropagation(); removeSession(s.id); }}>✕</button>
            </div>
          ))}
        </div>
        <div className="chatp-foot-note">已接入 apiserver（SSE 流式）<br />后端不可用时自动回退本地演示</div>
      </aside>

      {/* 对话主区 */}
      <div className="chatp-main card">
        <div className="chatp-head">
          <div className="chat-head-l">
            <div className="chat-avatar">AI</div>
            <div><b>DB Cockpit 助手</b><span className="chat-status"><i></i>在线 · 路由 / 诊断 / 问数专家</span></div>
          </div>
        </div>

        <div className="chatp-body" ref={bodyRef}>
          {empty && (
            <div className="chatp-welcome">
              <div className="chatp-welcome-ico"><IconRobot /></div>
              <div className="chatp-welcome-title">你好，我是 DB Cockpit 智能运维助手</div>
              <div className="chatp-welcome-desc">支持告警问数、指标问数、实例诊断（含异步深度扫描）。点击下方问题快速开始：</div>
              <div className="chatp-quick">
                {QUICK_QUESTIONS.map(q => <button key={q} onClick={() => doSend(q)}>{q}</button>)}
              </div>
            </div>
          )}
          {active.messages.map(m => <MessageView key={m.id} msg={m} onAsk={q => doSend(q)} />)}
          {streaming && (
            <div className="chatp-stop-row">
              <button className="btn sm" onClick={stop}>■ 停止生成</button>
            </div>
          )}
        </div>

        <div className="chatp-input">
          <div className="chatp-quick-inline">
            {!empty && QUICK_QUESTIONS.slice(0, 2).map(q => <button key={q} onClick={() => doSend(q)}>{q}</button>)}
          </div>
          <div className="chatp-input-row">
            <input
              value={input}
              placeholder={streaming ? 'AI 正在回复…' : '向 AI 助手提问，如：诊断 mysql-prod-order-01…'}
              onChange={e => setInput(e.target.value)}
              onKeyDown={e => { if (e.key === 'Enter') doSend(); }}
              disabled={streaming}
            />
            <button onClick={() => doSend()} disabled={streaming || !input.trim()}>发送</button>
          </div>
        </div>
      </div>
    </div>
  );
}
