/* ================= 卡片渲染器注册表（card-protocol/1.0 MVP 子集） =================
 * 新增卡片类型 = 注册渲染器，不改协议；未知 card_type 走 FallbackRenderer 绝不白屏。
 */
import { useMemo } from 'react';
import { Chart } from '../components/Chart';
import { TIP, axisStyle } from '../lib/chartOptions';
import type { CardEnvelope, CardType } from '../lib/types';

export interface CardRendererProps {
  card: CardEnvelope;
  onAsk?: (question: string) => void;
}

type Renderer = (props: CardRendererProps) => JSX.Element | null;

const registry = new Map<CardType, Renderer>();

export function registerCardRenderer(type: CardType, renderer: Renderer) { registry.set(type, renderer); }
export function resolveRenderer(type: string): Renderer { return registry.get(type as CardType) || FallbackRenderer; }

function FallbackRenderer({ card }: CardRendererProps) {
  return (
    <div className="ccard-fallback">
      <div className="ccard-fallback-title">卡片类型「{card.card_type}」暂不支持</div>
      <div className="ccard-fallback-text">{card.fallback_text}</div>
    </div>
  );
}

/* ---------- text ---------- */
const TextCard: Renderer = ({ card }) => (
  <div className="ccard-text">{card.payload?.markdown ?? card.fallback_text}</div>
);

/* ---------- metric_chart ---------- */
const MetricChartCard: Renderer = ({ card }) => {
  const option = useMemo(() => {
    const p = card.payload || {};
    const points: [string, number][] = p.data?.points || [];
    const xs = points.map(pt => pt[0]);
    const metrics = p.metrics || [];
    const colors = ['#006aff', '#ff9500', '#00b365', '#7a5af8', '#f53f3f'];
    const series = metrics.map((m: any, i: number) => ({
      name: m.label || m.name, type: p.chart_type === 'bar' ? 'bar' : 'line',
      smooth: true, symbol: 'none', data: points.map(pt => pt[1]),
      lineStyle: { width: 2, color: colors[i % colors.length] },
      itemStyle: { color: colors[i % colors.length] },
      areaStyle: p.chart_type === 'area' || metrics.length === 1 ? {
        color: { type: 'linear', x: 0, y: 0, x2: 0, y2: 1, colorStops: [
          { offset: 0, color: colors[i % colors.length] + '33' }, { offset: 1, color: colors[i % colors.length] + '05' }] } as any,
      } : undefined,
    }));
    const mkData: any[] = [];
    (p.thresholds || []).forEach((t: any) => mkData.push({
      yAxis: t.value, lineStyle: { color: t.severity === 'critical' ? '#f53f3f' : '#ff9500', width: 1, type: 'dashed' },
      label: { show: true, formatter: t.label || `阈值 ${t.value}`, position: 'insideEndTop', fontSize: 10, color: '#8a97ad' },
    }));
    if (series.length) series[0].markLine = { silent: true, symbol: 'none', data: mkData };
    return {
      tooltip: { trigger: 'axis', ...TIP },
      legend: metrics.length > 1 ? { show: true, top: 0, textStyle: { color: '#4e5d78', fontSize: 11 } } : undefined,
      grid: { left: 48, right: 16, top: metrics.length > 1 ? 30 : 18, bottom: 24 },
      xAxis: { type: 'category', data: xs, boundaryGap: false, ...axisStyle },
      yAxis: { type: 'value', ...axisStyle },
      series,
    };
  }, [card]);
  return <Chart option={option} style={{ height: 210 }} />;
};

