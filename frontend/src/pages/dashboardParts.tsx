import { useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Chart } from '../components/Chart';
import { fetchAnnotations, fetchSeries, rangeTicks } from '../lib/query';
import { gaugeOpt, tsChartOpt } from '../lib/chartOptions';
import type { Annotation, DashCfg, Panel, ResolvedTarget } from '../lib/types';
import { CLUSTERS } from '../lib/mockData';

/* 面板数据解析：targets → 查询层取数 → 渲染序列；支持对比昨日 + 事件标注 */
export async function resolveTargets(p: Panel, cfg: DashCfg): Promise<{ xs: string[]; targets: ResolvedTarget[]; annotations: Annotation[] }> {
  const xs = rangeTicks(cfg.range);
  const tgs = (p.targets || []).filter(t => t.metric);
  const compare = !!cfg.compareYesterday && p.type !== 'gauge' && p.type !== 'stat';
  const out: ResolvedTarget[] = [];
  let pending = tgs.length * (compare ? 2 : 1);
  let annotations: Annotation[] | null = null;

  const panelColor = (metric: string) => ({
    qps: '#006aff', cpu: '#00a3e0', mem: '#7a5af8', sessions: '#ff9500',
    slow_sql: '#7a5af8', lock_wait: '#f53f3f', disk: '#00b365', repl_delay: '#f53f3f',
  } as Record<string, string>)[metric] || '#006aff';

  const result = await new Promise<{ xs: string[]; targets: ResolvedTarget[]; annotations: Annotation[] }>(resolve => {
    const tryFinish = () => {
      if (pending > 0 || annotations === null) return;
      resolve({ xs, targets: out, annotations });
    };
    const onSeries = (t: any, suffix: string, dashed: boolean) => (res: { series: ResolvedTarget[] }) => {
      (res.series || []).forEach(s => {
        out.push({
          name: suffix ? s.name + suffix : s.name,
          data: s.data,
          color: s.color || panelColor(t.metric),
          axis: s.axis === 'right' ? 'right' : 'left',
          type: s.type || 'line',
          unit: s.unit || '',
          dashed,
        });
      });
      pending -= 1;
      tryFinish();
    };
    if (!tgs.length) { pending = 0; annotations = []; tryFinish(); return; }
    tgs.forEach(t => {
      const q = { metric: t.metric, scope: p.scope, range: cfg.range, shift: '', name: t.name, color: t.color, axis: t.axis, type: t.type, unit: t.unit, agg: t.agg || '', groupBy: t.groupBy || '', dbType: t.dbType || '' };
      fetchSeries(q).then(onSeries(t, '', false));
      if (compare) fetchSeries({ ...q, shift: '24h' }).then(onSeries(t, '（昨日）', true));
    });
    fetchAnnotations(cfg.range).then(anns => {
      const annCfg = p.annotations || { enable: false, types: [] as string[] };
      annotations = annCfg.enable && Array.isArray(annCfg.types)
        ? (anns || []).filter(a => annCfg.types.includes(a.type))
        : [];
      tryFinish();
    });
  });
  return result;
}

/* 异步面板数据 Hook：reloadKey 变化时重新取数 */
export function usePanelData(panel: Panel, cfg: DashCfg, reloadKey: number) {
  const [data, setData] = useState<{ xs: string[]; targets: ResolvedTarget[]; annotations: Annotation[] } | null>(null);
  useEffect(() => {
    let alive = true;
    resolveTargets(panel, cfg).then(d => { if (alive) setData(d); });
    return () => { alive = false; };
  }, [panel, cfg, reloadKey]);
  return data;
}

export function thresholdColor(steps: { value: number; color: string }[] | undefined, val: number): string {
  let c = '#1d2b45';
  (steps || []).forEach(s => { if (val >= s.value) c = s.color; });
  return c;
}

