/* ================= 统一查询数据层（mock / api 双 Provider 并存） =================
 * 查询协议对齐后端接口预期（架构文档 §7）：
 *   fetchSeries(query) -> Promise<{ series }>
 *   fetchAnnotations(range) -> Promise<Annotation[]>
 * 后端就绪后 setQueryProvider(ApiProvider) 即可，渲染层零改动。
 */
import type { Annotation, QueryRequest, ResolvedTarget } from './types';
import { CLUSTERS, METRIC_LIB } from './mockData';

const API_BASE = '/api/dash';

export const HOURS = Array.from({ length: 24 }, (_, i) => `${String(i).padStart(2, '0')}:00`);

export function rangeTicks(range: string): string[] {
  if (range === '1h') return Array.from({ length: 12 }, (_, i) => `${i * 5}m`);
  if (range === '6h') return Array.from({ length: 12 }, (_, i) => `${i * 30}m`);
  if (range === '7d') return ['周一', '周二', '周三', '周四', '周五', '周六', '周日'];
  return HOURS;
}
export function rangeSpike(range: string): number { return range === '7d' ? 3 : 14; }

/* 可复现伪随机：固定种子 → 时间偏移对比的"昨日"数据保持稳定 */
export function seededRandom(seed: number) {
  let a = seed >>> 0;
  return function () {
    a |= 0; a = (a + 0x6D2B79F5) | 0;
    let t = Math.imul(a ^ (a >>> 15), 1 | a);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

export function genSeries(base: number, jitter: number, spikeAt?: number, seed?: number): number[] {
  const rand = seed === undefined ? Math.random : seededRandom(seed);
  const arr: number[] = [];
  for (let i = 0; i < 24; i++) {
    let v = base + Math.sin(i / 3.2) * jitter + (rand() - 0.5) * jitter;
    if (spikeAt && Math.abs(i - spikeAt) <= 1) v += jitter * 2.6;
    arr.push(Math.max(1, Math.round(v)));
  }
  return arr;
}

function applyAgg(data: number[], agg?: string): number[] {
  if (!agg || agg === 'avg') return data;
  const arr = data.slice();
  if (agg === 'max') return arr.map(v => Math.round(v * 1.35 + 5));
  if (agg === 'min') return arr.map(v => Math.max(1, Math.round(v * 0.65 - 3)));
  if (agg === 'last') { const lv = arr[arr.length - 1] || 1; return arr.map(() => lv); }
  if (agg === 'p95') {
    const sorted = data.slice().sort((a, b) => a - b);
    const p95v = sorted[Math.floor(sorted.length * 0.95)] || sorted[sorted.length - 1] || 1;
    return arr.map(v => Math.round((v + p95v) / 2));
  }
  return data;
}

function resolveGroups(q: QueryRequest): { name: string; seedBase: number }[] {
  const clusters = CLUSTERS.filter(c => !q.dbType || c.type === q.dbType);
  if (q.groupBy === 'cluster') return clusters.map((c, i) => ({ name: c.name, seedBase: i + 1 }));
  if (q.groupBy === 'instance') {
    return clusters.flatMap((c, ci) => c.instances.map((inst, ii) => ({
      name: inst.name, seedBase: ci * 100 + ii + 1,
    }))).slice(0, 12);
  }
  return [{ name: '', seedBase: 0 }];
}

const GROUP_PALETTE = ['#006aff', '#ff9500', '#00b365', '#7a5af8', '#f53f3f', '#00a3e0', '#d46b08', '#0091c9', '#6a3fd4', '#00897b', '#d4380d', '#8a97ad'];

/* ---------------- Mock Provider ---------------- */
export const MockProvider = {
  fetchSeries(q: QueryRequest): Promise<{ series: ResolvedTarget[] }> {
    const m = METRIC_LIB.find(x => x.id === q.metric) || { id: q.metric, name: q.metric, unit: '', base: 50, jitter: 10 };
    const xs = rangeTicks(q.range);
    const spike = rangeSpike(q.range);
    const shiftSeed = q.shift ? 20260809 : 20260811;
    const groups = resolveGroups(q);
    const useGroup = !!q.groupBy && q.groupBy !== 'none' && groups.length > 1;
    const series = groups.map((g, gi) => {
      const seed = (shiftSeed + g.seedBase * 7919) >>> 0;
      let data = genSeries(m.base, m.jitter, spike, seed).slice(0, xs.length);
      data = applyAgg(data, q.agg);
      return {
        name: useGroup ? (q.name ? `${q.name} · ${g.name}` : g.name) : (q.name || m.name),
        data,
        color: useGroup ? GROUP_PALETTE[gi % GROUP_PALETTE.length] : (q.color || ''),
        unit: q.unit || m.unit,
        axis: (q.axis === 'right' ? 'right' : 'left') as 'left' | 'right',
        type: q.type || 'line',
      };
    });
    return new Promise(resolve => setTimeout(() => resolve({ series }), 0));
  },
  fetchAnnotations(range: string): Promise<Annotation[]> {
    const seedMap: Record<string, number> = { '1h': 1, '6h': 6, '7d': 7 };
    const rand = seededRandom(seedMap[range] || 24);
    const xs = rangeTicks(range);
    const pool = [
      { type: 'release' as const, title: '版本发布 v8.0.36' },
      { type: 'switch' as const, title: '主备切换' },
      { type: 'alert' as const, title: 'CPU 使用率告警' },
      { type: 'alert' as const, title: '慢 SQL 告警' },
      { type: 'release' as const, title: '参数配置变更' },
    ];
    const n = Math.max(1, Math.floor(xs.length / 8));
    const anns: Annotation[] = [];
    for (let i = 0; i < n; i++) {
      const idx = Math.floor(rand() * xs.length);
      const t = pool[Math.floor(rand() * pool.length)];
      anns.push({ time: xs[idx], title: t.title, type: t.type });
    }
    return new Promise(resolve => setTimeout(() => resolve(anns), 0));
  },
};

/* ---------------- API Provider（对接 Go apiserver，失败降级 mock） ----------------
 * 响应为统一包裹 {code, message, data}，此处解包；
 * name/color/axis/type/unit 由面板配置透传给服务端原样回显，保证渲染层零改动。
 */
export const ApiProvider = {
  fetchSeries(q: QueryRequest): Promise<{ series: ResolvedTarget[] }> {
    const params = new URLSearchParams({
      metric: q.metric, scope: q.scope || 'global', range: q.range, shift: q.shift || '',
      agg: q.agg || 'avg', groupBy: q.groupBy || 'none', dbType: q.dbType || '',
      name: q.name || '', color: q.color || '', axis: q.axis || '', type: q.type || '', unit: q.unit || '',
    });
    return fetch(`${API_BASE}/series?${params}`)
      .then(r => { if (!r.ok) throw new Error('bad status'); return r.json(); })
      .then((env: { code: number; data: { series: ResolvedTarget[] } }) => {
        if (env.code !== 0) throw new Error('bad envelope');
        return env.data;
      })
      .catch(() => MockProvider.fetchSeries(q));
  },
  fetchAnnotations(range: string): Promise<Annotation[]> {
    return fetch(`${API_BASE}/annotations?range=${range}`)
      .then(r => { if (!r.ok) throw new Error('bad status'); return r.json(); })
      .then((env: { code: number; data: Annotation[] }) => {
        if (env.code !== 0) throw new Error('bad envelope');
        return Array.isArray(env.data) ? env.data : [];
      })
      .catch(() => MockProvider.fetchAnnotations(range));
  },
};

type Provider = typeof MockProvider;
let activeProvider: Provider = MockProvider;
export function setQueryProvider(p: Provider) { activeProvider = p || MockProvider; }
export function fetchSeries(q: QueryRequest) { return activeProvider.fetchSeries(q); }
export function fetchAnnotations(range: string) { return activeProvider.fetchAnnotations(range); }
