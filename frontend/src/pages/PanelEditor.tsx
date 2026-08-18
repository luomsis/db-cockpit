import { useEffect, useMemo, useRef, useState } from 'react';
import { Chart } from '../components/Chart';
import { METRIC_LIB, CLUSTERS } from '../lib/mockData';
import { DEFAULT_PANEL_STYLE, gaugeOpt, tsChartOpt } from '../lib/chartOptions';
import { normalizeStyle, blankTarget } from '../lib/dashboards';
import { resolveTargets, StatPanel, TablePanel } from './dashboardParts';
import type { DashCfg, Panel, PanelTarget, ThresholdStep } from '../lib/types';

const ANN_TYPES: [string, string][] = [['release', '发布'], ['switch', '主备切换'], ['alert', '告警']];

/* 添加 / 编辑面板弹窗：
 * 主区（上=实时预览图表，下=指标序列行——逐指标选项贴近图表）
 * 右侧栏（面板级全局选项：基础设置 / 高级显示 / 图表样式） */
export function PanelEditor({ panel, cfg, editing, onOk, onClose }: {
  panel: Panel; cfg: DashCfg; editing: boolean;
  onOk: (draft: Panel) => void; onClose: () => void;
}) {
  const [title, setTitle] = useState(panel.title);
  const [type, setType] = useState<Panel['type']>(panel.type);
  const [scope, setScope] = useState(panel.scope);
  const [legend, setLegend] = useState(panel.legend);
  const [targets, setTargets] = useState<PanelTarget[]>(panel.targets.length ? panel.targets.map(t => ({ ...t })) : [blankTarget()]);
  const [steps, setSteps] = useState<ThresholdStep[]>(panel.thresholds.steps.map(s => ({ ...s })));
  const [annEnable, setAnnEnable] = useState(panel.annotations.enable);
  const [annTypes, setAnnTypes] = useState<string[]>(panel.annotations.types);
  const [drill, setDrill] = useState<string>(panel.drilldown ? `${panel.drilldown.kind}:${panel.drilldown.targetId}` : '');
  const [style, setStyle] = useState<Panel['style']>(normalizeStyle(panel.style));
  const [advOpen, setAdvOpen] = useState(() => !!(panel.thresholds.steps.length || panel.annotations.enable || panel.drilldown));
  const [styleOpen, setStyleOpen] = useState(() =>
    Object.keys(DEFAULT_PANEL_STYLE).some(k => (panel.style as any)?.[k] !== (DEFAULT_PANEL_STYLE as any)[k]));
  const [previewCollapsed, setPreviewCollapsed] = useState(false);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose(); };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [onClose]);

  const draft: Panel = useMemo(() => ({
    ...panel,
    title: title.trim() || '未命名面板',
    type, scope, legend, style,
    targets: targets.filter(t => t.metric),
    thresholds: { steps: steps.filter(s => !isNaN(s.value)) },
    annotations: { enable: annEnable, types: annTypes },
    drilldown: drill
      ? (drill.startsWith('cluster:')
        ? { kind: 'cluster', targetId: drill.slice('cluster:'.length) }
        : { kind: 'instance', targetId: drill.slice('instance:'.length) })
      : null,
  }), [panel, title, type, scope, legend, style, targets, steps, annEnable, annTypes, drill]);

  const setTarget = (i: number, patch: Partial<PanelTarget>) => {
    setTargets(ts => ts.map((t, idx) => idx === i ? { ...t, ...patch } : t));
  };

  /* 实时预览：草稿变化防抖 300ms 后重新取数渲染 */
  const [previewData, setPreviewData] = useState<Awaited<ReturnType<typeof resolveTargets>> | null>(null);
  const timerRef = useRef<any>(null);
  useEffect(() => {
    clearTimeout(timerRef.current);
    timerRef.current = setTimeout(() => {
      if (previewCollapsed || !draft.targets.length) { setPreviewData(null); return; }
      resolveTargets(draft, cfg).then(d => setPreviewData(d));
    }, 300);
    return () => clearTimeout(timerRef.current);
  }, [draft, cfg, previewCollapsed]);

  const previewOption = useMemo(() => {
    if (!previewData || !previewData.targets.length) return null;
    if (type === 'gauge') return gaugeOpt(draft.title, previewData.targets[0].data[previewData.targets[0].data.length - 1], previewData.targets[0].unit);
    if (type === 'stat' || type === 'table') return null;
    const tg = previewData.targets.map(t => ({ ...t, type: type === 'bar' ? 'bar' : t.type }));
    return tsChartOpt(draft.title, previewData.xs, tg, {
      legend: legend !== false,
      thresholds: draft.thresholds.steps,
      annotations: previewData.annotations,
      style,
    });
  }, [previewData, type, draft, legend, style]);

  const secCls = (open: boolean) => 'dap-section' + (open ? '' : ' closed');

  return (
    <div className="dash-popover-overlay" onMouseDown={e => { if (e.target === e.currentTarget) onClose(); }}>
      <div className="dash-add-popover">
        <div className="dap-head">{editing ? '编辑面板' : '添加面板'}</div>

        <div className={`dap-layout ${previewCollapsed ? 'preview-closed' : ''}`}>
          {/* ===== 主区：图表预览（上） + 指标序列（下） ===== */}
          <div className="dap-main">
            <div className="dap-preview">
              <div className="dap-preview-head">
                {!previewCollapsed && <span className="dap-preview-title">实时预览</span>}
                {!previewCollapsed && <span className="dap-preview-sub">面板级选项（类型 / 阈值 / 样式）见右侧栏</span>}
                <button className="dap-preview-toggle" title="收起/展开预览" onClick={() => setPreviewCollapsed(c => !c)}>
                  {previewCollapsed ? '展开预览 ▴' : '收起预览 ▾'}
                </button>
              </div>
              <div className="panel-chart">
                {previewCollapsed || !previewData || !previewData.targets.length ? (
                  <div className="dap-preview-empty">{previewCollapsed ? '' : (draft.targets.length ? '暂无数据' : '请在下方添加至少一个指标')}</div>
                ) : type === 'stat' ? <StatPanel panel={draft} targets={previewData.targets} />
                  : type === 'table' ? <TablePanel panel={draft} targets={previewData.targets} />
                  : previewOption ? <Chart option={previewOption} style={{ height: '100%' }} /> : null}
              </div>
            </div>

            {/* 指标序列：每行一条曲线（指标 / 名称 / 颜色 / 轴 / 图型 / 聚合 / 分组） */}
            <div className="dap-targets-zone">
              <div className="dap-section-head" style={{ cursor: 'default' }}>
                <span className="dap-section-arrow">▾</span>指标序列
                <span className="dap-section-sub">每行一条曲线 · 共享时间轴 · 可叠加多指标、分别挂左/右 Y 轴</span>
              </div>
              <div className="dap-targets">
                {targets.map((t, i) => (
                  <div className="dap-target" key={i}>
                    <span className="dap-t-idx">{i + 1}</span>
                    <select data-f="metric" value={t.metric} onChange={e => setTarget(i, { metric: e.target.value })}>
                      {METRIC_LIB.map(m => <option key={m.id} value={m.id}>{m.name}</option>)}
                    </select>
                    <input data-f="name" placeholder="名称" value={t.name} onChange={e => setTarget(i, { name: e.target.value })} />
                    <input type="color" data-f="color" value={t.color || '#006aff'} title="颜色" onChange={e => setTarget(i, { color: e.target.value })} />
                    <select data-f="axis" value={t.axis} onChange={e => setTarget(i, { axis: e.target.value as 'left' | 'right' })} title="Y 轴">
                      <option value="left">左轴</option>
                      <option value="right">右轴</option>
                    </select>
                    <select data-f="type" value={t.type} onChange={e => setTarget(i, { type: e.target.value as PanelTarget['type'] })} title="图型">
                      {[['line', '折线'], ['bar', '柱状'], ['area', '面积'], ['points', '点']].map(([v, l]) => <option key={v} value={v}>{l}</option>)}
                    </select>
                    <select data-f="agg" title="聚合" value={t.agg || ''} onChange={e => setTarget(i, { agg: e.target.value })}>
                      {[['', '原始'], ['avg', '均值'], ['max', '最大'], ['min', '最小'], ['last', '末值'], ['p95', 'P95']].map(([v, l]) => <option key={v} value={v}>{l}</option>)}
                    </select>
                    <select data-f="groupBy" title="分组" value={t.groupBy || ''} onChange={e => setTarget(i, { groupBy: e.target.value })}>
                      {[['', '不分组'], ['cluster', '按集群'], ['instance', '按实例']].map(([v, l]) => <option key={v} value={v}>{l}</option>)}
                    </select>
                    <button className="dap-t-del" title="删除指标" onClick={() => setTargets(ts => ts.filter((_, idx) => idx !== i))}>✕</button>
                  </div>
                ))}
              </div>
              <button className="btn sm dap-add-target" onClick={() => setTargets(ts => [...ts, blankTarget()])}>＋ 添加指标</button>
            </div>
          </div>

          {/* ===== 右侧栏：全局选项（对所有指标生效） ===== */}
          <div className="dap-sidebar">
            <div className={secCls(true)}>
              <div className="dap-section-head" style={{ cursor: 'default' }}><span className="dap-section-arrow">▾</span>基础设置</div>
              <div className="dap-section-body">
                <label>标题<input value={title} onChange={e => setTitle(e.target.value)} placeholder="如：交易库 CPU 与 QPS" /></label>
                <div className="dap-row">
                  <label>图表类型
                    <select value={type} onChange={e => setType(e.target.value as Panel['type'])}>
                      <option value="timeseries">时序图（多指标同轴）</option>
                      <option value="bar">柱状图</option>
                      <option value="stat">单值 Stat（取第 1 个指标）</option>
                      <option value="gauge">仪表盘（取第 1 个指标）</option>
                      <option value="table">表格（指标汇总）</option>
                    </select>
                  </label>
                  <label>范围
                    <select value={scope} onChange={e => setScope(e.target.value)}>
                      <option value="global">全局</option>
                      {CLUSTERS.map(c => <option key={c.id} value={c.id}>{c.name}</option>)}
                    </select>
                  </label>
                </div>
                <label className="dap-check">
                  <input type="checkbox" checked={legend} onChange={e => setLegend(e.target.checked)} /> 显示图例
                </label>
              </div>
            </div>

            <div className={secCls(advOpen)}>
              <div className="dap-section-head" onClick={() => setAdvOpen(o => !o)}><span className="dap-section-arrow">▾</span>高级显示<span className="dap-section-sub">阈值 / 事件标注 / 下钻</span></div>
              <div className="dap-section-body">
                <div className="dap-subhead">阈值 · 超限分段着色 + 阈值虚线（升序生效）</div>
                <div className="dap-targets">
                  {steps.map((s, i) => (
                    <div className="dap-threshold" key={i}>
                      <input type="number" data-f="tval" value={isNaN(s.value) ? '' : s.value} placeholder="阈值" step="any"
                        onChange={e => setSteps(ss => ss.map((x, idx) => idx === i ? { ...x, value: parseFloat(e.target.value) } : x))} />
                      <input type="color" data-f="tcolor" value={s.color || '#ff9500'} title="颜色"
                        onChange={e => setSteps(ss => ss.map((x, idx) => idx === i ? { ...x, color: e.target.value } : x))} />
                      <button className="dap-t-del" title="删除阈值" onClick={() => setSteps(ss => ss.filter((_, idx) => idx !== i))}>✕</button>
                    </div>
                  ))}
                </div>
                <button className="btn sm" onClick={() => setSteps(ss => [...ss, { value: NaN, color: '#ff9500' }])}>＋ 添加阈值</button>
                <div className="dap-subhead">事件标注 · 发布/切换/告警叠加在时间轴上</div>
                <label className="dap-check">
                  <input type="checkbox" checked={annEnable} onChange={e => setAnnEnable(e.target.checked)} /> 启用事件标注
                </label>
                {annEnable && (
                  <div className="dap-ann-types">
                    {ANN_TYPES.map(([v, l]) => (
                      <label key={v} className="dap-check">
                        <input type="checkbox" value={v} checked={annTypes.includes(v)}
                          onChange={e => setAnnTypes(ts => e.target.checked ? [...ts, v] : ts.filter(x => x !== v))} /> {l}
                      </label>
                    ))}
                  </div>
                )}
                <div className="dap-subhead">下钻 · 点击图表跳转详情页（携带时间上下文）</div>
                <label>下钻目标
                  <select value={drill} onChange={e => setDrill(e.target.value)}>
                    <option value="">无</option>
                    <optgroup label="集群详情">
                      {CLUSTERS.map(c => <option key={c.id} value={`cluster:${c.id}`}>{c.name}</option>)}
                    </optgroup>
                    <optgroup label="实例 / OBServer 详情">
                      {CLUSTERS.flatMap(c => c.instances.map(i => <option key={i.id} value={`instance:${i.id}`}>{i.name}</option>))}
                    </optgroup>
                  </select>
                </label>
              </div>
            </div>

            <div className={secCls(styleOpen)}>
              <div className="dap-section-head" onClick={() => setStyleOpen(o => !o)}><span className="dap-section-arrow">▾</span>图表样式<span className="dap-section-sub">线宽 / 填充 / 堆叠 / 图例 / 坐标轴范围等</span></div>
              <div className="dap-section-body">
                <div className="dap-style-grid">
                  <label>线宽
                    <select value={style.lineWidth} onChange={e => setStyle(s => ({ ...s, lineWidth: +e.target.value }))}>
                      {[1, 2, 3].map(v => <option key={v} value={v}>{v}px</option>)}
                    </select>
                  </label>
                  <label>填充
                    <input type="range" min={0} max={80} value={style.fill} onChange={e => setStyle(s => ({ ...s, fill: +e.target.value }))} />
                  </label>
                  <label>点大小
                    <select value={style.points} onChange={e => setStyle(s => ({ ...s, points: +e.target.value }))}>
                      {[[0, '无'], [3, '3'], [5, '5'], [8, '8']].map(([v, l]) => <option key={v} value={v}>{l}</option>)}
                    </select>
                  </label>
                  <label className="dap-check">
                    <input type="checkbox" checked={style.connectNulls} onChange={e => setStyle(s => ({ ...s, connectNulls: e.target.checked }))} /> 连接空值
                  </label>
                  <label>堆叠
                    <select value={style.stack} onChange={e => setStyle(s => ({ ...s, stack: e.target.value as any }))}>
                      {[['none', '不堆叠'], ['normal', '常规'], ['percent', '百分比']].map(([v, l]) => <option key={v} value={v}>{l}</option>)}
                    </select>
                  </label>
                  <label>图例
                    <select value={style.legend} onChange={e => setStyle(s => ({ ...s, legend: e.target.value as any }))}>
                      {[['top', '顶部'], ['bottom', '底部'], ['right', '右侧'], ['hide', '隐藏']].map(([v, l]) => <option key={v} value={v}>{l}</option>)}
                    </select>
                  </label>
                  <label>Y轴最小
                    <input type="number" value={style.yMin != null ? style.yMin : ''} placeholder="自动"
                      onChange={e => setStyle(s => ({ ...s, yMin: e.target.value === '' ? null : +e.target.value }))} />
                  </label>
                  <label>Y轴最大
                    <input type="number" value={style.yMax != null ? style.yMax : ''} placeholder="自动"
                      onChange={e => setStyle(s => ({ ...s, yMax: e.target.value === '' ? null : +e.target.value }))} />
                  </label>
                  <label>小数位
                    <select value={String(style.decimals)} onChange={e => setStyle(s => ({ ...s, decimals: e.target.value === 'null' ? null : +e.target.value }))}>
                      {[['null', '自动'], ['0', '0'], ['1', '1'], ['2', '2']].map(([v, l]) => <option key={v} value={v}>{l}</option>)}
                    </select>
                  </label>
                  <label>单位覆盖
                    <input type="text" value={style.unit} placeholder="如 %" onChange={e => setStyle(s => ({ ...s, unit: e.target.value }))} />
                  </label>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div className="dap-foot">
          <button className="btn sm" onClick={onClose}>取消</button>
          <button className="btn sm primary" onClick={() => {
            if (!targets.filter(t => t.metric).length) return;
            onOk(draft);
          }}>{editing ? '保存' : '添加'}</button>
        </div>
      </div>
    </div>
  );
}
