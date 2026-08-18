/* ================= DB Copilot · 统一查询数据层 ================= */
/* 查询协议（对齐后端接口预期，mock / api 双 Provider 并存）：
 *   fetchSeries(query, cb)
 *     query = { metric, scope, range, shift, name, color, axis, type, unit,
 *               agg, groupBy, dbType }
 *               agg:     avg|max|min|last|p95（聚合方式）
 *               groupBy: none|cluster|instance（分组维度）
 *               dbType:  mysql|pg|redis…（库类型过滤，空=全部）
 *     cb(null, { series: [{ name, color, unit, axis, type, data: [..] }] })   data 与 rangeTicks 等长
 *   fetchAnnotations(range, cb)
 *     cb(null, [{ time, title, type }])   type: release | switch | alert
 * 后端就绪后调用 setQueryProvider(ApiProvider) 即可，渲染层零改动。
 */
const API_BASE = '/api/dash'; // 后端网关就绪后替换为真实地址

const HOURS = Array.from({ length: 24 }, (_, i) => `${String(i).padStart(2, '0')}:00`);

/* 根据时间范围生成轴标签 */
function rangeTicks(range) {
  if (range === '1h') return Array.from({ length: 12 }, (_, i) => `${i * 5}m`);
  if (range === '6h') return Array.from({ length: 12 }, (_, i) => `${i * 30}m`);
  if (range === '7d') return ['周一', '周二', '周三', '周四', '周五', '周六', '周日'];
  return HOURS; // 24h
}
function rangeSpike(range) { return range === '7d' ? 3 : 14; }

