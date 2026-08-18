/* ================= 共享聊天逻辑（完整对话页 + 悬浮面板共用） =================
 * 数据源：Go apiserver（会话 CRUD + SSE 六事件流）；
 * apiserver 不可达时整链路回退本地 mock（runMockTurn），演示不依赖后端存活。
 */
import { useEffect, useRef, useState } from 'react';
import type { AgentEvent, ChatMessage, ChatSession } from './types';
import { apiGet, apiPost, apiDelete } from './api';
import { runMockTurn } from './mockAgent';
import { loadSessions, persist, newSession, uid } from './chatSessions';

/* 一次性迁移：localStorage 会话导入服务端（仅在服务端为空且本地有数据时） */
let migrationDone = false;
async function ensureMigration(): Promise<boolean> {
  if (migrationDone) return true;
  migrationDone = true;
  const local = loadSessions();
  try {
    const server = await apiGet<ChatSession[]>('/api/chat/sessions');
    if (!server.length && local.length) {
      await apiPost('/api/chat/sessions/import', { sessions: local }).catch(() => { /* 迁移失败保持本地 */ });
    }
    return true;
  } catch (e) {
    return false;
  }
}

export function useChat() {
  const [sessions, setSessions] = useState<ChatSession[]>(() => {
    const s = loadSessions();
    return s.length ? s : [newSession()];
  });
  const [activeId, setActiveId] = useState<string>(() => loadSessions()[0]?.id ?? '');
  const [streaming, setStreaming] = useState(false);
  const offlineRef = useRef(false);
  const seqRef = useRef<Record<string, number>>({});
  const esRef = useRef<EventSource | null>(null);
  const mockRef = useRef<{ cancel: () => void } | null>(null);

  /* 启动：迁移 + 拉取服务端会话（失败进入离线 mock 模式） */
  useEffect(() => {
    (async () => {
      const ok = await ensureMigration();
      if (!ok) { offlineRef.current = true; return; }
      try {
        const list = await apiGet<ChatSession[]>('/api/chat/sessions');
        const mapped = (list || []).map(s => ({ ...s, messages: s.messages || [] }));
        if (mapped.length) {
          setSessions(mapped);
          setActiveId(cur => (mapped.some(s => s.id === cur) ? cur : mapped[0].id));
        }
      } catch (e) {
        offlineRef.current = true;
      }
    })();
  }, []);

  /* localStorage 镜像（离线时数据连续） */
  useEffect(() => { persist(sessions); }, [sessions]);

  const active = sessions.find(s => s.id === activeId) || sessions[0];
  useEffect(() => { if (active && active.id !== activeId) setActiveId(active.id); }, [active, activeId]);

  const updateSession = (id: string, fn: (s: ChatSession) => ChatSession) => {
    setSessions(ss => ss.map(s => (s.id === id ? { ...fn(s), updatedAt: Date.now() } : s)));
  };
  const patchMessage = (sid: string, mid: string, fn: (m: ChatMessage) => ChatMessage) => {
    updateSession(sid, s => ({ ...s, messages: s.messages.map(m => (m.id === mid ? fn(m) : m)) }));
  };

  const finish = () => {
    esRef.current?.close();
    esRef.current = null;
    mockRef.current = null;
    setStreaming(false);
  };

  const applyEvent = (sid: string, mid: string, ev: AgentEvent) => {
    if (ev.type === 'token') {
      patchMessage(sid, mid, m => ({ ...m, text: m.text + ev.text_delta }));
    } else if (ev.type === 'thought') {
      patchMessage(sid, mid, m => {
        const idx = m.thoughts.findIndex(t => t.tool_name === ev.tool_name);
        const thoughts = idx >= 0
          ? m.thoughts.map((t, i) => (i === idx ? { ...t, status: ev.status } : t))
          : [...m.thoughts, { tool_name: ev.tool_name, status: ev.status }];
        return { ...m, thoughts };
      });
    } else if (ev.type === 'card') {
      patchMessage(sid, mid, m => {
        const idx = m.cards.findIndex(c => c.card_id === ev.card.card_id);
        const cards = idx >= 0
          ? m.cards.map((c, i) => (i === idx ? ev.card : c))
          : [...m.cards, ev.card];
        return { ...m, cards };
      });
    } else if (ev.type === 'progress') {
      patchMessage(sid, mid, m => ({
        ...m,
        cards: m.cards.map(c => {
          if (c.card_type === 'task_progress' && c.payload?.task_id === ev.task_id) {
            const done = ev.progress >= 100;
            const activeStep = ev.progress < 40 ? 0 : ev.progress < 75 ? 1 : 2;
            const stages = (c.payload.stages || []).map((s: any, i: number) => ({
              ...s,
              status: done || i < activeStep ? 'done' : i === activeStep ? 'running' : 'pending',
            })) as any[];
            return { ...c, payload: { ...c.payload, progress: ev.progress, stage: ev.stage, status: done ? 'done' : 'running', stages } };
          }
          return c;
        }),
      }));
    } else if (ev.type === 'done') {
      patchMessage(sid, mid, m => ({ ...m, status: 'final' }));
      finish();
    } else if (ev.type === 'error') {
      patchMessage(sid, mid, m => ({ ...m, status: 'error', text: m.text + `\n\n[错误] ${ev.message}` }));
      finish();
    }
  };

  const send = (text?: string): string | undefined => {
    const q = (text ?? '').trim();
    if (!q || streaming || !active) return q;
    const sid = active.id;
    const botId = uid();
    updateSession(sid, s => ({
      ...s,
      title: s.messages.length === 0 ? q.slice(0, 24) : s.title,
      messages: [
        ...s.messages,
        { id: uid(), role: 'user', text: q, thoughts: [], cards: [], status: 'final' },
        { id: botId, role: 'assistant', text: '', thoughts: [], cards: [], status: 'streaming' },
      ],
    }));
    setStreaming(true);

    if (offlineRef.current) {
      mockRef.current = runMockTurn(q, ev => applyEvent(sid, botId, ev));
      return q;
    }
    (async () => {
      try {
        await apiPost(`/api/chat/sessions/${sid}/turns`, { text: q });
        const last = seqRef.current[sid] ?? 0;
        const es = new EventSource(`/api/chat/sessions/${sid}/stream?last_event_id=${last}`);
        esRef.current = es;
        es.onmessage = (e: MessageEvent) => {
          const seq = Number(e.lastEventId);
          if (Number.isFinite(seq) && seq > 0) seqRef.current[sid] = seq;
          let ev: AgentEvent;
          try { ev = JSON.parse(e.data); } catch (err) { return; }
          applyEvent(sid, botId, ev);
        };
        // onerror：EventSource 自动重连（带 Last-Event-ID），由服务端重放补齐
      } catch (e) {
        // POST 失败 → 本地 mock 兜底
        mockRef.current = runMockTurn(q, ev => applyEvent(sid, botId, ev));
      }
    })();
    return q;
  };

  const stop = () => {
    if (active) {
      apiPost(`/api/chat/sessions/${active.id}/turns/current/cancel`).catch(() => { /* 离线忽略 */ });
    }
    mockRef.current?.cancel();
    finish();
    setSessions(ss => ss.map(s => ({
      ...s,
      messages: s.messages.map(m => (m.status === 'streaming' ? { ...m, status: 'final' } : m)),
    })));
  };

  const createSession = async () => {
    if (!offlineRef.current) {
      try {
        const s = await apiPost<ChatSession & { messages?: ChatMessage[] }>('/api/chat/sessions', {});
        const created: ChatSession = { ...s, messages: s.messages || [] };
        setSessions(ss => [created, ...ss]);
        setActiveId(created.id);
        return;
      } catch (e) { /* 落回本地 */ }
    }
    const s = newSession();
    setSessions(ss => [s, ...ss]);
    setActiveId(s.id);
  };

  const removeSession = (id: string) => {
    if (!offlineRef.current) apiDelete(`/api/chat/sessions/${id}`).catch(() => { /* 离线忽略 */ });
    setSessions(ss => {
      const next = ss.filter(x => x.id !== id);
      const list = next.length ? next : [newSession()];
      if (id === activeId) setActiveId(list[0].id);
      return list;
    });
  };

  return { sessions, active, activeId, setActiveId, streaming, send, stop, createSession, removeSession };
}
