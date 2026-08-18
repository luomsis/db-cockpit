/* ================= 多大盘存储（服务端 apiserver 为主，localStorage 镜像 + 离线兜底） ================= */
import type { Dashboard, Panel, PanelStyle, PanelTarget } from './types';
import { DEFAULT_PANEL_STYLE } from './chartOptions';
import { METRIC_LIB } from './mockData';
import { apiGet, apiPost, apiPut, apiDelete } from './api';

const DASH_KEY = 'dbCopilotDashCfg';
const PANELS_KEY = 'dbCopilotDashPanels';
const DASHBOARDS_KEY = 'dbCopilotDashboards';

/* 预置面板：type 之外共享默认样式（normalizeStyle 补齐） */
const RAW_DEFAULT_PANELS: Omit<Panel, 'style'>[] = [
  { id: 'p1', title: 'QPS 与活跃会话', type: 'timeseries', legend: true, scope: 'global', visible: true, w: 6,
    thresholds: { steps: [] }, annotations: { enable: true, types: ['release', 'switch', 'alert'] }, drilldown: null,
    targets: [
      { metric: 'qps', name: 'QPS', color: '#006aff', axis: 'left', type: 'line' },
      { metric: 'sessions', name: '活跃会话', color: '#ff9500', axis: 'right', type: 'line' },
    ] },
  { id: 'p2', title: 'CPU / 内存同轴对比', type: 'timeseries', legend: true, scope: 'global', visible: true, w: 6,
    thresholds: { steps: [{ value: 85, color: '#ff9500' }, { value: 95, color: '#f53f3f' }] },
    annotations: { enable: false, types: ['release', 'switch', 'alert'] }, drilldown: null,
    targets: [
      { metric: 'cpu', name: 'CPU 使用率', color: '#00a3e0', axis: 'left', type: 'area' },
      { metric: 'mem', name: '内存使用率', color: '#7a5af8', axis: 'left', type: 'line' },
    ] },
  { id: 'p3', title: '慢 SQL 与锁等待', type: 'timeseries', legend: true, scope: 'global', visible: true, w: 6,
    thresholds: { steps: [] }, annotations: { enable: false, types: ['release', 'switch', 'alert'] }, drilldown: null,
    targets: [
      { metric: 'slow_sql', name: '慢 SQL', color: '#7a5af8', axis: 'left', type: 'bar' },
      { metric: 'lock_wait', name: '锁等待', color: '#f53f3f', axis: 'left', type: 'line' },
    ] },
  { id: 'p4', title: '当前活跃会话', type: 'stat', legend: false, scope: 'global', visible: true, w: 3,
    thresholds: { steps: [{ value: 150, color: '#ff9500' }, { value: 200, color: '#f53f3f' }] },
    annotations: { enable: false, types: ['release', 'switch', 'alert'] }, drilldown: null,
    targets: [{ metric: 'sessions', name: '活跃会话', color: '#ff9500', axis: 'left', type: 'line' }] },
  { id: 'p5', title: '磁盘使用率', type: 'gauge', legend: false, scope: 'global', visible: true, w: 3,
    thresholds: { steps: [] }, annotations: { enable: false, types: ['release', 'switch', 'alert'] }, drilldown: null,
    targets: [{ metric: 'disk', name: '磁盘使用率', color: '#00b365', axis: 'left', type: 'line' }] },
  { id: 'p6', title: '复制延迟', type: 'timeseries', legend: false, scope: 'global', visible: true, w: 6,
    thresholds: { steps: [{ value: 8, color: '#f53f3f' }] }, annotations: { enable: false, types: ['release', 'switch', 'alert'] }, drilldown: null,
    targets: [{ metric: 'repl_delay', name: '复制延迟', color: '#f53f3f', axis: 'left', type: 'line' }] },
];

export const DEFAULT_PANELS: Panel[] = RAW_DEFAULT_PANELS.map(p => ({ ...p, style: { ...DEFAULT_PANEL_STYLE } }));

