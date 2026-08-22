/* ================= ZCode 风格聊天输入框（主聊天页 + 抽屉共用） =================
 * 结构对标 ZCode 客户端 composer：一张圆角卡片 = 多行自动增高的输入区
 * + 底部工具栏（左：模型选择等；右：发送按钮，流式生成时同一位置变停止，Esc 亦可停止）。
 * Enter 发送 · Shift+Enter 换行 · 输入法组合期间 Enter 不发送。
 */
import { useEffect, useRef } from 'react';
import { IconArrowUp, IconStop } from './icons';

interface Props {
  value: string;
  onChange: (v: string) => void;
  onSend: (text: string) => void;
  streaming: boolean;
  onStop: () => void;
  placeholder?: string;
  modelSlot?: React.ReactNode;   // 底部工具栏左侧内容（ModelPicker 等）
}

export function ChatComposer({ value, onChange, onSend, streaming, onStop, placeholder, modelSlot }: Props) {
  const taRef = useRef<HTMLTextAreaElement>(null);

  /* 自动增高：随内容撑高，超过 160px 转内部滚动 */
  useEffect(() => {
    const el = taRef.current;
    if (!el) return;
    el.style.height = 'auto';
    el.style.height = `${Math.min(el.scrollHeight, 160)}px`;
  }, [value]);

  /* 流式生成中 Esc 停止（与 ZCode 一致） */
  useEffect(() => {
    if (!streaming) return;
    const onKey = (e: KeyboardEvent) => { if (e.key === 'Escape') onStop(); };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [streaming, onStop]);

  const submit = () => {
    if (streaming) return;             // 生成中拦截发送，草稿留在框内不丢字
    const q = value.trim();
    if (q) onSend(q);
  };

  const onKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key !== 'Enter' || e.shiftKey) return;
    if (e.nativeEvent.isComposing || e.keyCode === 229) return;
    e.preventDefault();
    submit();
  };

  return (
    <div className="chat-composer">
      <textarea ref={taRef} rows={1} value={value} placeholder={placeholder}
        onChange={e => onChange(e.target.value)} onKeyDown={onKeyDown} />
      <div className="chat-composer-bar">
        <div className="chat-composer-left">{modelSlot}</div>
        {streaming ? (
          <button type="button" className="chat-composer-stop" onClick={onStop}
            title="停止生成（Esc）" aria-label="停止生成">
            <IconStop size={13} />
          </button>
        ) : (
          <button type="button" className="chat-composer-send" onClick={submit}
            disabled={!value.trim()} title="发送（Enter · Shift+Enter 换行）" aria-label="发送">
            <IconArrowUp size={16} />
          </button>
        )}
      </div>
    </div>
  );
}