/* Stat 单值面板 */
export function StatPanel({ panel, targets }: { panel: Panel; targets: ResolvedTarget[] }) {
  const t0 = targets[0];
  const val = t0.data[t0.data.length - 1];
  const color = thresholdColor(panel.thresholds?.steps, val);
  const sparkOpt = useMemo(() => ({
    grid: { left: 2, right: 2, top: 4, bottom: 2 },
    xAxis: { type: 'category', show: false, boundaryGap: false, data: t0.data.map((_, i) => i) },
    yAxis: { type: 'value', show: false },
    tooltip: { show: false },
    series: [{
      type: 'line', smooth: true, symbol: 'none', data: t0.data,
      lineStyle: { width: 1.5, color: t0.color },
      areaStyle: { color: { type: 'linear', x: 0, y: 0, x2: 0, y2: 1, colorStops: [{ offset: 0, color: t0.color + '33' }, { offset: 1, color: t0.color + '05' }] } as any },
    }],
  }), [t0]);
  return (
    <div className="stat-panel">
      <div className="stat-val" style={{ color }}>{val}<span className="stat-unit">{t0.unit || ''}</span></div>
      <Chart option={sparkOpt} className="stat-spark" />
    </div>
  );
}

/* 表格面板：指标汇总 */
export function TablePanel({ panel, targets }: { panel: Panel; targets: ResolvedTarget[] }) {
  const steps = panel.thresholds?.steps || [];
  return (
    <div className="table-panel">
      <table>
        <thead><tr><th>指标</th><th>当前值</th><th>最小</th><th>最大</th><th>平均</th></tr></thead>
        <tbody>
          {targets.map((t, i) => {
            const cur = t.data[t.data.length - 1];
            const min = Math.min(...t.data);
            const max = Math.max(...t.data);
            const avg = Math.round(t.data.reduce((a, b) => a + b, 0) / t.data.length);
            const color = thresholdColor(steps, cur);
            return (
              <tr key={i}>
                <td><span className="tp-dot" style={{ background: t.color }}></span>{t.name}</td>
                <td className="tp-cur" style={{ color }}>{cur}{t.unit || ''}</td>
                <td>{min}</td><td>{max}</td><td>{avg}</td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

/* 面板视图：按类型渲染，异步取数，支持下钻 */
export function PanelView({ panel, cfg, reloadKey, onEdit, onDelete, onZoom, height }: {
  panel: Panel; cfg: DashCfg; reloadKey: number;
  onEdit: () => void; onDelete: () => void; onZoom: () => void; height?: number;
}) {
  const data = usePanelData(panel, cfg, reloadKey);
  const navigate = useNavigate();
  const drillRef = useRef(panel.drilldown);
  drillRef.current = panel.drilldown;

  const tgs = (panel.targets || []).filter(t => t.metric);
  const scopeName = panel.scope === 'global' ? '全局' : (CLUSTERS.find(c => c.id === panel.scope) || { name: panel.scope }).name;
  const drill = panel.drilldown && panel.drilldown.targetId;

  const chartOption = useMemo(() => {
    if (!data || !data.targets.length) return null;
    const type = panel.type;
    if (type === 'gauge') return gaugeOpt(panel.title, data.targets[0].data[data.targets[0].data.length - 1], data.targets[0].unit);
    if (type === 'stat' || type === 'table') return null;
    const targets = data.targets.map(t => ({ ...t, type: type === 'bar' ? 'bar' : t.type }));
    return tsChartOpt(panel.title, data.xs, targets, {
      legend: panel.legend !== false,
      thresholds: panel.thresholds?.steps || [],
      annotations: data.annotations,
      style: panel.style || {},
    });
  }, [data, panel]);

  const onReady = (chart: any) => {
    chart.on('click', () => {
      const d = drillRef.current;
      if (!d || !d.targetId) return;
      sessionStorage.setItem('drillContext', JSON.stringify({ range: cfg.range, scope: panel.scope }));
      if (d.kind === 'cluster') navigate(`/cluster/${d.targetId}`);
      else if (d.kind === 'instance') {
        const cl = CLUSTERS.find(c => c.instances.some(i => i.id === d.targetId));
        if (cl) navigate(`/instance/${cl.id}/${d.targetId}`);
      }
    });
  };

  return (
    <div className="panel" style={{ ['--w' as any]: panel.w || 6 }}>
      <div className="panel-bar">
        <span className="panel-title">{panel.title}</span>
        {drill ? <span className="panel-drill" title="点击图表可下钻">↗</span> : null}
        <span className="panel-meta">{tgs.length} 个指标 · {scopeName}</span>
        <PanelMenu onEdit={onEdit} onDelete={onDelete} onZoom={onZoom} />
      </div>
      <div className="panel-chart" style={height ? { height } : undefined}>
        {!data ? null
          : panel.type === 'stat' ? <StatPanel panel={panel} targets={data.targets} />
          : panel.type === 'table' ? <TablePanel panel={panel} targets={data.targets} />
          : chartOption ? <Chart option={chartOption} onReady={onReady} style={{ height: '100%' }} />
          : null}
      </div>
    </div>
  );
}

/* 面板 ⋯ 菜单（hover 显示，点击展开） */
function PanelMenu({ onEdit, onDelete, onZoom }: { onEdit: () => void; onDelete: () => void; onZoom: () => void }) {
  const [open, setOpen] = useState(false);
  return (
    <span style={{ position: 'relative', marginLeft: 2 }}>
      {open && (
        <div className="dash-card-popover" style={{ position: 'absolute', top: 26, right: 0, zIndex: 50 }}>
          <button className="exp-more-item" onClick={() => { setOpen(false); onZoom(); }}>查看</button>
          <button className="exp-more-item" onClick={() => { setOpen(false); onEdit(); }}>编辑</button>
          <button className="exp-more-item" onClick={() => { setOpen(false); onDelete(); }}>删除</button>
        </div>
      )}
      <button className="panel-menu-btn" title="面板操作" onClick={e => { e.stopPropagation(); setOpen(o => !o); }}>⋯</button>
    </span>
  );
}

/* 面板放大全屏 */
export function ZoomOverlay({ panel, cfg, onClose }: { panel: Panel; cfg: DashCfg; onClose: () => void }) {
  const data = usePanelData(panel, cfg, 0);
  const navigate = useNavigate();
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose(); };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [onClose]);

  const onReady = (chart: any) => {
    chart.on('click', () => {
      const d = panel.drilldown;
      if (!d || !d.targetId) return;
      if (d.kind === 'cluster') navigate(`/cluster/${d.targetId}`);
      else if (d.kind === 'instance') {
        const cl = CLUSTERS.find(c => c.instances.some(i => i.id === d.targetId));
        if (cl) navigate(`/instance/${cl.id}/${d.targetId}`);
      }
    });
  };

  const option = useMemo(() => {
    if (!data || !data.targets.length) return null;
    if (panel.type === 'gauge') return gaugeOpt(panel.title, data.targets[0].data[data.targets[0].data.length - 1], data.targets[0].unit);
    if (panel.type === 'stat' || panel.type === 'table') return null;
    const targets = data.targets.map(t => ({ ...t, type: panel.type === 'bar' ? 'bar' : t.type }));
    return tsChartOpt(panel.title, data.xs, targets, {
      legend: panel.legend !== false,
      thresholds: panel.thresholds?.steps || [],
      annotations: data.annotations,
      style: panel.style || {},
    });
  }, [data, panel]);

  return (
    <div className="panel-fullscreen" onMouseDown={e => { if (e.target === e.currentTarget) onClose(); }}>
      <div className="pf-bar"><span>{panel.title}</span><button className="btn sm" onClick={onClose}>关闭</button></div>
      <div className="pf-chart">
        {!data ? null
          : panel.type === 'stat' ? <StatPanel panel={panel} targets={data.targets} />
          : panel.type === 'table' ? <TablePanel panel={panel} targets={data.targets} />
          : option ? <Chart option={option} onReady={onReady} style={{ height: '100%' }} />
          : null}
      </div>
    </div>
  );
}
