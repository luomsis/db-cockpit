/* ================= 用户输入锚点刻度簇（对话页与悬浮抽屉共用） =================
 * 交互对齐已确认的 demo v4：
 * - 刻度簇固定在消息可视区左缘纵向居中（不随滚动消失），间隔 8px，数量随用户输入增长
 * - 滚动联动：以视口中线为界判断正在浏览的提问，对应刻度常亮
 * - 悬停单枚刻度：仅显示该条提问（#序号 + 文本，浮层在刻度右侧，CSS hover 显示）
 * - 点击刻度：平滑滚动到对应用户消息（居中）并高亮 2s
 * 用法：用本组件包裹消息滚动容器，并传入该容器的 ref 与消息列表。
 */
import { useCallback, useEffect, useState } from 'react';
import type { ReactNode, RefObject } from 'react';
import type { ChatMessage } from '../lib/types';

export function ChatAnchorRail({ scrollRef, messages, children }: {
  scrollRef: RefObject<HTMLDivElement | null>;
  messages: ChatMessage[];
  children: ReactNode;
}) {
  const users = messages.filter(m => m.role === 'user');
  const [activeIdx, setActiveIdx] = useState(0);

  /* 滚动联动：视口中线落在哪条提问区域，就高亮哪条刻度 */
  const syncActive = useCallback(() => {
    const el = scrollRef.current;
    if (!el || !users.length) return;
    const mid = el.scrollTop + el.clientHeight / 2;
    const nodes = el.querySelectorAll<HTMLElement>('[data-role="user"]');
    let idx = 0;
    nodes.forEach((n, i) => { if (n.offsetTop <= mid) idx = i; });
    setActiveIdx(prev => (prev === idx ? prev : idx));
  }, [scrollRef, users.length]);

  useEffect(() => {
    const el = scrollRef.current;
    if (!el) return;
    const onScroll = () => requestAnimationFrame(syncActive);
    el.addEventListener('scroll', onScroll, { passive: true });
    syncActive();
    return () => el.removeEventListener('scroll', onScroll);
  }, [scrollRef, syncActive, messages]);

  /* 点击刻度：定位 + 高亮（滚动结束后联动高亮自动跟随） */
  const locate = (i: number) => {
    const el = scrollRef.current?.querySelectorAll<HTMLElement>('[data-role="user"]')[i];
    if (!el) return;
    el.scrollIntoView({ behavior: 'smooth', block: 'center' });
    el.classList.remove('anchor-hit');
    void el.offsetWidth;
    el.classList.add('anchor-hit');
    setTimeout(() => el.classList.remove('anchor-hit'), 2100);
  };

  return (
    <div className="chat-rail-wrap">
      {children}
      {users.length > 0 && (
        <div className="anchor-cluster">
          {users.map((m, i) => (
            <div key={i} className={`anchor-tick${i === activeIdx ? ' active' : ''}`} data-idx={i} onClick={() => locate(i)}>
              <span className="tick-pop"><i>#{i + 1}</i>{m.text}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
