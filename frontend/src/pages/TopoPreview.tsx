import { useState } from 'react';
import { useBreadcrumb } from '../App';
import TopoV2, { TopoV2Legend } from '../components/TopoV2';
import { TOPO_SCENARIOS } from '../lib/topoMock';

/* 拓扑 v2 原型审核页（#/topo-demo）：纯前端 mock，验证「纵向数据流 + 横向复制 +
 * 物理分区 + OB 双视图 + 实例高亮」方案；审核通过后再做 v2 后端实施与正式改造。 */

type PhysDim = 'off' | 'az' | 'host_cluster';

const PHYS_OPTS: { v: PhysDim; label: string }[] = [
  { v: 'off', label: '关闭' },
  { v: 'az', label: '按 AZ' },
  { v: 'host_cluster', label: '按主机集群' },
];

export default function TopoPreview() {
  useBreadcrumb([{ label: '首页' }, { label: '拓扑原型审核' }]);
  const [skey, setSkey] = useState('ob');
  const [dim, setDim] = useState<PhysDim>('az');
  const [obView, setObView] = useState<'cluster' | 'tenant'>('cluster');
  const [hl, setHl] = useState<string>('obs-z2-1');

  const sc = TOPO_SCENARIOS.find(s => s.key === skey)!;
  const view = sc.hasTenantView ? obView : 'cluster';

  const segBtn = (active: boolean) => ({
    padding: '5px 12px', fontSize: 12, cursor: 'pointer', borderRadius: 6, border: '1px solid ' + (active ? 'var(--blue)' : '#d5deeb'),
    background: active ? 'var(--blue-bg)' : '#fff', color: active ? 'var(--blue)' : '#5a6779', fontWeight: active ? 600 : 400,
  });

  return (
    <>
      <div className="page-title">集群拓扑 · v2 原型</div>
      <div className="page-desc">
        通用引擎（一套渲染代码 + 类型配置）：纵向数据流分层（实线箭头）、横向复制（虚线箭头 + 延迟）、
        背景物理分区（AZ / 主机集群）、OB 租户视图（unit 落位点线）、实例高亮（关联链常亮）。mock 数据对齐元数据域 v2 表设计。
      </div>

      <div className="card" style={{ marginBottom: 14 }}>
        <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap', alignItems: 'center', padding: '4px 0' }}>
          {TOPO_SCENARIOS.map(s => (
            <button key={s.key} style={segBtn(s.key === skey)} onClick={() => { setSkey(s.key); setHl(''); }}>{s.dbType}</button>
          ))}
          <span style={{ width: 1, height: 20, background: '#e3e9f2' }} />
          <span style={{ fontSize: 12, color: '#8a97ad' }}>物理分区：</span>
          {PHYS_OPTS.map(o => (
            <button key={o.v} style={segBtn(dim === o.v)} onClick={() => setDim(o.v)}>{o.label}</button>
          ))}
          {sc.hasTenantView && (
            <>
              <span style={{ width: 1, height: 20, background: '#e3e9f2' }} />
              <span style={{ fontSize: 12, color: '#8a97ad' }}>视图：</span>
              <button style={segBtn(obView === 'cluster')} onClick={() => setObView('cluster')}>集群视图</button>
              <button style={segBtn(obView === 'tenant')} onClick={() => setObView('tenant')}>租户视图</button>
            </>
          )}
          <span style={{ width: 1, height: 20, background: '#e3e9f2' }} />
          <span style={{ fontSize: 12, color: '#8a97ad' }}>高亮实例：</span>
          <select value={hl} onChange={e => setHl(e.target.value)}
            style={{ fontSize: 12, padding: '4px 8px', borderRadius: 6, border: '1px solid #d5deeb', color: '#253248' }}>
            <option value="">（无）</option>
            {sc.components.map(c => <option key={c.id} value={c.id}>{c.name} · {c.kind}{c.role ? ` / ${c.role}` : ''}</option>)}
          </select>
        </div>
      </div>

      <div className="card">
        <div className="card-head">
          <div className="card-title">{sc.name} <span style={{ fontWeight: 400, color: '#8a97ad', fontSize: 12 }}>· {sc.desc}</span></div>
          {sc.hasTenantView && <div className="card-sub">{view === 'cluster' ? '集群视图：obproxy → 租户 → observer（Paxos）' : '租户视图：租户规格 + unit 落位'}</div>}
        </div>
        <TopoV2 scenario={sc} physicalDim={dim} view={view} highlightId={hl || undefined} onSelect={setHl} />
        <TopoV2Legend />
      </div>

      <div className="card" style={{ marginTop: 14 }}>
        <div className="card-title" style={{ marginBottom: 8 }}>审核要点</div>
        <ul style={{ fontSize: 12.5, color: '#5a6779', lineHeight: 2, paddingLeft: 18 }}>
          <li><b>纵向箭头</b>（蓝实线折线）：proxy → 租户 → 存储的数据流（不画客户端节点）；成员级流量指向（如 MySQL 存储→haproxy）由 <code>traffic_upstream_id</code> 驱动</li>
          <li><b>横向箭头</b>（灰虚线折线桥）：主 → 备复制 + 延迟标签，同主多备分层避让；Redis/Mongo 分组内主备、PG 跨分组主备均支持；OB observer 为 Paxos 多主（对称虚线，复制字段置空）</li>
          <li><b>背景虚线框</b>：物理隔离（region · AZ / 主机集群），全局分栏、框间互不重叠，右上角可切换维度或关闭</li>
          <li><b>OB 租户视图</b>：租户大卡（模式/规格）+ 点线落位到 observer；从实例详情进入时 = 下拉/点击选中该实例 → 整图保持、目标脉冲高亮、关联链常亮、其余淡出</li>
          <li><b>数据模型</b>：mock 结构与元数据域 v2 表设计一一对应（db_component 双上游字段 / db_host 三级位置）——审核通过即按此实施后端与正式页面</li>
        </ul>
      </div>
    </>
  );
}
