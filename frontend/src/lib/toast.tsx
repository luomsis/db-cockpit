/* ================= 全局 Toast（右下角滑入，2.6s 自动消失） =================
 * 用法：toast.success('已创建') / toast.error('失败：...') / toast.info('...')
 * 在 App 根部挂 <ToastHost /> 渲染。
 */
import { useEffect, useState } from 'react';

export interface ToastMsg { id: number; kind: 'success' | 'error' | 'info'; text: string }
type Listener = (t: ToastMsg) => void;

const listeners = new Set<Listener>();
let seq = 0;

function emit(kind: ToastMsg['kind'], text: string) {
  const msg = { id: ++seq, kind, text };
  listeners.forEach(l => l(msg));
}

export const toast = {
  success: (text: string) => emit('success', text),
  error: (text: string) => emit('error', text),
  info: (text: string) => emit('info', text),
};

export function ToastHost() {
  const [items, setItems] = useState<ToastMsg[]>([]);
  useEffect(() => {
    const on = (t: ToastMsg) => {
      setItems(cur => [...cur, t]);
      setTimeout(() => setItems(cur => cur.filter(x => x.id !== t.id)), 2600);
    };
    listeners.add(on);
    return () => { listeners.delete(on); };
  }, []);
  if (!items.length) return null;
  return (
    <div className="toast-host">
      {items.map(t => <div key={t.id} className={`toast ${t.kind}`}>{t.text}</div>)}
    </div>
  );
}
