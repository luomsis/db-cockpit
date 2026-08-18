import { Link } from 'react-router-dom';
import type { Cluster, Instance } from '../lib/types';

/* PG 集群拓扑：应用层 → 主备流复制（沿用原型 SVG 布局） */
function PgTopoSVG({ cluster }: { cluster: Cluster }) {
  const W = 760, H = 300;
  const insts = cluster.instances;
  const masters = insts.filter(i => /主|Primary/i.test(i.role));
  const slaves = insts.filter(i => !/主|Primary/i.test(i.role));
  const statusColor: Record<string, string> = { ok: '#00b365', warn: '#ff9500', err: '#f53f3f' };
  const masterY = 90, slaveY = 220;
  const mNodes = masters.map((m, i) => ({ ...m, x: W / 2 - ((masters.length - 1) * 130) / 2 + i * 130, y: masterY }));
  const sNodes = slaves.map((s, i) => ({ ...s, x: W / 2 - ((slaves.length - 1) * 120) / 2 + i * 120, y: slaveY }));
  const appNode = { x: W / 2, y: 28 };

  const node = (n: Instance & { x: number; y: number }, isMaster: boolean) => (
    <g key={n.id}>
      <circle className="halo" cx={n.x} cy={n.y} r={30} fill={statusColor[n.status]} opacity={0.09} />
      <circle className="main" cx={n.x} cy={n.y} r={22} fill="#ffffff" stroke={statusColor[n.status]} strokeWidth={2} />
      <text x={n.x} y={n.y + 4} textAnchor="middle" fontSize={13} fill="#1d2b45">{isMaster ? '主' : '备'}</text>
      <circle cx={n.x + 16} cy={n.y - 16} r={4} fill={statusColor[n.status]} />
      <text x={n.x} y={n.y + 42} textAnchor="middle" fontSize={10.5} fill="#4e5d78">{n.name}</text>
      <text x={n.x} y={n.y + 56} textAnchor="middle" fontSize={9.5} fill="#8a97ad">{n.role} · {n.ip}</text>
    </g>
  );

  return (
    <svg viewBox={`0 0 ${W} ${H}`} width="100%" height={H}>
      <defs>
        <linearGradient id="lgEdge" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" stopColor="#006aff" /><stop offset="1" stopColor="#00a3e0" />
        </linearGradient>
      </defs>
      <g>
        <rect x={appNode.x - 60} y={appNode.y - 16} width={120} height={32} rx={8} fill="#e8f2ff" stroke="rgba(0,106,255,0.55)" />
        <text x={appNode.x} y={appNode.y + 4} textAnchor="middle" fontSize={12} fill="#006aff" fontWeight={600}>应用接入层</text>
      </g>
      {mNodes.map(m => (
        <line key={`e-${m.id}`} x1={appNode.x} y1={appNode.y + 16} x2={m.x} y2={m.y - 26} stroke="url(#lgEdge)" strokeWidth={1.6} strokeDasharray="5 4" opacity={0.75} />
      ))}
      {sNodes.map((s, i) => {
        const m = mNodes[i % Math.max(1, mNodes.length)] || mNodes[0];
        return m ? <line key={`s-${s.id}`} x1={m.x} y1={m.y + 26} x2={s.x} y2={s.y - 26} stroke="rgba(0,106,255,0.35)" strokeWidth={1.4} /> : null;
      })}
      {mNodes.map(m => (
        <Link key={`n-${m.id}`} to={`/instance/${cluster.id}/${m.id}`} className="topo-node" style={{ cursor: 'pointer' }}>{node(m, true)}</Link>
      ))}
      {sNodes.map(s => (
        <Link key={`n-${s.id}`} to={`/instance/${cluster.id}/${s.id}`} className="topo-node" style={{ cursor: 'pointer' }}>{node(s, false)}</Link>
      ))}
    </svg>
  );
}

