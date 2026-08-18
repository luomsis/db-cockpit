import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { Pill, TypeTag, Bar, Stat } from '../components/bits';
import { TopoSVG, TopoLegend } from '../components/TopoSVG';
import { MonitorTab } from '../components/MonitorTab';
import { TYPE_ICON, REPORTS } from '../lib/mockData';
import { apiGet, apiPost, apiPut, withFallback } from '../lib/api';
import { openChatDrawer } from '../lib/chatDrawer';
import type { Cluster, ObTenant, ParamItem } from '../lib/types';
import { IconRefresh, IconBolt, IconRobot } from '../components/icons';

interface ReportRow { id: number; ico: string; title: string; desc: string; date: string; size: string }

/* OceanBase 集群详情：Zone/OBServer + 租户体系 维度 */
const TABS = [
  { id: 'overview', label: '集群概览' },
  { id: 'tenant', label: '租户管理' },
  { id: 'monitor', label: '性能监控' },
  { id: 'param', label: '参数管理' },
  { id: 'report', label: '性能报告' },
];

export default function ClusterDetailOb({ cluster: c, reload }: { cluster: Cluster; reload: () => void }) {
  const [tab, setTab] = useState('overview');
  const [reports, setReports] = useState<ReportRow[]>([]);
  const [histFor, setHistFor] = useState<string | null>(null);
  const [history, setHistory] = useState<{ oldValue: string; newValue: string; changedAt: number }[]>([]);
  const tenants = c.tenants || [];
  const usedCpu = tenants.reduce((a, t) => a + t.usedCpu, 0);
  const maxCpu = tenants.reduce((a, t) => a + t.maxCpu, 0);
  const usedMem = tenants.reduce((a, t) => a + t.usedMemGb, 0);
  const maxMem = tenants.reduce((a, t) => a + t.maxMemGb, 0);

  useEffect(() => {
    let alive = true;
    withFallback(apiGet<ReportRow[]>(`/api/clusters/${c.id}/reports`), () => REPORTS.map((r, i) => ({ id: i, ...r })))
      .then(d => { if (alive && d?.length) setReports(d); });
    return () => { alive = false; };
  }, [c.id]);

  const createTenant = async () => {
    const name = window.prompt('租户名称');
    if (!name?.trim()) return;
    const cpu = Number(window.prompt('Unit CPU 上限（C）', '4') || '4');
    await apiPost(`/api/clusters/${c.id}/tenants`, { name: name.trim(), maxCpu: cpu || 4 }).catch(() => { });
    reload();
  };
  const resizeTenant = async (t: ObTenant) => {
    const cpu = window.prompt(`扩缩容 ${t.name}：Unit CPU 上限（当前 ${t.maxCpu}C）`, String(t.maxCpu + 2));
    if (cpu === null) return;
    const v = Number(cpu);
    if (!Number.isFinite(v) || v <= 0) return;
    if (!window.confirm(`确认将 ${t.name} Unit CPU 调整为 ${v}C？`)) return;
    await apiPost(`/api/tenants/${t.id}/resize`, { maxCpu: v }).catch(() => { });
    reload();
  };
  const editParam = async (p: ParamItem) => {
    const value = window.prompt(`修改参数 ${p.name}（范围 ${p.range}）`, p.value);
    if (value === null || value === p.value) return;
    await apiPut(`/api/clusters/${c.id}/params/${encodeURIComponent(p.name)}`, { value }).catch(() => { });
    reload();
  };
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
          <div className="detail-sub"><span>{c.version}</span><span>{c.mode}</span><span>{c.desc}</span></div>
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
            <Stat num={(c.zones || []).length} lbl="Zone 数" cls="ok" />
            <Stat num={c.instances.length} lbl="OBServer 数" />
            <Stat num={tenants.filter(t => t.kind === 'user').length} lbl="用户租户数" />
            <Stat num={`${usedCpu.toFixed(1)}/${maxCpu} C`} lbl="已分配 / 总 CPU" cls={usedCpu / Math.max(1, maxCpu) > 0.8 ? 'warn' : ''} />
            <Stat num={`${usedMem}/${maxMem} G`} lbl="已分配 / 总内存" cls={usedMem / Math.max(1, maxMem) > 0.8 ? 'warn' : ''} />
          </div>
          <div className="card" style={{ marginBottom: 14 }}>
            <div className="topo-wrap"><TopoSVG cluster={c} /></div>
            <TopoLegend cluster={c} />
          </div>
          <div className="card">
            <div className="card-head">
              <div className="card-title"><span className="t-ico"></span>OBServer 健康摘要</div>
              <span className="card-sub">合并状态：IDLE（上次每日合并 04:30 完成）</span>
            </div>
            <table className="tbl">
              <thead><tr><th>OBServer</th><th>Zone</th><th>地址</th><th>版本</th><th>CPU</th><th>内存</th><th>活跃连接</th><th>状态</th></tr></thead>
              <tbody>
                {c.instances.map(i => (
                  <tr key={i.id}>
                    <td className="mono"><Link to={`/instance/${c.id}/${i.id}`} style={{ color: 'var(--blue)' }}>{i.name}</Link></td>
                    <td>{i.zone}</td>
                    <td className="mono">{i.ip}:{i.port}</td>
                    <td>{i.ver}</td>
                    <td><Bar value={i.cpu} hot={i.cpu > 85} /></td>
                    <td><Bar value={i.mem} hot={i.mem > 85} /></td>
                    <td>{i.conn}</td><td><Pill st={i.status} /></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}

      {tab === 'tenant' && (
        <>
          <div className="card">
            <div className="card-head">
              <div className="card-title"><span className="t-ico"></span>租户管理（资源池与隔离的单位）</div>
              <button className="btn sm primary" onClick={createTenant}>＋ 创建租户</button>
            </div>
            <table className="tbl">
              <thead>
                <tr>
                  <th>租户</th><th>类型 / 模式</th><th>PRIMARY_ZONE</th><th>Unit 数</th>
                  <th>CPU（已用/上限）</th><th>内存（已用/上限）</th><th>存储</th><th>状态</th><th>操作</th>
                </tr>
              </thead>
              <tbody>
                {tenants.map(t => (
                  <tr key={t.id}>
                    <td className="mono">
                      <Link to={`/tenant/${c.id}/${t.id}`} style={{ color: 'var(--blue)' }}>{t.name}</Link>
                    </td>
                    <td>{t.kind === 'sys' ? '系统租户' : '用户租户'} · {t.mode === 'mysql' ? 'MySQL' : 'Oracle'} 模式</td>
                    <td className="mono">{t.primaryZone}</td>
                    <td>{t.unitNum} × {t.units.length}</td>
                    <td>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                        <Bar value={Math.round((t.usedCpu / t.maxCpu) * 100)} hot={t.usedCpu / t.maxCpu > 0.85} />
                        <span style={{ fontSize: 11.5, color: 'var(--text-2)' }}>{t.usedCpu}/{t.maxCpu}C</span>
                      </div>
                    </td>
                    <td>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                        <Bar value={Math.round((t.usedMemGb / t.maxMemGb) * 100)} hot={t.usedMemGb / t.maxMemGb > 0.85} />
                        <span style={{ fontSize: 11.5, color: 'var(--text-2)' }}>{t.usedMemGb}/{t.maxMemGb}G</span>
                      </div>
                    </td>
                    <td>{t.storageUsed} / {t.storageTotal}</td>
                    <td><Pill st={t.status} /></td>
                    <td>
                      <Link to={`/tenant/${c.id}/${t.id}`} style={{ color: 'var(--blue)' }}>详情</Link>
                      {' · '}<span style={{ color: 'var(--blue)', cursor: 'pointer' }} onClick={() => resizeTenant(t)}>扩缩容</span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            <div className="advice">
              <h4>🌊 租户资源提示</h4>
              <ul>
                <li><code>trade_tenant</code> CPU 已用 13.1/14C（94%）：建议 <b>Unit 扩容</b>（14C→16C）或新增 Unit（unit_num 1→2，需 observer 有余量）；</li>
                <li><code>pay_tenant</code> 内存水位 91%：优先排查内存泄漏 / 大事务，暂缓扩容观察一周；</li>
                <li>租户间资源通过 Unit 物理隔离，扩容不影响其他租户（无需停机，在线生效）。</li>
              </ul>
            </div>
          </div>

          {/* Unit 分布一览 */}
          <div className="card" style={{ marginTop: 14 }}>
            <div className="card-head">
              <div className="card-title"><span className="t-ico"></span>Unit 资源池分布（Zone × OBServer）</div>
              <span className="card-sub">Paxos 副本按 Zone 分布，Unit 可在线迁移均衡</span>
            </div>
            <table className="tbl">
              <thead><tr><th>租户</th><th>Zone</th><th>OBServer</th><th>Unit CPU（已用/上限）</th><th>Unit 内存（已用/上限）</th></tr></thead>
              <tbody>
                {tenants.flatMap(t => t.units.map((u, i) => (
                  <tr key={`${t.id}-${i}`}>
                    <td className="mono">{i === 0 ? t.name : ''}</td>
                    <td>{u.zone}</td>
                    <td className="mono">{u.observer}</td>
                    <td>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                        <Bar value={Math.round((u.usedCpu / u.maxCpu) * 100)} hot={u.usedCpu / u.maxCpu > 0.85} />
                        <span style={{ fontSize: 11.5, color: 'var(--text-2)' }}>{u.usedCpu}/{u.maxCpu}C</span>
                      </div>
                    </td>
                    <td>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                        <Bar value={Math.round((u.usedMemGb / u.maxMemGb) * 100)} hot={u.usedMemGb / u.maxMemGb > 0.85} />
                        <span style={{ fontSize: 11.5, color: 'var(--text-2)' }}>{u.usedMemGb}/{u.maxMemGb}G</span>
                      </div>
                    </td>
                  </tr>
                )))}
              </tbody>
            </table>
          </div>
        </>
      )}

      {tab === 'monitor' && <MonitorTab />}

      {tab === 'param' && (
        <div className="card">
          <div className="card-head">
            <div className="card-title"><span className="t-ico"></span>参数管理</div>
            <span className="card-sub">集群级参数（租户级参数请在租户详情页调整）</span>
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
    </>
  );
}
