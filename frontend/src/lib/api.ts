/* ================= REST 客户端：统一包裹 {code, message, data} 解包 =================
 * - 成功 code=0 返回 data；失败抛错（HTTP 状态码保留语义）
 * - withFallback：apiserver 不可达时回退本地 mock（演示不依赖后端存活）
 */
export interface Envelope<T> { code: number; message: string; data: T }

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const r = await fetch(path, init);
  let body: Envelope<T> | null = null;
  try { body = await r.json(); } catch (e) { /* non-json */ }
  if (!r.ok || !body || body.code !== 0) {
    throw new Error(body?.message || `HTTP ${r.status}`);
  }
  return body.data;
}

const jsonInit = (method: string, payload?: unknown): RequestInit => ({
  method,
  headers: { 'Content-Type': 'application/json' },
  body: payload === undefined ? undefined : JSON.stringify(payload),
});

export const apiGet = <T>(path: string) => request<T>(path);
export const apiPost = <T>(path: string, payload?: unknown) => request<T>(path, jsonInit('POST', payload));
export const apiPut = <T>(path: string, payload?: unknown) => request<T>(path, jsonInit('PUT', payload));
export const apiDelete = <T>(path: string) => request<T>(path, jsonInit('DELETE'));

/* mock 兜底：请求失败时返回 fallback()，演示环境后端可停机 */
export async function withFallback<T>(p: Promise<T>, fallback: () => T): Promise<T> {
  try { return await p; } catch (e) { return fallback(); }
}