/* OceanBase 集群拓扑：Zone 分组（每列一个 Zone，列内 OBServer 卡片），Zone 间 Paxos */
function ObTopoSVG({ cluster }: { cluster: Cluster }) {
  const W = 760, H = 310;
  const zones = cluster.zones && cluster.zones.length ? cluster.zones : Array.from(new Set(cluster.instances.map(i => i.zone || 'ZONE1')));
  const colW = Math.min(220, (W - 40) / zones.length - 14);
  const gap = (W - 40 - colW * zones.length) / Math.max(1, zones.length - 1);
  const statusColor: Record<string, string> = { ok: '#00b365', warn: '#ff9500', err: '#f53f3f' };

  return (
    <svg viewBox={`0 0 ${W} ${H}`} width="100%" height={H}>
      <defs>
        <linearGradient id="obZone" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" stopColor="#0052D9" /><stop offset="1" stopColor="#006aff" />
        </linearGradient>
      </defs>
      {/* Zone 间 Paxos/均衡连线 */}
      {zones.slice(0, -1).map((_, i) => {
        const x1 = 20 + i * (colW + gap) + colW, x2 = x1 + gap;
        return <line key={`px-${i}`} x1={x1} y1={170} x2={x2} y2={170} stroke="rgba(0,106,255,0.35)" strokeWidth={1.4} strokeDasharray="4 3" />;
      })}
      {zones.map((z, zi) => {
        const x = 20 + zi * (colW + gap);
        const observers = cluster.instances.filter(i => (i.zone || '') === z);
        return (
          <g key={z}>
            {/* Zone 标题条 */}
            <rect x={x} y={26} width={colW} height={30} rx={6} fill="#e6efff" stroke="#b8d4ff" />
            <text x={x + colW / 2} y={45} textAnchor="middle" fontSize={11.5} fill="#0052d9" fontWeight={700}>{z} · 可用区{zi + 1}</text>
            {observers.map((o, oi) => {
              const y = 72 + oi * 92;
              return (
                <Link key={o.id} to={`/instance/${cluster.id}/${o.id}`} className="topo-node">
                  <rect x={x} y={y} width={colW} height={78} rx={8} fill="#ffffff" stroke="#c9d4e8" strokeWidth={1} />
                  <circle cx={x + 14} cy={y + 15} r={4} fill={statusColor[o.status] || '#8a97ad'} />
                  <text x={x + 26} y={y + 19} fontSize={10.5} fill="#1d2b45" fontWeight={600}>{o.name}</text>
                  <text x={x + 12} y={y + 38} fontSize={9.5} fill="#8a97ad">{o.ip}:{o.port}</text>
                  <text x={x + 12} y={y + 54} fontSize={9.5} fill="#8a97ad">CPU {o.cpu}% · MEM {o.mem}%</text>
                  <text x={x + 12} y={y + 70} fontSize={9.5} fill="#006aff">Unit 数 {countUnits(cluster, o.name)}</text>
                </Link>
              );
            })}
          </g>
        );
      })}
      <text x={W / 2} y={296} textAnchor="middle" fontSize={9.5} fill="#8a97ad">Zone 间：Paxos 多数派投票 · 资源均衡（Unit 迁移）</text>
    </svg>
  );
}

function countUnits(cluster: Cluster, observer: string): number {
  return (cluster.tenants || []).reduce((acc, t) => acc + t.units.filter(u => u.observer === observer).length, 0);
}

export function TopoSVG({ cluster }: { cluster: Cluster }) {
  return cluster.type === 'oceanbase' ? <ObTopoSVG cluster={cluster} /> : <PgTopoSVG cluster={cluster} />;
}

export function TopoLegend({ cluster }: { cluster: Cluster }) {
  if (cluster.type === 'oceanbase') {
    return (
      <div className="topo-legend">
        <span><i style={{ background: '#00b365' }}></i>正常</span>
        <span><i style={{ background: '#ff9500' }}></i>警告</span>
        <span><i style={{ background: '#f53f3f' }}></i>异常</span>
        <span><i style={{ background: '#0052d9', width: 16, height: 2, borderRadius: 2 }}></i>Zone（可用区）</span>
        <span><i style={{ background: 'rgba(0,106,255,.35)', width: 16, height: 2, borderRadius: 2 }}></i>Paxos / Unit 均衡</span>
      </div>
    );
  }
  return (
    <div className="topo-legend">
      <span><i style={{ background: '#00b365' }}></i>正常</span>
      <span><i style={{ background: '#ff9500' }}></i>警告</span>
      <span><i style={{ background: '#f53f3f' }}></i>异常</span>
      <span><i style={{ background: '#006aff', width: 16, height: 2, borderRadius: 2 }}></i>读写链路</span>
      <span><i style={{ background: 'rgba(0,106,255,.35)', width: 16, height: 2, borderRadius: 2 }}></i>流复制链路</span>
    </div>
  );
}
