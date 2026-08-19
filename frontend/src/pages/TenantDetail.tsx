import { useCallback, useEffect, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { Pill, TypeTag, Bar, Stat } from '../components/bits';
import { MonitorTab } from '../components/MonitorTab';
import { CLUSTERS, SESSIONS, SLOW_SQLS } from '../lib/mockData';
import { apiGet, apiPost, apiPut, withFallback } from '../lib/api';
import { useOpDialog, runOp } from '../components/opDialog';
import { Overlay } from '../components/dialogs';
import { toast } from '../lib/toast';
import { useBreadcrumb } from '../App';
import { openChatDrawer } from '../lib/chatDrawer';
import { IconRefresh, IconRobot } from '../components/icons';
import type { ObTenant, ObTenantDb, ParamItem } from '../lib/types';

interface SessionRow { id: number; user: string; host: string; db: string; cmd: string; time: string; state: string; lock: string; status: string }
interface SlowSqlRow { sql: string; db: string; time: string; rows: string; count: number }

/* OceanBase 租户详情：Unit 资源池 + 租户内 Database + 会话/慢SQL（apiserver 数据，mock 兜底） */
const TABS = [
  { id: 'overview', label: '租户概览' },
  { id: 'db', label: '数据库管理' },
  { id: 'monitor', label: '性能监控' },
  { id: 'session', label: '会话管理' },
  { id: 'sql', label: '慢 SQL 诊断' },
  { id: 'param', label: '租户参数' },
];

export default function TenantDetail() {
  const { cid, tid } = useParams();
  const navigate = useNavigate();
  const [c, setC] = useState(() => CLUSTERS.find(x => x.id === cid) || null);
  const [t, setT] = useState<ObTenant | null>(() => c?.tenants?.find(x => x.id === tid) || null);
  const [dbs, setDbs] = useState<ObTenantDb[]>(() => t?.databases || []);
  const [params, setParams] = useState<ParamItem[]>([]);
  const [sessions, setSessions] = useState<SessionRow[]>(SESSIONS);
  const [sqls, setSqls] = useState<SlowSqlRow[]>(SLOW_SQLS);
  const [tab, setTab] = useState('overview');
  const [diagSql, setDiagSql] = useState<number | null>(null);
  const [viewSql, setViewSql] = useState<SlowSqlRow | null>(null);

  const reload = useCallback(() => {
    if (!cid || !tid) return;
    const localCluster = CLUSTERS.find(x => x.id === cid);
    const localTenant = localCluster?.tenants?.find(x => x.id === tid) || null;
    withFallback(apiGet<{ tenant: ObTenant; databases: ObTenantDb[]; params: ParamItem[] }>(`/api/tenants/${tid}`), () => null)
      .then(d => {
        if (!d) return;
        setT(d.tenant);
        setDbs(d.databases || []);
        setParams(d.params || []);
      });
    if (localTenant) { setC(localCluster || null); setT(cur => cur || localTenant); setDbs(cur => cur.length ? cur : localTenant.databases); }
    withFallback(apiGet<SessionRow[]>(`/api/tenants/${tid}/sessions`), () => null)
      .then(d => { if (d?.length) setSessions(d); });
    withFallback(apiGet<SlowSqlRow[]>(`/api/tenants/${tid}/slow-sqls`), () => null)
      .then(d => { if (d?.length) setSqls(d); });
  }, [cid, tid]);

  useEffect(() => { reload(); }, [reload]);

  useBreadcrumb(
    c && t
      ? [{ label: '首页' }, { label: '集群', hash: '#/clusters' }, { label: c.name, hash: `#/cluster/${c.id}` }, { label: `租户 ${t.name}` }]
      : [{ label: '首页' }, { label: '集群管理' }],
  );

  if (!c || !t) { navigate('/clusters', { replace: true }); return null; }

  /* ---- 写操作（页内弹窗 + toast 反馈） ---- */
  const ops = useOpDialog();

  const resize = () => ops.prompt(`Unit 扩缩容（${t.name}）`, [
    { key: 'maxCpu', label: `Unit CPU 上限（当前 ${t.maxCpu}C）`, initial: String(t.maxCpu + 2), required: true, hint: '在线生效，按 Zone 同步下发到每个 Unit' },
  ], v => {
    const cpu = Number(v.maxCpu);
    if (!Number.isFinite(cpu) || cpu <= 0) { toast.error('CPU 上限需为正数'); return; }
    if (cpu === t.maxCpu) { toast.info('CPU 上限未变化'); return; }
    runOp(`${t.name} Unit CPU 已调整为 ${cpu}C`, () =>
      apiPost(`/api/tenants/${t.id}/resize`, { maxCpu: cpu }), reload);
  });

  const createDb = () => ops.prompt(`创建数据库（租户 ${t.name}）`, [
    { key: 'name', label: '数据库名称', placeholder: '如 marketing', required: true },
  ], v => runOp(`数据库 ${v.name} 已创建`, () =>
    apiPost(`/api/tenants/${t.id}/databases`, { name: v.name }), reload), '创建');

  const killSession = (s: SessionRow) => ops.confirm('Kill 会话', `确认 Kill 会话 ${s.id}（${s.user}）？该操作将终止其执行。`, () =>
    runOp(`会话 ${s.id} 已终止`, () => apiPost(`/api/tenants/${t.id}/sessions/${s.id}/kill`), reload),
    { okText: 'Kill', danger: true });

  const editParam = (p: ParamItem) => ops.prompt(`修改参数 ${p.name}`, [
    { key: 'value', label: `新值（范围 ${p.range === '—' ? '见说明' : p.range}）`, initial: p.value, required: true, hint: p.desc },
  ], v => {
    if (v.value === p.value) { toast.info('参数值未变化'); return; }
    runOp(`参数 ${p.name} 已修改，待工单下发`, () =>
      apiPut(`/api/tenants/${t.id}/params/${encodeURIComponent(p.name)}`, { value: v.value }), reload);
  });

  const diagnose = (i: number) => setDiagSql(i);

  return (
    <>
      <div className="detail-head">
        <div className="cluster-icon">🏢</div>
        <div>
          <div className="detail-title">
            {t.name} <TypeTag t="oceanbase" />
            <Pill st={t.status} />
            {t.kind === 'sys' && <span className="pill info"><i></i>系统租户</span>}
          </div>
          <div className="detail-sub">
            <span>{t.mode === 'mysql' ? 'MySQL' : 'Oracle'} 兼容模式</span>
            <span>PRIMARY_ZONE：{t.primaryZone}</span>
            <span>LOCALITY：{t.locality}</span>
            <span>unit_num：{t.unitNum}</span>
          </div>
        </div>
        <div className="detail-head-right">
          <button className="btn" onClick={reload}><IconRefresh size={13} /> 刷新</button>
          <button className="btn primary" onClick={() => openChatDrawer(`诊断 ${t.name}`)}><IconRobot size={14} /> AI 诊断</button>
        </div>
      </div>
      <div className="tabs">
        {TABS.map(x => (
          <div key={x.id} className={`tab ${tab === x.id ? 'active' : ''}`} onClick={() => setTab(x.id)}>{x.label}</div>
        ))}
      </div>

      {tab === 'overview' && (
        <>
          <div className="stat-row">
            <Stat num={`${t.usedCpu}/${t.maxCpu} C`} lbl="CPU（已用/Unit 上限）" cls={t.usedCpu / t.maxCpu > 0.85 ? 'warn' : 'ok'} />
            <Stat num={`${t.usedMemGb}/${t.maxMemGb} G`} lbl="内存（已用/Unit 上限）" cls={t.usedMemGb / t.maxMemGb > 0.85 ? 'warn' : 'ok'} />
            <Stat num={`${t.storageUsed} / ${t.storageTotal}`} lbl="存储（已用/总量）" />
            <Stat num={dbs.length} lbl="数据库数" />
            <Stat num={dbs.reduce((a, d) => a + d.conn, 0)} lbl="活跃连接" />
          </div>

          <div className="card" style={{ marginBottom: 14 }}>
            <div className="card-head">
              <div className="card-title"><span className="t-ico"></span>Unit 资源池分布（Paxos 副本按 Zone）</div>
              <button className="btn sm primary" onClick={resize}>⇕ Unit 扩缩容</button>
            </div>
            <table className="tbl">
              <thead><tr><th>Zone</th><th>OBServer</th><th>Unit CPU（已用/上限）</th><th>Unit 内存（已用/上限）</th><th>副本角色</th></tr></thead>
              <tbody>
                {t.units.map((u, i) => (
                  <tr key={i}>
                    <td>{u.zone}</td>
                    <td className="mono">
                      <Link to={`/instance/${c.id}/${c.instances.find(x => x.name === u.observer)?.id || ''}`} style={{ color: 'var(--blue)' }}>{u.observer}</Link>
                    </td>
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
                    <td>{u.zone === t.primaryZone ? <Pill st="info" text="LEADER 优先" /> : <Pill st="ok" text="FOLLOWER" />}</td>
                  </tr>
                ))}
              {!t.units.length && <tr><td colSpan={5}><div className="empty" style={{ padding: '32px 0' }}>暂无 Unit 资源池</div></td></tr>}
              </tbody>
            </table>
          </div>

          <div className="card">
            <div className="card-head">
              <div className="card-title"><span className="t-ico"></span>连接信息</div>
              <span className="card-sub">白名单外来源将被拒绝</span>
            </div>
            <div style={{ display: 'flex', gap: 16, alignItems: 'flex-start', padding: '4px 2px' }}>
              <div style={{ flex: 1 }}>
                <div style={{ fontSize: 11.5, color: 'var(--text-3)', marginBottom: 6 }}>连接示例</div>
                <div className="mono" style={{ background: '#f7f9fd', border: '1px solid #edf1f8', borderRadius: 8, padding: '10px 12px', fontFamily: 'Menlo, monospace', fontSize: 12 }}>{t.connHint}</div>
              </div>
              <div style={{ flex: 1 }}>
                <div style={{ fontSize: 11.5, color: 'var(--text-3)', marginBottom: 6 }}>连接白名单</div>
                <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
                  {t.whitelist.map(w => <span key={w} className="pill info"><i></i>{w}</span>)}
                </div>
              </div>
            </div>
          </div>
        </>
      )}

      {tab === 'db' && (
        <div className="card">
          <div className="card-head">
            <div className="card-title"><span className="t-ico"></span>数据库管理（租户 {t.name}）</div>
            <button className="btn sm primary" onClick={createDb}>＋ 创建数据库</button>
          </div>
          <table className="tbl">
            <thead><tr><th>数据库</th><th>表数量</th><th>大小</th><th>连接数</th><th>状态</th><th>操作</th></tr></thead>
            <tbody>
              {dbs.map(d => (
                <tr key={d.name}>
                  <td className="mono">{d.name}</td>
                  <td>{d.tables}</td>
                  <td>{d.size}</td>
                  <td>{d.conn}</td>
                  <td><Pill st={d.status} /></td>
                  <td><span style={{ color: 'var(--blue)', cursor: 'pointer' }}>会话</span> · <span style={{ color: 'var(--blue)', cursor: 'pointer' }}>备份</span></td>
                </tr>
              ))}
            {!dbs.length && <tr><td colSpan={6}><div className="empty" style={{ padding: '32px 0' }}>暂无数据库，点击右上角创建</div></td></tr>}
            </tbody>
          </table>
        </div>
      )}

      {tab === 'monitor' && <MonitorTab />}

      {tab === 'session' && (
        <div className="card">
          <div className="card-head">
            <div className="card-title"><span className="t-ico"></span>租户活跃会话</div>
            <div><span className="card-sub">共 {sessions.length} 个会话 · {sessions.filter(s => s.status !== 'ok').length} 个异常</span></div>
          </div>
          <table className="tbl">
            <thead><tr><th>ID</th><th>用户</th><th>来源</th><th>库</th><th>命令</th><th>时长</th><th>状态</th><th>锁信息</th><th>操作</th></tr></thead>
            <tbody>
              {sessions.map(s => (
                <tr key={s.id}>
                  <td className="mono">{s.id}</td><td className="mono">{s.user}</td><td className="mono">{s.host}</td>
                  <td>{s.db}</td><td>{s.cmd}</td>
                  <td style={{ color: s.status !== 'ok' ? 'var(--amber)' : 'inherit' }}>{s.time}</td>
                  <td>{s.state}</td>
                  <td>{s.lock === '—' ? '—' : <span style={{ color: 'var(--red)' }}>{s.lock}</span>}</td>
                  <td>{s.status !== 'ok'
                    ? <button className="btn sm danger" onClick={() => killSession(s)}>Kill</button>
                    : <span className="card-sub">—</span>}</td>
                </tr>
              ))}
            {!sessions.length && <tr><td colSpan={9}><div className="empty" style={{ padding: '32px 0' }}>暂无活跃会话</div></td></tr>}
            </tbody>
          </table>
        </div>
      )}

      {tab === 'sql' && (
        <div className="card">
          <div className="card-head">
            <div className="card-title"><span className="t-ico"></span>慢 SQL 诊断（租户 {t.name} · 近 24h）</div>
            <span className="card-sub">共 {sqls.length} 条待优化</span>
          </div>
          <table className="tbl">
            <thead><tr><th>SQL 指纹</th><th>库</th><th>平均耗时</th><th>扫描行数</th><th>执行次数</th><th>操作</th></tr></thead>
            <tbody>
              {sqls.map((s, i) => (
                <tr key={i}>
                  <td className="mono sql-cell" title="点击查看完整 SQL" onClick={() => setViewSql(s)} style={{ maxWidth: 380, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{s.sql}</td>
                  <td>{s.db}</td><td style={{ color: 'var(--amber)' }}>{s.time}</td><td>{s.rows}</td><td>{s.count}</td>
                  <td><span style={{ color: 'var(--blue)', cursor: 'pointer' }} onClick={() => diagnose(i)}>AI 诊断</span></td>
                </tr>
              ))}
            {!sqls.length && <tr><td colSpan={6}><div className="empty" style={{ padding: '32px 0' }}>暂无慢 SQL 记录</div></td></tr>}
            </tbody>
          </table>
          {diagSql != null && (
            <div className="advice">
              <h4 className="ai-advice-title"><IconRobot size={15} /> AI 优化建议</h4>
              <ul>
                <li><code>{sqls[diagSql].db}</code> 缺少合适索引，建议添加联合索引 <code>idx_status_uid (status, uid)</code>，预计扫描行数下降 <b style={{ color: 'var(--green)' }}>92%</b>；</li>
                <li>存在隐式类型转换导致索引失效，请核对字段类型与传参类型一致；</li>
                <li>该 SQL 日均执行 <b>{sqls[diagSql].count}</b> 次，优化后预计租户 CPU 下降约 <b style={{ color: 'var(--green)' }}>18%</b>（当前 {t.usedCpu}/{t.maxCpu}C，可暂缓 Unit 扩容）；</li>
                <li>租户 CPU 达阈值时，本条 SQL 优化优先级最高。</li>
              </ul>
            </div>
          )}
        </div>
      )}

      {tab === 'param' && (
        <div className="card">
          <div className="card-head">
            <div className="card-title"><span className="t-ico"></span>租户参数</div>
            <span className="card-sub">仅作用于本租户，与集群级参数相互独立</span>
          </div>
          <table className="tbl">
            <thead><tr><th>参数名</th><th>当前值</th><th>说明</th><th>状态</th><th>操作</th></tr></thead>
            <tbody>
              {params.map(p => (
                <tr key={p.name}>
                  <td className="mono">{p.name}</td>
                  <td className="mono" style={{ color: 'var(--cyan)' }}>{p.value}</td>
                  <td>{p.desc}</td>
                  <td>{p.status === 'pending' ? <Pill st="warn" text="待下发" /> : <Pill st="ok" text="已生效" />}</td>
                  <td><span style={{ color: 'var(--blue)', cursor: 'pointer' }} onClick={() => editParam(p)}>修改</span></td>
                </tr>
              ))}
              {!params.length && <tr><td colSpan={5}><div className="empty" style={{ padding: '40px 0' }}>暂无租户参数</div></td></tr>}
            </tbody>
          </table>
        </div>
      )}
      {viewSql && (
        <Overlay onClose={() => setViewSql(null)}>
          <div className="dap-head">SQL 指纹</div>
          <div className="dap-body">
            <pre className="sql-view">{viewSql.sql}</pre>
            <div className="sql-view-meta">库 {viewSql.db} · 平均耗时 {viewSql.time} · 扫描 {viewSql.rows} 行 · 日均 {viewSql.count} 次</div>
          </div>
          <div className="dap-foot">
            <button className="btn sm" onClick={() => setViewSql(null)}>关闭</button>
            <button className="btn sm primary" onClick={() => { navigator.clipboard?.writeText(viewSql.sql).then(() => toast.success('SQL 已复制')).catch(() => toast.error('复制失败')); }}>复制</button>
          </div>
        </Overlay>
      )}
      {ops.view}
    </>
  );
}
