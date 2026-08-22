/* ================= 聊天会话本地存储（完整对话页与悬浮球面板共用） =================
 * 同一份 localStorage 数据：面板里的会话在完整对话页可见，反之亦然。
 */
import type { ChatSession } from './types';

const STORE_KEY = 'dbCopilotChatSessions';
const WIDTH_KEY = 'dbChatDrawerWidth';
const ACTIVE_KEY = 'dbChatActiveSession';

export const uid = () => `${Date.now().toString(36)}${Math.random().toString(36).slice(2, 6)}`;

export function newSession(): ChatSession {
  const now = Date.now();
  return { id: uid(), title: '新会话', createdAt: now, updatedAt: now, messages: [], draft: true };
}

export function loadSessions(): ChatSession[] {
  try {
    const arr = JSON.parse(localStorage.getItem(STORE_KEY) || 'null');
    if (Array.isArray(arr)) return arr;
  } catch (e) { /* ignore */ }
  return [];
}

export function persist(sessions: ChatSession[]) {
  localStorage.setItem(STORE_KEY, JSON.stringify(sessions));
}

/* 面板宽度记忆：合法区间外回退默认值 */
export function loadDrawerWidth(): number {
  const w = Number(localStorage.getItem(WIDTH_KEY));
  return Number.isFinite(w) && w >= 320 && w <= 920 ? w : 430;
}

export function saveDrawerWidth(w: number) {
  localStorage.setItem(WIDTH_KEY, String(Math.round(w)));
}

/* 面板上次浏览的会话：重开时恢复 */
export function loadActiveSession(fallback: string): string {
  const id = localStorage.getItem(ACTIVE_KEY);
  return id || fallback;
}
export function saveActiveSession(id: string) {
  localStorage.setItem(ACTIVE_KEY, id);
}