export function normalizeStyle(s?: Partial<PanelStyle> | null): PanelStyle {
  return { ...DEFAULT_PANEL_STYLE, ...(s || {}) };
}

export function normalizePanel(p: any): Panel | null {
  if (!p) return null;
  if (!p.targets && p.metric) {
    const m = METRIC_LIB.find(x => x.id === p.metric) || { name: p.title };
    return {
      id: p.id, title: p.title, scope: p.scope || 'global', visible: p.visible !== false,
      type: ({ line: 'timeseries', bar: 'bar', gauge: 'gauge' } as any)[p.chart] || 'timeseries',
      legend: true, w: p.w || 6,
      style: normalizeStyle(p.style),
      thresholds: p.thresholds || { steps: [] },
      annotations: p.annotations || { enable: false, types: ['release', 'switch', 'alert'] },
      drilldown: p.drilldown || null,
      targets: [{ metric: p.metric, name: m.name || p.title, color: '', axis: 'left', type: 'line', agg: '', groupBy: '' }],
    };
  }
  if (Array.isArray(p.targets)) {
    p.targets = p.targets.map((t: any) => ({
      metric: t.metric, name: t.name || '', color: t.color || '',
      axis: t.axis === 'right' ? 'right' : 'left', type: t.type || 'line', unit: t.unit || '',
      agg: t.agg || '', groupBy: t.groupBy || '', dbType: t.dbType || '',
    }));
  }
  if (!p.type) p.type = 'timeseries';
  p.w = p.w || 6;
  p.style = normalizeStyle(p.style);
  p.thresholds = p.thresholds || { steps: [] };
  p.annotations = p.annotations || { enable: false, types: ['release', 'switch', 'alert'] };
  p.drilldown = p.drilldown || null;
  return p as Panel;
}

export function normalizeDashboard(d: any): Dashboard | null {
  if (!d || typeof d !== 'object') return null;
  return {
    id: d.id || ('d' + Date.now()),
    title: d.title || '未命名大盘',
    description: d.description || '',
    cfg: d.cfg || { range: '24h', refresh: '0', compareYesterday: false },
    panels: Array.isArray(d.panels) ? d.panels.map(normalizePanel).filter(Boolean) as Panel[] : [],
    createdAt: d.createdAt || Date.now(),
    updatedAt: d.updatedAt || Date.now(),
  };
}

function makeDefaultDashboard(): Dashboard {
  const now = Date.now();
  return {
    id: 'd-default', title: '数据库综合监控大盘',
    description: '默认大盘 · QPS、CPU/内存、慢 SQL、磁盘、复制延迟等核心指标',
    cfg: { range: '24h', refresh: '0', compareYesterday: false },
    panels: JSON.parse(JSON.stringify(DEFAULT_PANELS)), createdAt: now, updatedAt: now,
  };
}

function migrateOldDashboard(): Dashboard | null {
  const oldCfg = localStorage.getItem(DASH_KEY);
  const oldPanels = localStorage.getItem(PANELS_KEY);
  if (!oldCfg && !oldPanels) return null;
  let cfg: any = { range: '24h', refresh: '0', compareYesterday: false };
  try { cfg = { ...cfg, ...JSON.parse(oldCfg || '{}') }; } catch (e) { /* ignore */ }
  let panels: Panel[] = JSON.parse(JSON.stringify(DEFAULT_PANELS));
  try {
    const arr = JSON.parse(oldPanels || 'null');
    if (Array.isArray(arr)) panels = arr.map(normalizePanel).filter(Boolean) as Panel[];
  } catch (e) { /* ignore */ }
  const now = Date.now();
  return { id: 'd-default', title: '数据库综合监控大盘', description: '由历史配置迁移 · 核心指标监控', cfg, panels, createdAt: now, updatedAt: now };
}

export function loadDashboards(): Dashboard[] {
  try {
    const arr = JSON.parse(localStorage.getItem(DASHBOARDS_KEY) || 'null');
    if (Array.isArray(arr)) {
      const norm = arr.map(normalizeDashboard).filter(Boolean) as Dashboard[];
      if (norm.length) { if (norm.length !== arr.length) saveDashboards(norm); return norm; }
    }
  } catch (e) { /* ignore */ }
  const list = [migrateOldDashboard() || makeDefaultDashboard()];
  saveDashboards(list);
  return list;
}

