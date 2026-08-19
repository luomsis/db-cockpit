import { useCallback, useEffect, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { Pill, TypeTag } from '../components/bits';
import { TopoSVG, TopoLegend } from '../components/TopoSVG';
import { MonitorTab } from '../components/MonitorTab';
import { CLUSTERS, TYPE_ICON, INSTANCE_USERS, SESSIONS, TRANSACTIONS, SLOW_SQLS } from '../lib/mockData';
import { apiGet, apiPost, withFallback } from '../lib/api';
import { useOpDialog, runOp } from '../components/opDialog';
import { Overlay } from '../components/dialogs';
import { toast } from '../lib/toast';
import { useBreadcrumb } from '../App';
import { openChatDrawer } from '../lib/chatDrawer';
import { IconRefresh, IconRobot } from '../components/icons';
import type { Cluster, Instance } from '../lib/types';

interface UserRow { user: string; host: string; priv: string; lastLogin: string; status: string }
interface SessionRow { id: number; user: string; host: string; db: string; cmd: string; time: string; state: string; lock: string; status: string }
interface TrxRow { id: string; session: number; user: string; dur: string; undo: string; lockRows: string; waiting: string; sql: string; status: string }
interface SlowSqlRow { sql: string; db: string; time: string; rows: string; count: number }

const INST_TABS = [
  { id: 'topo', label: '拓扑图' },
  { id: 'user', label: '用户管理' },
  { id: 'monitor', label: '性能监控' },
  { id: 'sql', label: 'SQL 诊断' },
  { id: 'trx', label: '事务诊断' },
  { id: 'session', label: '会话管理' },
];

export default function InstanceDetail() {
  const { cid, iid } = useParams();
  const navigate = useNavigate();
  const [tab, setTab] = useState('topo');
  const [diagSql, setDiagSql] = useState<number | null>(null);
  const [viewSql, setViewSql] = useState<SlowSqlRow | null>(null);
  const [diagAdvice, setDiagAdvice] = useState<string[]>([]);

  /* 实例 / 集群：apiserver 优先，mock 兜底 */
  const [c, setC] = useState<Cluster | null>(() => CLUSTERS.find(x => x.id === cid) || null);
  const [inst, setInst] = useState<Instance | null>(() => c?.instances.find(i => i.id === iid) || null);
  const [users, setUsers] = useState<UserRow[]>(INSTANCE_USERS);
  const [sessions, setSessions] = useState<SessionRow[]>(SESSIONS);
  const [trxs, setTrxs] = useState<TrxRow[]>(TRANSACTIONS);
  const [sqls, setSqls] = useState<SlowSqlRow[]>(SLOW_SQLS);

  const reload = useCallback(() => {
    if (!cid || !iid) return;
    withFallback(apiGet<{ cluster: Cluster; instance: Instance }>(`/api/clusters/${cid}/instances/${iid}`), () => null)
      .then(d => { if (d) { setC(d.cluster); setInst(d.instance); } });
    withFallback(apiGet<UserRow[]>(`/api/clusters/${cid}/instances/${iid}/users`), () => null)
      .then(d => { if (d?.length) setUsers(d); });
    withFallback(apiGet<SessionRow[]>(`/api/clusters/${cid}/instances/${iid}/sessions`), () => null)
      .then(d => { if (d?.length) setSessions(d); else if (d) setSessions([]); });
    withFallback(apiGet<TrxRow[]>(`/api/clusters/${cid}/instances/${iid}/transactions`), () => null)
      .then(d => { if (d?.length) setTrxs(d); });
    withFallback(apiGet<SlowSqlRow[]>(`/api/clusters/${cid}/instances/${iid}/slow-sqls`), () => null)
      .then(d => { if (d?.length) setSqls(d); });
  }, [cid, iid]);

  useEffect(() => { reload(); }, [reload]);

  useBreadcrumb(
    c && inst
      ? [{ label: '首页' }, { label: '集群', hash: '#/clusters' }, { label: c.name, hash: `#/cluster/${c.id}` }, { label: inst.name }]
      : [{ label: '首页' }, { label: '集群管理' }],
  );

  if (!c || !inst) { navigate('/clusters', { replace: true }); return null; }
  const base = `/api/clusters/${c.id}/instances/${inst.id}`;

  /* ---- 写操作（页内弹窗 + toast 反馈） ---- */
  const ops = useOpDialog();

  const createUser = () => ops.prompt('创建账号', [
    { key: 'user', label: '账号名', placeholder: '如 app_rw / trade_rw@tenant', required: true },
    { key: 'host', label: '允许主机', initial: '%' },
    { key: 'priv', label: '权限', initial: 'SELECT' },
  ], v => runOp(`账号 ${v.user} 已创建`, () =>
    apiPost(`${base}/users`, { user: v.user, host: v.host || '%', priv: v.priv || 'SELECT' }), reload), '创建');

  const grantUser = (u: UserRow) => ops.prompt(`调整权限（${u.user}）`, [
    { key: 'priv', label: '权限', initial: u.priv, required: true },
  ], v => {
    if (v.priv === u.priv) { toast.info('权限未变化'); return; }
    runOp(`已更新 ${u.user} 的权限`, () =>
      apiPost(`${base}/users/${encodeURIComponent(u.user)}/grant`, { priv: v.priv }), reload);
  });

  const resetPassword = (u: UserRow) => ops.confirm('重置密码', `确认重置 ${u.user} 的密码？新密码将通过安全渠道下发。`, () =>
    runOp(`已提交 ${u.user} 的密码重置（见审计日志）`, () =>
      apiPost(`${base}/users/${encodeURIComponent(u.user)}/reset-password`)), { okText: '重置' });

  const lockUser = (u: UserRow) => {
    const locking = u.status !== 'err';
    return ops.confirm(locking ? '锁定账号' : '解锁账号', locking
      ? `确认锁定账号 ${u.user}？锁定后该账号无法登录。`
      : `确认解锁账号 ${u.user}？`, () =>
      runOp(locking ? `已锁定 ${u.user}` : `已解锁 ${u.user}`, () =>
        apiPost(`${base}/users/${encodeURIComponent(u.user)}/lock`), reload),
      { okText: locking ? '锁定' : '解锁', danger: locking });
  };

  const killSession = (s: SessionRow) => ops.confirm('Kill 会话', `确认 Kill 会话 ${s.id}（${s.user}）？该操作将终止其执行。`, () =>
    runOp(`会话 ${s.id} 已终止`, () => apiPost(`${base}/sessions/${s.id}/kill`), reload),
    { okText: 'Kill', danger: true });

  const diagnose = async (i: number) => {
    setDiagSql(i);
    const s = sqls[i];
    const advice = await withFallback(
      apiPost<{ suggestions: string[] }>('/api/diagnosis/sql', { sql: s.sql, db: s.db, count: s.count }),
      () => ({ suggestions: [] as string[] }),
    );
    setDiagAdvice(advice.suggestions || []);
  };

  return (
    <>
      <div className="detail-head">
        <div className="cluster-icon">{TYPE_ICON[c.type]}</div>
        <div>
          <div className="detail-title">{inst.name} <Pill st={inst.status} /> <TypeTag t={c.type} /></div>
          <div className="detail-sub">
            <span className="mono" style={{ fontFamily: 'Menlo,monospace' }}>{inst.ip}:{inst.port}</span>
            <span>角色：{inst.role}</span><span>版本：{inst.ver}</span><span>连接数：{inst.conn}</span>
          </div>
        </div>
        <div className="detail-head-right">
          <button className="btn" onClick={reload}><IconRefresh size={13} /> 刷新</button>
          <button className="btn primary" onClick={() => openChatDrawer(`诊断 ${inst.name}`)}><IconRobot size={14} /> AI 诊断</button>
        </div>
      </div>
      <div className="tabs">
        {INST_TABS.map(t => (
          <div key={t.id} className={`tab ${tab === t.id ? 'active' : ''}`} onClick={() => setTab(t.id)}>{t.label}</div>
        ))}
      </div>

      {tab === 'topo' && <div className="card"><div className="topo-wrap"><TopoSVG cluster={c} /></div><TopoLegend cluster={c} /></div>}

      {tab === 'user' && (
        <div className="card">
          <div className="card-head">
            <div className="card-title"><span className="t-ico"></span>数据库账号</div>
            <button className="btn sm primary" onClick={createUser}>＋ 创建账号</button>
          </div>
          <table className="tbl">
            <thead><tr><th>用户名</th><th>允许主机</th><th>权限</th><th>最近登录</th><th>状态</th><th>操作</th></tr></thead>
            <tbody>
              {users.map(u => (
                <tr key={u.user}>
                  <td className="mono">{u.user}</td><td className="mono">{u.host}</td><td className="mono">{u.priv}</td>
                  <td>{u.lastLogin}</td>
                  <td>{u.status === 'ok' ? <Pill st="ok" /> : u.status === 'warn' ? <Pill st="warn" text="长期未活跃" /> : <Pill st="err" text={u.status === 'err' ? '已锁定 / 建议回收' : u.status} />}</td>
                  <td>
                    <span style={{ color: 'var(--blue)', cursor: 'pointer' }} onClick={() => grantUser(u)}>授权</span>
                    {' · '}<span style={{ color: 'var(--blue)', cursor: 'pointer' }} onClick={() => resetPassword(u)}>重置密码</span>
                    {' · '}<span style={{ color: 'var(--blue)', cursor: 'pointer' }} onClick={() => lockUser(u)}>{u.status === 'err' ? '解锁' : '锁定'}</span>
                  </td>
                </tr>
              ))}
            {!users.length && <tr><td colSpan={6}><div className="empty" style={{ padding: '32px 0' }}>暂无账号，点击右上角创建</div></td></tr>}
            </tbody>
          </table>
        </div>
      )}

      {tab === 'monitor' && <MonitorTab />}

      {tab === 'sql' && (
        <div className="card">
          <div className="card-head">
            <div className="card-title"><span className="t-ico"></span>慢 SQL 诊断（近 24h）</div>
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
                {(diagAdvice.length ? diagAdvice : [
                  `对 ${sqls[diagSql].db} 表缺少合适索引，建议添加联合索引，预计扫描行数下降 92%；`,
                  '存在隐式类型转换导致索引失效，请核对字段类型与传参类型一致；',
                  `该 SQL 日均执行 ${sqls[diagSql].count} 次，优化后预计集群 CPU 下降约 8%；`,
                  '可一键生成索引变更工单，由参数管理通道灰度下发。',
                ]).map((t, i) => <li key={i} dangerouslySetInnerHTML={{ __html: t.replace(/idx_\S+/, m => `<code>${m}</code>`) }} />)}
              </ul>
            </div>
          )}
        </div>
      )}

      {tab === 'trx' && (
        <div className="card">
          <div className="card-head">
            <div className="card-title"><span className="t-ico"></span>长事务 / 未提交事务</div>
            <span className="card-sub">阈值 &gt; 30s</span>
          </div>
          <table className="tbl">
            <thead><tr><th>事务 ID</th><th>会话</th><th>用户</th><th>持续时间</th><th>Undo 大小</th><th>锁行数</th><th>等待锁</th><th>当前 SQL</th><th>状态</th></tr></thead>
            <tbody>
              {trxs.map(t => (
                <tr key={t.id}>
                  <td className="mono">{t.id}</td><td>{t.session}</td><td className="mono">{t.user}</td>
                  <td style={{ color: t.status === 'err' ? 'var(--red)' : 'inherit' }}>{t.dur}</td>
                  <td>{t.undo}</td><td>{t.lockRows}</td><td>{t.waiting}</td>
                  <td className="mono" style={{ maxWidth: 220, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{t.sql}</td>
                  <td><Pill st={t.status} /></td>
                </tr>
              ))}
            {!trxs.length && <tr><td colSpan={9}><div className="empty" style={{ padding: '32px 0' }}>暂无长事务</div></td></tr>}
            </tbody>
          </table>
          <div className="advice">
            <h4>🔗 阻塞链分析</h4>
            <ul>
              <li><code>TRX-998231</code>（会话 88231）持有 <code>stock_record</code> 行锁 1m28s，阻塞 3 个后续会话；</li>
              <li>根因：应用侧开启事务后未及时提交，建议联系 <code>app_rw</code> 业务方确认；</li>
              <li>若持续超过 5 分钟，可执行「Kill 会话」一键止血（会话管理中操作）。</li>
            </ul>
          </div>
        </div>
      )}

      {tab === 'session' && (
        <div className="card">
          <div className="card-head">
            <div className="card-title"><span className="t-ico"></span>活跃会话</div>
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
              {!sessions.length && <tr><td colSpan={9}><div className="empty" style={{ padding: '40px 0' }}>暂无活跃会话</div></td></tr>}
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
