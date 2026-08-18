import { useEffect, useRef } from 'react';

/* 模态浮层：点击遮罩关闭，Esc 关闭 */
export function Overlay({ onClose, children, width }: { onClose: () => void; children: React.ReactNode; width?: string }) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose(); };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [onClose]);
  return (
    <div className="dash-popover-overlay" onMouseDown={e => { if (e.target === e.currentTarget) onClose(); }}>
      <div className="dash-dialog" style={width ? { width } : undefined}>{children}</div>
    </div>
  );
}

/* 信息弹窗（新建/编辑大盘信息） */
export function InfoDialog({ title, okText, initial, onOk, onClose }: {
  title: string; okText?: string;
  initial?: { title: string; description: string };
  onOk: (title: string, description: string) => void;
  onClose: () => void;
}) {
  const tRef = useRef<HTMLInputElement>(null);
  const dRef = useRef<HTMLTextAreaElement>(null);
  useEffect(() => { tRef.current?.focus(); tRef.current?.select(); }, []);
  const submit = () => {
    const title = (tRef.current?.value || '').trim();
    if (!title) { tRef.current?.focus(); return; }
    onOk(title, (dRef.current?.value || '').trim());
  };
  return (
    <Overlay onClose={onClose}>
      <div className="dap-head">{title}</div>
      <div className="dap-body">
        <label>标题
          <input ref={tRef} defaultValue={initial?.title || ''} placeholder="如：交易核心大盘"
            onKeyDown={e => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); submit(); } }} />
        </label>
        <label>描述
          <textarea ref={dRef} rows={3} placeholder="可选，简述该大盘的用途" defaultValue={initial?.description || ''} />
        </label>
      </div>
      <div className="dap-foot">
        <button className="btn sm" onClick={onClose}>取消</button>
        <button className="btn sm primary" onClick={submit}>{okText || '确定'}</button>
      </div>
    </Overlay>
  );
}

/* 确认弹窗 */
export function ConfirmDialog({ title, message, okText, danger, onOk, onClose }: {
  title: string; message: string; okText?: string; danger?: boolean;
  onOk: () => void; onClose: () => void;
}) {
  return (
    <Overlay onClose={onClose}>
      <div className="dap-head">{title}</div>
      <div className="dap-body"><div className="dash-confirm-msg">{message}</div></div>
      <div className="dap-foot">
        <button className="btn sm" onClick={onClose}>取消</button>
        <button className={`btn sm ${danger ? 'danger' : 'primary'}`} onClick={() => { onClose(); onOk(); }}>{okText || '确认'}</button>
      </div>
    </Overlay>
  );
}

/* 弹出菜单（卡片 ⋯ 菜单等） */
export function MenuPopover({ items, onClose, style }: {
  items: { label: string; onClick: () => void }[];
  onClose: () => void;
  style?: React.CSSProperties;
}) {
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const onDown = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) onClose();
    };
    document.addEventListener('mousedown', onDown);
    return () => document.removeEventListener('mousedown', onDown);
  }, [onClose]);
  return (
    <div className="dash-card-popover" ref={ref} style={style}>
      {items.map(it => (
        <button key={it.label} className="exp-more-item" onClick={e => { e.stopPropagation(); onClose(); it.onClick(); }}>{it.label}</button>
      ))}
    </div>
  );
}