/* ---------- data_table ---------- */
const DataTableCard: Renderer = ({ card, onAsk }) => {
  const p = card.payload || {};
  const columns = p.columns || [];
  const rows = p.rows || [];
  return (
    <div className="ccard-table">
      <table>
        <thead><tr>{columns.map((c: any) => <th key={c.key}>{c.label}</th>)}
          {p.row_actions?.includes('ask') && <th style={{ width: 70 }}>操作</th>}</tr></thead>
        <tbody>
          {rows.map((r: any, i: number) => (
            <tr key={i}>
              {columns.map((c: any) => {
                const v = r[c.key];
                const isSql = c.type === 'sql';
                const isStatus = c.type === 'status';
                return (
                  <td key={c.key} className={isSql ? 'mono' : ''}>
                    {isStatus ? <span className={`ccard-sev sev-${String(v).toLowerCase()}`}>{v}</span> : String(v ?? '—')}
                  </td>
                );
              })}
              {p.row_actions?.includes('ask') && (
                <td><span className="ccard-ask" onClick={() => onAsk?.(`解释 ${r[columns[0].key]} 这一行数据`)}>问 AI</span></td>
              )}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};

/* ---------- task_progress ---------- */
const TaskProgressCard: Renderer = ({ card }) => {
  const p = card.payload || {};
  const pct = p.status === 'done' ? 100 : (p.progress ?? 0);
  const statusText: Record<string, string> = { pending: '排队中', running: '执行中', done: '已完成', failed: '失败', cancelled: '已取消' };
  return (
    <div className="tprog">
      <div className="tprog-head">
        <span className={`tprog-dot tprog-dot-${p.status}`}></span>
        <span className="tprog-type">{p.task_type || '任务'}</span>
        <span className={`tprog-status tprog-status-${p.status}`}>{statusText[p.status] || p.status}</span>
        <span className="tprog-taskid">{p.task_id}</span>
      </div>
      {p.status !== 'done' && (
        <>
          <div className="tprog-bar"><i style={{ width: `${pct}%` }}></i></div>
          <div className="tprog-stage">{p.stage || ''} {p.progress != null ? `${p.progress}%` : ''}</div>
        </>
      )}
      {Array.isArray(p.stages) && p.stages.length > 0 && (
        <div className="tprog-steps">
          {p.stages.map((s: any, i: number) => (
            <div key={i} className={`tprog-step tprog-step-${s.status}`}>
              <span className="tprog-step-ico">{s.status === 'done' ? '✓' : s.status === 'running' ? '◐' : s.status === 'failed' ? '✕' : '·'}</span>
              {s.name}
            </div>
          ))}
        </div>
      )}
      {p.status === 'done' && <div className="tprog-done">✓ 任务完成，结果见下方报告卡片</div>}
      {p.status === 'failed' && <div className="tprog-failed">✕ {p.error || '任务执行失败'}</div>}
    </div>
  );
};

/* ---------- diagnosis_report ---------- */
const SEV_TEXT: Record<string, string> = { normal: '正常', notice: '提示', warning: '警告', critical: '严重' };
const DiagnosisReportCard: Renderer = ({ card }) => {
  const p = card.payload || {};
  return (
    <div className={`dr sev-band-${p.severity || 'notice'}`}>
      <div className="dr-summary">{p.summary}</div>
      {Array.isArray(p.root_causes) && p.root_causes.length > 0 && (
        <div className="dr-sec">
          <div className="dr-sec-title">根因假设</div>
          {p.root_causes.map((rc: any, i: number) => (
            <div className="dr-root" key={i}>
              <div className="dr-root-hyp">{rc.hypothesis}</div>
              <div className="dr-root-conf"><i style={{ width: `${Math.round((rc.confidence || 0) * 100)}%` }}></i><span>{Math.round((rc.confidence || 0) * 100)}%</span></div>
            </div>
          ))}
        </div>
      )}
      <div className="dr-sec">
        <div className="dr-sec-title">发现（{p.findings?.length || 0}）</div>
        {Array.isArray(p.findings) && p.findings.map((f: any, i: number) => (
          <div className="dr-finding" key={i}>
            <span className="dr-finding-cat">{f.category}</span>
            <span>{f.detail}</span>
          </div>
        ))}
      </div>
      <div className="dr-sec">
        <div className="dr-sec-title">建议</div>
        {Array.isArray(p.suggestions) && p.suggestions.map((s: string, i: number) => (
          <div className="dr-sug" key={i}><span className="dr-sug-no">{i + 1}</span>{s}</div>
        ))}
      </div>
      <div className="dr-foot">
        <span className={`dr-sev dr-sev-${p.severity}`}>{SEV_TEXT[p.severity] || p.severity}</span>
        {p.provider === 'vendor_agent' && <span className="dr-provider">专家诊断引擎</span>}
      </div>
    </div>
  );
};

/* ---------- 注册（应用启动时执行一次） ---------- */
let registered = false;
export function ensureCardRenderers() {
  if (registered) return;
  registered = true;
  registerCardRenderer('text', TextCard);
  registerCardRenderer('metric_chart', MetricChartCard);
  registerCardRenderer('data_table', DataTableCard);
  registerCardRenderer('task_progress', TaskProgressCard);
  registerCardRenderer('diagnosis_report', DiagnosisReportCard);
}

/* ---------- 卡片容器：信封统一处理（标题/状态/交互），渲染器只消费 payload ---------- */
export function CardContainer({ card, onAsk }: CardRendererProps) {
  const Renderer = resolveRenderer(card.card_type);
  return (
    <div className={`ccard ${card.status === 'streaming' ? 'ccard-streaming' : ''}`}>
      <div className="ccard-head">
        <span className="ccard-title">{card.title}</span>
        {card.status === 'streaming' && <span className="ccard-status">生成中…</span>}
        {card.source?.agent && <span className="ccard-agent">{card.source.agent}</span>}
      </div>
      <div className="ccard-body"><Renderer card={card} onAsk={onAsk} /></div>
    </div>
  );
}
