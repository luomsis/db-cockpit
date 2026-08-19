import { useEffect, useState } from 'react';
import { ALERT_INSTANCES } from '../lib/mockData';
import { apiGet, withFallback } from '../lib/api';
import { openChatDrawer } from '../lib/chatDrawer';
import { useBreadcrumb } from '../App';
import { FilterChips, Pagination, SearchInput, Th } from '../components/tableKit';
import type { ActiveFilter } from '../components/tableKit';
import { IconAlertTriangle, IconBell, IconEye } from '../components/icons';

interface AlertRow { id?: number; name: string; severity: string; title: string; time: string; count: number }

const SEV_OPTS = [
  { value: 'P1', label: 'P1 紧急' },
  { value: 'P2', label: 'P2 重要' },
  { value: 'P3', label: 'P3 关注' },
];

/* 对象类型推导：主机 host- 前缀 / 租户 *_tenant / 实例 observer·pg-xx-0N·（主库/备库）/ 其余为集群 */
function kindOf(name: string): string {
  if (name.startsWith('host-')) return '主机';
  if (/(^|\s|@)[a-z0-9_]*tenant/i.test(name)) return '租户';
  if (/observer-|（主库）|（备库）|^pg-\S+-\d/i.test(name)) return '实例';
  return '集群';
}

function SevPill({ sev }: { sev: string }) {
  return <span className={`sev-pill ${sev.toLowerCase()}`}>{sev}</span>;
}

function AlertStat({ sev, num, lbl, ico }: { sev: string; num: number; lbl: string; ico: React.ReactNode }) {
  return (
    <div className={`host-stat ${sev}`}>
      <div className="hs-ico">{ico}</div>
      <div className="hs-body">
        <div className="hs-num">{num}</div>
        <div className="hs-lbl">{lbl}</div>
      </div>
    </div>
  );
}

export default function Alerts() {
  useBreadcrumb([{ label: '首页' }, { label: '告警中心' }]);

  /* apiserver 告警列表（失败回退本地 mock） */
  const [alerts, setAlerts] = useState<AlertRow[]>(ALERT_INSTANCES);
  useEffect(() => {
    let alive = true;
    withFallback(apiGet<{ items: AlertRow[] }>('/api/alerts'), () => null)
      .then(d => { if (alive && d?.items?.length) setAlerts(d.items); });
    return () => { alive = false; };
  }, []);

  const [kw, setKw] = useState('');
  const [fSev, setFSev] = useState('');
  const [fKind, setFKind] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  /* 列排序：级别 / 首次触发 / 次数（升 → 降 → 取消） */
  const [sortKey, setSortKey] = useState<'severity' | 'time' | 'count' | ''>('');
  const [sortDir, setSortDir] = useState<1 | -1>(1);
  const sortBy = (k: 'severity' | 'time' | 'count') => {
    if (sortKey !== k) { setSortKey(k); setSortDir(1); return; }
    if (sortDir === 1) { setSortDir(-1); return; }
    setSortKey(''); setSortDir(1);
  };

  const kinds = Array.from(new Set(alerts.map(a => kindOf(a.name))));
  const filtered = alerts.filter(a =>
    (!kw.trim() || (a.name + ' ' + a.title).toLowerCase().includes(kw.trim().toLowerCase()))
    && (!fSev || a.severity === fSev)
    && (!fKind || kindOf(a.name) === fKind));
  const sorted = sortKey
    ? [...filtered].sort((a, b) => ((a[sortKey] > b[sortKey] ? 1 : a[sortKey] < b[sortKey] ? -1 : 0)) * sortDir)
    : filtered;

  useEffect(() => { setPage(1); }, [kw, fSev, fKind]);

  const chips: ActiveFilter[] = [
    fSev && { key: 'sev', label: '级别', value: SEV_OPTS.find(o => o.value === fSev)?.label || fSev, onRemove: () => setFSev('') },
    fKind && { key: 'kind', label: '对象类型', value: fKind, onRemove: () => setFKind('') },
  ].filter(Boolean) as ActiveFilter[];

  const total = filtered.length;
  const pageCount = Math.max(1, Math.ceil(total / pageSize));
  const cur = Math.min(page, pageCount);
  const rows = sorted.slice((cur - 1) * pageSize, cur * pageSize);
  const cnt = (s: string) => alerts.filter(a => a.severity === s).length;

  /* 问 AI：打开侧边浮窗聊天并自动发送诊断 */
  const askAI = (name: string) => {
    openChatDrawer(`诊断 ${name.split(' @ ')[0]}`);
  };

  return (
    <>
      <div className="page-title">告警中心</div>
      <div className="page-desc">平台活跃告警（按级别与对象维度过滤），共 {alerts.length} 条 · P1 {cnt('P1')} / P2 {cnt('P2')} / P3 {cnt('P3')}</div>
      <div className="host-stat-row">
        <AlertStat sev="p1" num={cnt('P1')} lbl="P1 紧急" ico={<IconAlertTriangle />} />
        <AlertStat sev="p2" num={cnt('P2')} lbl="P2 重要" ico={<IconBell />} />
        <AlertStat sev="p3" num={cnt('P3')} lbl="P3 关注" ico={<IconEye />} />
      </div>
      <div className="filter-bar">
        <SearchInput value={kw} onChange={setKw} placeholder="搜索告警对象 / 内容…" />
        <FilterChips items={chips} />
      </div>
      <div className="card">
        <div className="card-head">
          <div className="card-title">活跃告警列表</div>
          <div className="card-sub">共 {total} 条</div>
        </div>
        <table className="tbl">
          <thead>
            <tr>
              <Th>告警对象</Th>
              <Th filter={{ value: fSev, options: SEV_OPTS, onChange: setFSev }} sort={{ active: sortKey === "severity", dir: sortDir, onSort: () => sortBy("severity") }}>级别</Th>
              <Th>告警内容</Th>
              <Th filter={{ value: fKind, options: kinds.map(k => ({ value: k, label: k })), onChange: setFKind }}>对象类型</Th>
              <Th sort={{ active: sortKey === "time", dir: sortDir, onSort: () => sortBy("time") }}>首次触发</Th>
              <Th sort={{ active: sortKey === "count", dir: sortDir, onSort: () => sortBy("count") }}>次数</Th>
              <th style={{ width: 72 }}>操作</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((a, i) => (
              <tr key={a.id ?? i}>
                <td className="mono">{a.name}</td>
                <td><SevPill sev={a.severity} /></td>
                <td>{a.title}</td>
                <td><span className="tag zone">{kindOf(a.name)}</span></td>
                <td className="mono">{a.time}</td>
                <td>{a.count}</td>
                <td><span style={{ color: 'var(--blue)', cursor: 'pointer' }} onClick={() => askAI(a.name)}>问 AI</span></td>
              </tr>
            ))}
            {!rows.length && <tr><td colSpan={7}><div className="empty" style={{ padding: '40px 0' }}>没有匹配的告警</div></td></tr>}
          </tbody>
        </table>
        <Pagination total={total} page={cur} pageSize={pageSize} onPage={setPage} onPageSize={setPageSize} />
      </div>
    </>
  );
}