/* 可复现伪随机：固定种子 → 时间偏移对比的"昨日"数据保持稳定 */
function seededRandom(seed) {
  let a = seed >>> 0;
  return function () {
    a |= 0; a = (a + 0x6D2B79F5) | 0;
    let t = Math.imul(a ^ (a >>> 15), 1 | a);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

function genSeries(base, jitter, spikeAt, seed) {
  const rand = seed === undefined ? Math.random : seededRandom(seed);
  const arr = [];
  for (let i = 0; i < 24; i++) {
    let v = base + Math.sin(i / 3.2) * jitter + (rand() - 0.5) * jitter;
    if (spikeAt && Math.abs(i - spikeAt) <= 1) v += jitter * 2.6;
    arr.push(Math.max(1, Math.round(v)));
  }
  return arr;
}

/* 聚合变换（mock 近似）：对已生成的数据数组按 agg 类型做变换 */
function applyAgg(data, agg) {
  if (!agg || agg === 'avg') return data; // avg = 原值
  const arr = data.slice();
  if (agg === 'max') return arr.map(v => Math.round(v * 1.35 + 5));   // 上抬
  if (agg === 'min') return arr.map(v => Math.max(1, Math.round(v * 0.65 - 3))); // 下压
  if (agg === 'last') { const lv = arr[arr.length - 1] || 1; return arr.map(() => lv); } // 末点平铺
  if (agg === 'p95') {
    const sorted = data.slice().sort((a, b) => a - b);
    const p95v = sorted[Math.floor(sorted.length * 0.95)] || sorted[sorted.length - 1] || 1;
    return arr.map(v => Math.round((v + p95v) / 2)); // 95 分位附近
  }
  return data;
}

/* 按 groupBy / dbType 解析出分组目标列表（name + seed）*/
function resolveGroups(q) {
  const clusters = (window.CLUSTERS || []).filter(c => !q.dbType || c.type === q.dbType);
  if (q.groupBy === 'cluster') {
    return clusters.map((c, i) => ({ name: c.name, seedBase: i + 1 }));
  }
  if (q.groupBy === 'instance') {
    return clusters.flatMap((c, ci) => c.instances.map((inst, ii) => ({
      name: inst.name, seedBase: ci * 100 + ii + 1,
    }))).slice(0, 12); // 最多 12 条，避免图表过密
  }
  return [{ name: '', seedBase: 0 }]; // none
}

/* 分组多序列时自动分配的调色板 */
const GROUP_PALETTE = ['#006aff', '#ff9500', '#00b365', '#7a5af8', '#f53f3f', '#00a3e0', '#d46b08', '#0091c9', '#6a3fd4', '#00897b', '#d4380d', '#8a97ad'];

/* ---------------- Mock Provider（后端就绪前的数据生成器） ---------------- */
const MockProvider = {
  fetchSeries(q, cb) {
    const m = (window.METRIC_LIB || []).find(x => x.id === q.metric)
      || { name: q.metric, base: 50, jitter: 10, unit: '' };
    const xs = rangeTicks(q.range);
    const spike = rangeSpike(q.range);
    const shiftSeed = q.shift ? 20260809 : 20260811;
    const groups = resolveGroups(q);
    const useGroup = q.groupBy && q.groupBy !== 'none' && groups.length > 1;
    const series = groups.map((g, gi) => {
      const seed = (shiftSeed + g.seedBase * 7919) >>> 0; // 每组不同种子
      let data = genSeries(m.base, m.jitter, spike, seed).slice(0, xs.length);
      data = applyAgg(data, q.agg);
      return {
        name: useGroup ? (q.name ? q.name + ' · ' + g.name : g.name) : (q.name || m.name),
        data,
        color: useGroup ? GROUP_PALETTE[gi % GROUP_PALETTE.length] : (q.color || ''),
        unit: q.unit || m.unit,
        axis: q.axis || 'left',
        type: q.type || 'line',
      };
    });
    setTimeout(() => cb(null, { series }), 0);
  },
  fetchAnnotations(range, cb) {
    const seedMap = { '1h': 1, '6h': 6, '7d': 7 };
    const rand = seededRandom(seedMap[range] || 24);
    const xs = rangeTicks(range);
    const pool = [
      { type: 'release', title: '版本发布 v8.0.36' },
      { type: 'switch', title: '主备切换' },
      { type: 'alert', title: 'CPU 使用率告警' },
      { type: 'alert', title: '慢 SQL 告警' },
      { type: 'release', title: '参数配置变更' },
    ];
    const n = Math.max(1, Math.floor(xs.length / 8));
    const anns = [];
    for (let i = 0; i < n; i++) {
      const idx = Math.floor(rand() * xs.length);
      const t = pool[Math.floor(rand() * pool.length)];
      anns.push({ time: xs[idx], title: t.title, type: t.type });
    }
    setTimeout(() => cb(null, anns), 0);
  },
};

/* ---------------- API Provider（对接后端网关，失败降级 mock） ---------------- */
const ApiProvider = {
  fetchSeries(q, cb) {
    const params = new URLSearchParams({
      metric: q.metric, scope: q.scope || 'global', range: q.range, shift: q.shift || '',
      agg: q.agg || 'avg', groupBy: q.groupBy || 'none', dbType: q.dbType || '',
    });
    fetch(`${API_BASE}/series?${params}`)
      .then(r => { if (!r.ok) throw new Error('bad status'); return r.json(); })
      .then(j => cb(null, j))
      .catch(() => MockProvider.fetchSeries(q, cb));
  },
  fetchAnnotations(range, cb) {
    fetch(`${API_BASE}/annotations?range=${range}`)
      .then(r => { if (!r.ok) throw new Error('bad status'); return r.json(); })
      .then(j => cb(null, Array.isArray(j) ? j : []))
      .catch(() => MockProvider.fetchAnnotations(range, cb));
  },
};

let activeProvider = MockProvider;
function setQueryProvider(p) { activeProvider = p || MockProvider; }
function fetchSeries(q, cb) { activeProvider.fetchSeries(q, cb); }
function fetchAnnotations(range, cb) { activeProvider.fetchAnnotations(range, cb); }
