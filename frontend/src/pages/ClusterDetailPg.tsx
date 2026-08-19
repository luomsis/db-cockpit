import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { Pill, TypeTag, Bar, Stat } from '../components/bits';
import { TopoSVG, TopoLegend } from '../components/TopoSVG';
import { MonitorTab } from '../components/MonitorTab';
import { TYPE_ICON, REPORTS } from '../lib/mockData';
import { apiGet, apiPost, apiPut, withFallback } from '../lib/api';
import { openChatDrawer } from '../lib/chatDrawer';
import { useOpDialog, runOp } from '../components/opDialog';
import { toast } from '../lib/toast';
import type { Cluster, ParamItem } from '../lib/types';
import { IconRefresh, IconBolt, IconRobot } from '../components/icons';

interface ReportRow { id: number; ico: string; title: string; desc: string; date: string; size: string }

/* PostgreSQL 集群详情：实例 + Database + 复制高可用 维度 */
const TABS = [
  { id: 'overview', label: '集群概览' },
  { id: 'db', label: '数据库管理' },
  { id: 'ha', label: '复制与高可用' },
  { id: 'monitor', label: '性能监控' },
  { id: 'param', label: '参数管理' },
  { id: 'report', label: '性能报告' },
];

export default function ClusterDetailPg({ cluster: c, reload }: { cluster: Cluster; reload: () => void }) {
  const [tab, setTab] = useState('overview');
  const [reports, setReports] = useState<ReportRow[]>([]);
  const [histFor, setHistFor] = useState<string | null>(null);
  const [history, setHistory] = useState<{ oldValue: string; newValue: string; changedAt: number }[]>([]);

  useEffect(() => {
    let alive = true;
    withFallback(apiGet<ReportRow[]>(`/api/clusters/${c.id}/reports`), () => REPORTS.map((r, i) => ({ id: i, ...r })))
      .then(d => { if (alive && d?.length) setReports(d); });
    return () => { alive = false; };
  }, [c.id]);

  /* ---- 写操作（页内弹窗 + toast 反馈） ---- */
  const ops = useOpDialog();

  const createDatabase = () => ops.prompt(`创建数据库（${c.name}）`, [
    { key: 'name', label: '数据库名称', placeholder: '如 trade_order', required: true },
    { key: 'owner', label: 'Owner', initial: 'app_rw' },
  ], v => runOp(`数据库 ${v.name} 已创建`, () =>
    apiPost(`/api/clusters/${c.id}/databases`, { name: v.name, owner: v.owner || 'app_rw' }), reload), '创建');

  const switchDrill = (instance: string) => ops.confirm('切换演练', `确认对 ${instance} 执行切换演练？该备库将升为主库。`, () =>
    runOp(`切换演练完成：${instance} 已升为主库`, () =>
      apiPost(`/api/clusters/${c.id}/replicas/${encodeURIComponent(instance)}/switch-drill`), reload),
    { okText: '执行演练', danger: true });

  const rebuildReplica = (instance: string) => ops.confirm('重建复制', `确认重建 ${instance} 的复制链路？重建期间该备库暂时落后。`, () =>
    runOp(`已开始重建 ${instance} 的复制链路`, () =>
      apiPost(`/api/clusters/${c.id}/replicas/${encodeURIComponent(instance)}/rebuild`), reload),
    { okText: '重建' });

  const editParam = (p: ParamItem) => ops.prompt(`修改参数 ${p.name}`, [
    { key: 'value', label: `新值（范围 ${p.range}）`, initial: p.value, required: true, hint: p.desc },
  ], v => {
    if (v.value === p.value) { toast.info('参数值未变化'); return; }
    runOp(`参数 ${p.name} 已修改，待工单下发`, () =>
      apiPut(`/api/clusters/${c.id}/params/${encodeURIComponent(p.name)}`, { value: v.value }), reload);
  });
  const showHistory = async (name: string) => {
    if (histFor === name) { setHistFor(null); return; }
    setHistFor(name);
    const rows = await withFallback(
      apiGet<{ oldValue: string; newValue: string; changedAt: number }[]>(`/api/clusters/${c.id}/params/${encodeURIComponent(name)}/history`),
      () => [],
    );
    setHistory(rows);
  };

  return (
    <>
      <div className="detail-head">
        <div className="cluster-icon">{TYPE_ICON[c.type]}</div>
        <div>
          <div className="detail-title">{c.name} <TypeTag t={c.type} /> <Pill st="ok" text="运行中" /></div>
          <div className="detail-sub"><span>{c.version}</span><span>{c.mode}</span><span>{c.syncMode}</span><span>{c.desc}</span></div>
        </div>
        <div className="detail-head-right">
          <button className="btn" onClick={reload}><IconRefresh size={13} /> 刷新</button>
          <button className="btn primary" onClick={() => openChatDrawer(`诊断 ${c.name}`)}><IconBolt size={13} /> 智能巡检</button>
        </div>
      </div>
      <div className="tabs">
        {TABS.map(t => (
          <div key={t.id} className={`tab ${tab === t.id ? 'active' : ''}`} onClick={() => setTab(t.id)}>{t.label}</div>
        ))}
      </div>

      {tab === 'overview' && (
        <>
          <div className="stat-row">
            <Stat num={c.instances.length} lbl="实例数" cls="ok" />
            <Stat num={(c.databases || []).length} lbl="数据库数" />
            <Stat num={`${c.cpu}%`} lbl="CPU 均值" cls={c.cpu > 80 ? 'warn' : ''} />
            <Stat num={`${c.mem}%`} lbl="内存均值" cls={c.mem > 85 ? 'warn' : ''} />
            <Stat num={`${(c.qps / 1000).toFixed(1)}k`} lbl="QPS" />
            <Stat num={c.conn} lbl="连接数" />
          </div>
          <div className="card" style={{ marginBottom: 14 }}>
            <div className="topo-wrap"><TopoSVG cluster={c} /></div>
            <TopoLegend cluster={c} />
          </div>
          <div className="card">
            <div className="card-head">
              <div className="card-title"><span className="t-ico"></span>实例健康摘要</div>
              <Link className="card-link" to="/dashboards">监控大盘 ›</Link>
            </div>
            <table className="tbl">
              <thead><tr><th>实例</th><th>角色</th><th>地址</th><th>CPU</th><th>内存</th><th>连接数</th><th>状态</th></tr></thead>
              <tbody>
                {c.instances.map(i => (
                  <tr key={i.id}>
                    <td className="mono"><Link to={`/instance/${c.id}/${i.id}`} style={{ color: 'var(--blue)' }}>{i.name}</Link></td>
                    <td>{i.role}</td>
                    <td className="mono">{i.ip}:{i.port}</td>
                    <td><Bar value={i.cpu} hot={i.cpu > 85} /></td>
                    <td><Bar value={i.mem} hot={i.mem > 85} /></td>
                    <td>{i.conn}</td><td><Pill st={i.status} /></td>
                  </tr>
                ))}
              {!c.instances.length && <tr><td colSpan={7}><div className="empty" style={{ padding: '32px 0' }}>暂无实例</div></td></tr>}
              </tbody>
            </table>
          </div>
        </>
      )}

      {tab === 'db' && (
        <div className="card">
          <div className="card-head">
            <div className="card-title"><span className="t-ico"></span>数据库管理（{c.name}）</div>
            <button className="btn sm primary" onClick={createDatabase}>＋ 创建数据库</button>
          </div>
          <table className="tbl">
            <thead><tr><th>数据库</th><th>Owner</th><th>大小</th><th>表数量</th><th>连接数 / 上限</th><th>状态</th><th>操作</th></tr></thead>
            <tbody>
              {(c.databases || []).map(d => (
                <tr key={d.name}>
                  <td className="mono">{d.name}</td>
                  <td className="mono">{d.owner}</td>
                  <td>{d.size}</td>
                  <td>{d.tables}</td>
                  <td>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                      <Bar value={Math.round((d.conn / d.connLimit) * 100)} hot={d.conn / d.connLimit > 0.8} />
                      <span style={{ fontSize: 11.5, color: 'var(--text-2)' }}>{d.conn}/{d.connLimit}</span>
                    </div>
                  </td>
                  <td><Pill st={d.status} /></td>
                  <td><span style={{ color: 'var(--blue)', cursor: 'pointer' }}>监控</span> · <span style={{ color: 'var(--blue)', cursor: 'pointer' }}>会话</span> · <span style={{ color: 'var(--blue)', cursor: 'pointer' }}>备份</span></td>
                </tr>
              ))}
            {!((c.databases || [])).length && <tr><td colSpan={7}><div className="empty" style={{ padding: '32px 0' }}>暂无数据库，点击右上角创建</div></td></tr>}
            </tbody>
          </table>
        </div>
      )}

      {tab === 'ha' && (
        <>
          <div className="stat-row">
            <Stat num={c.replicas?.filter(r => r.status === 'ok').length ?? 0} lbl="健康备库" cls="ok" />
            <Stat num={(() => { const worst = c.replicas?.find(r => r.status !== 'ok'); return worst ? `${worst.delayMs}ms` : '—'; })()} lbl="最大复制延迟" cls={c.replicas?.some(r => r.status !== 'ok') ? 'warn' : ''} />
            <Stat num={c.syncMode || '—'} lbl="同步模式" />
            <Stat num={(() => { const gb = (c.replicas || []).reduce((a, r) => a + parseFloat(r.walLag), 0); return `${gb.toFixed(2)} GB`; })()} lbl="WAL 积压合计" />
          </div>
          <div className="card">
            <div className="card-head">
              <div className="card-title"><span className="t-ico"></span>流复制状态</div>
              <span className="card-sub">延迟阈值 300ms</span>
            </div>
            <table className="tbl">
              <thead><tr><th>备库实例</th><th>角色</th><th>复制延迟</th><th>WAL 积压</th><th>状态</th><th>操作</th></tr></thead>
              <tbody>
                {(c.replicas || []).map(r => (
                  <tr key={r.instance}>
                    <td className="mono">{r.instance}</td>
                    <td>{r.role}</td>
                    <td style={{ color: r.status !== 'ok' ? 'var(--red)' : 'inherit' }}>{r.delayMs}ms</td>
                    <td>{r.walLag}</td>
                  <td><Pill st={r.status} /></td>
                  <td>
                    <span style={{ color: 'var(--blue)', cursor: 'pointer' }} onClick={() => switchDrill(r.instance)}>切换演练</span>
                    {' · '}<span style={{ color: 'var(--blue)', cursor: 'pointer' }} onClick={() => rebuildReplica(r.instance)}>重建复制</span>
                  </td>
                  </tr>
                ))}
              {!((c.replicas || [])).length && <tr><td colSpan={6}><div className="empty" style={{ padding: '32px 0' }}>暂无备库复制信息</div></td></tr>}
              </tbody>
            </table>
            <div className="advice">
              <h4>🛡️ 高可用建议</h4>
              <ul>
                <li><code>pg-order-02</code> 复制延迟 850ms 超阈值：WAL 积压 1.2GB，建议检查备库 IO 或网络带宽；</li>
                <li>quorum 模式下任一健康备库即可保证 RPO=0，当前仍满足，但建议在低峰期重建复制槽；</li>
                <li>Patroni 主备切换演练建议每月一次（最近一次 2026-07-30）。</li>
              </ul>
            </div>
          </div>
        </>
      )}

      {tab === 'monitor' && <MonitorTab />}

      {tab === 'param' && (
        <div className="card">
          <div className="card-head">
            <div className="card-title"><span className="t-ico"></span>参数管理（集群级）</div>
            <span className="card-sub">修改参数将通过工单审批后下发</span>
          </div>
          <table className="tbl">
            <thead><tr><th>参数名</th><th>当前值</th><th>可选范围</th><th>说明</th><th>状态</th><th>操作</th></tr></thead>
              <tbody>
                {c.params.map(p => (
                  <React.Fragment key={p.name}>
                    <tr>
                      <td className="mono">{p.name}</td>
                      <td className="mono" style={{ color: 'var(--cyan)' }}>{p.value}</td>
                      <td className="mono">{p.range}</td><td>{p.desc}</td>
                      <td>{p.status === 'pending' ? <Pill st="warn" text="待下发" /> : <Pill st="ok" text="已生效" />}</td>
                      <td>
                        <span style={{ color: 'var(--blue)', cursor: 'pointer' }} onClick={() => editParam(p)}>修改</span>
                        {' · '}<span style={{ color: 'var(--blue)', cursor: 'pointer' }} onClick={() => showHistory(p.name)}>历史</span>
                      </td>
                    </tr>
                    {histFor === p.name && (
                      <tr>
                        <td colSpan={6} style={{ background: '#f7f9fd' }}>
                          {history.length
                            ? history.map((h, i) => (
                              <div key={i} style={{ fontSize: 12, color: 'var(--text-2)', padding: '2px 0' }}>
                                {new Date(h.changedAt).toLocaleString('zh-CN', { hour12: false })} · {h.oldValue} → <b style={{ color: 'var(--cyan)' }}>{h.newValue}</b>
                              </div>
                            ))
                            : <span style={{ fontSize: 12, color: 'var(--text-3)' }}>暂无变更历史</span>}
                        </td>
                      </tr>
                    )}
                  </React.Fragment>
                ))}
              {!c.params.length && <tr><td colSpan={6}><div className="empty" style={{ padding: '32px 0' }}>暂无参数</div></td></tr>}
              </tbody>
          </table>
        </div>
      )}

      {tab === 'report' && (
        <div className="report-grid">
          {reports.map(r => (
            <div className="report-card" key={r.title}>
              <div className="r-ico">{r.ico}</div>
              <h4>{r.title}</h4><p>{r.desc}</p>
              <div className="r-foot"><span>{r.date}</span>
                <a className="card-link" href={`/api/clusters/${c.id}/reports/${r.id}/download`}>下载（{r.size}）</a>
              </div>
            </div>
          ))}
        </div>
      )}
      {ops.view}
    </>
  );
}