export function saveDashboards(arr: Dashboard[]) {
  localStorage.setItem(DASHBOARDS_KEY, JSON.stringify(arr));
}
export function getDashboard(id: string): Dashboard | null {
  return loadDashboards().find(d => d.id === id) || null;
}
export function updateDashboard(id: string, patch: Partial<Dashboard>): Dashboard | null {
  const arr = loadDashboards();
  const idx = arr.findIndex(d => d.id === id);
  if (idx < 0) return null;
  arr[idx] = { ...arr[idx], ...patch, updatedAt: Date.now() };
  saveDashboards(arr);
  return arr[idx];
}

export function relTime(ts?: number): string {
  const d = Date.now() - (ts || 0);
  if (d < 60000) return '刚刚';
  if (d < 3600000) return Math.floor(d / 60000) + ' 分钟前';
  if (d < 86400000) return Math.floor(d / 3600000) + ' 小时前';
  if (d < 2592000000) return Math.floor(d / 86400000) + ' 天前';
  try { return new Date(ts!).toLocaleDateString(); } catch (e) { return ''; }
}

export function blankPanel(): Panel {
  return {
    id: '', title: '', type: 'timeseries', legend: true, scope: 'global', visible: true, w: 6,
    style: { ...DEFAULT_PANEL_STYLE },
    thresholds: { steps: [] },
    annotations: { enable: false, types: ['release', 'switch', 'alert'] },
    drilldown: null,
    targets: [{ metric: 'qps', name: '', color: '', axis: 'left', type: 'line', agg: '', groupBy: '' }],
  };
}

export function blankTarget(): PanelTarget {
  return { metric: 'qps', name: '', color: '', axis: 'left', type: 'line', agg: '', groupBy: '' };
}

/* ================= 服务端同步（apiserver 主存储，localStorage 为镜像/兜底） ================= */

function normalizeList(raw: any): Dashboard[] {
  return Array.isArray(raw) ? raw.map(normalizeDashboard).filter(Boolean) as Dashboard[] : [];
}

let migrateDone = false;

/* 启动同步：拉服务端大盘；服务端为空且本地有数据时做一次性迁移导入 */
export async function serverInit(): Promise<Dashboard[]> {
  const local = loadDashboards();
  let server: Dashboard[] = [];
  try {
    server = normalizeList(await apiGet('/api/dashboards'));
    if (!migrateDone) {
      migrateDone = true;
      if (!server.length && local.length) {
        await apiPost('/api/dashboards/import', { dashboards: local }).catch(() => { /* 离线保持本地 */ });
        server = normalizeList(await apiGet('/api/dashboards'));
      }
    }
  } catch (e) {
    return local; // apiserver 不可达 → 离线模式
  }
  if (server.length) {
    saveDashboards(server); // 本地镜像，保证离线连续
    return server;
  }
  return local;
}

export async function serverCreate(title: string, description: string, cfg?: any, panels?: any[]): Promise<Dashboard | null> {
  const d = normalizeDashboard(await apiPost<Dashboard>('/api/dashboards', {
    title, description,
    cfg: cfg || { range: '24h', refresh: '0', compareYesterday: false },
    panels: panels || [],
  }));
  return d;
}

export async function serverUpdate(id: string, patch: Partial<Dashboard>): Promise<Dashboard | null> {
  const cur = getDashboard(id);
  const body = {
    title: patch.title ?? cur?.title ?? '',
    description: patch.description ?? cur?.description ?? '',
    cfg: patch.cfg ?? cur?.cfg,
    panels: patch.panels ?? cur?.panels,
  };
  return normalizeDashboard(await apiPut<Dashboard>(`/api/dashboards/${id}`, body));
}

export async function serverDelete(id: string): Promise<void> {
  await apiDelete(`/api/dashboards/${id}`);
}
