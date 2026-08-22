/* ================= 聊天输入框内左侧 LLM 模型选择胶囊 =================
 * 纯前端展示：选择仅记录在当前会话（useChat 内存态），不传后端。
 * 候选 = 已启用的模型配置（useChat 拉取并过滤），默认选中第一个。
 */
import { useEffect, useRef, useState } from 'react';
import { Link } from 'react-router-dom';
import type { ModelConfig } from '../lib/types';
import { IconChevronDown, IconCheck } from './icons';

interface Props {
  configs: ModelConfig[];      // 已启用的模型配置（按创建顺序）
  value: string | null;        // 当前会话选中的配置 id
  onChange: (id: string) => void;
  onNavigate?: () => void;     // 抽屉场景：跳设置页前先收起面板
}

/* 显示名回退链：name → model → provider → id */
const label = (c: ModelConfig) => c.name || c.model || c.provider || c.id;

/* 交互元素用 div + role=button 而非 <button>：与原生 button 默认样式解耦，便于保持幽灵胶囊样式 */
export function ModelPicker({ configs, value, onChange, onNavigate }: Props) {
  const [open, setOpen] = useState(false);
  const wrapRef = useRef<HTMLDivElement>(null);

  const current = configs.find(c => c.id === value) ?? null;

  /* 点击外部 / Escape 关闭 */
  useEffect(() => {
    if (!open) return;
    const onDown = (e: MouseEvent) => {
      if (wrapRef.current && !wrapRef.current.contains(e.target as Node)) setOpen(false);
    };
    const onKey = (e: KeyboardEvent) => { if (e.key === 'Escape') setOpen(false); };
    document.addEventListener('mousedown', onDown);
    document.addEventListener('keydown', onKey);
    return () => {
      document.removeEventListener('mousedown', onDown);
      document.removeEventListener('keydown', onKey);
    };
  }, [open]);

  const pick = (id: string) => { onChange(id); setOpen(false); };

  return (
    <div className="chat-model-wrap" ref={wrapRef}>
      <div className={`chat-model-chip${open ? ' open' : ''}`} role="button" tabIndex={0} title="选择 LLM 模型"
        onClick={() => setOpen(v => !v)}
        onKeyDown={e => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); setOpen(v => !v); } }}>
        <span className="chat-model-chip-name">{current ? label(current) : '未配置模型'}</span>
        <IconChevronDown size={12} />
      </div>
      {open && (
        <div className="chat-model-pop">
          {configs.length === 0 ? (
            <div className="chat-model-empty">
              <div>暂无启用的 LLM 模型</div>
              <Link to="/settings" onClick={onNavigate}>去『模型设置』页添加</Link>
            </div>
          ) : (
            configs.map(c => (
              <div key={c.id} role="button" tabIndex={0}
                className={`chat-model-item${c.id === value ? ' active' : ''}`}
                onClick={() => pick(c.id)}
                onKeyDown={e => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); pick(c.id); } }}>
                <span className="chat-model-item-main">{label(c)}</span>
                {c.id === value && <IconCheck size={14} />}
                <span className="chat-model-item-sub">{[c.provider, c.model].filter(Boolean).join(' · ') || '—'}</span>
              </div>
            ))
          )}
        </div>
      )}
    </div>
  );
}
