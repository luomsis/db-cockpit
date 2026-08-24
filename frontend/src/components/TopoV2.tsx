import { useMemo } from 'react';
import type { TopoComponent, TopoScenario } from '../lib/topoMock';

/* 通用拓扑引擎（v2 设计原型）：
 * 纵向 = 数据流分层（proxy/access → tenant → storage，实线折线箭头顺流向下；不画客户端节点）
 * 横向 = 复制关系（主→备 虚线折线桥 + 延迟标签；Paxos 对称虚线）
 * 背景 = 物理分区（region · az / host_cluster 全局分栏：每栏一个贯穿各层的虚线大框，互不重叠）
 * 租户视图 = 租户大卡 + units 落位点线
 * 高亮 = 目标脉冲描边 + 关联链（流量/复制/落位一跳）常亮，其余降透明 */

export interface TopoV2Props {
  scenario: TopoScenario;
  physicalDim: 'off' | 'az' | 'host_cluster';
  view?: 'cluster' | 'tenant';
  highlightId?: string;
  onSelect?: (id: string) => void;
}

const W = 150, H = 68, TENANT_W = 256, TENANT_H = 86;
const NODE_GAP = 24, GROUP_GAP = 46, MARGIN = 34, LAYER_GAP = 96;
const AZ_GAP = 44, PHYS_PAD = 12;

/* 平台色板：物理分区栏 / 数据流来源着色共用（超 5 循环） */
const PALETTE = ['#006aff', '#00a3e0', '#ff9500', '#3ecf8e', '#7c5cff'];
const KIND_COLOR: Record<string, string> = {
  proxy: '#00a3e0', storage: '#006aff', tenant: '#7c5cff', compute: '#5a6f99', access: '#3ecf8e', arbiter: '#ff9500',
};
const STATUS_DOT: Record<string, string> = { ok: '#3ecf8e', warn: '#ff9500', err: '#e5484d' };
const ROLE_PILL: Record<string, [string, string]> = {
  primary: ['#e8f2ff', '#006aff'], master: ['#e8f2ff', '#006aff'], active: ['#e8f2ff', '#006aff'],
  secondary: ['#f1f4f9', '#5a6779'], replica: ['#f1f4f9', '#5a6779'], backup: ['#f1f4f9', '#5a6779'],
  observer: ['#e6f7fd', '#0089c4'], router: ['#e6f7fd', '#0089c4'], user: ['#f3efff', '#7c5cff'],
};

interface Pos { x: number; y: number; w: number; h: number; cx: number; cy: number }
interface PhysBox { label: string; x1: number; y1: number; x2: number; y2: number }

function nodeSize(c: TopoComponent, view: 'cluster' | 'tenant') {
  if (c.kind === 'tenant' && view === 'tenant') return { w: TENANT_W, h: TENANT_H };
  return { w: W, h: H };
}

/* 自绘内联 SVG 线性图标（14×14，stroke 1.5，随 kind 色） */
function KindIcon({ kind, x, y, color }: { kind: string; x: number; y: number; color: string }) {
  const s = { stroke: color, strokeWidth: 1.5, fill: 'none', strokeLinecap: 'round' as const, strokeLinejoin: 'round' as const };
  const body = () => {
    switch (kind) {
      case 'proxy': // 分流路由：一进两出
        return <>
          <path d="M7 2v4" {...s} />
          <path d="M7 6C7 8.2 3.4 8.2 3.4 11" {...s} />
          <path d="M7 6C7 8.2 10.6 8.2 10.6 11" {...s} />
          <path d="M2.2 9.6L3.4 11.4L4.6 9.6" {...s} />
          <path d="M9.4 9.6L10.6 11.4L11.8 9.6" {...s} />
        </>;
      case 'storage': // 数据库圆柱
        return <>
          <ellipse cx="7" cy="3.6" rx="5" ry="2" {...s} />
          <path d="M2 3.6v6.8c0 1.1 2.24 2 5 2s5-.9 5-2V3.6" {...s} />
          <path d="M2 7c0 1.1 2.24 2 5 2s5-.9 5-2" {...s} />
        </>;
      case 'tenant': // 层叠方块
        return <>
          <rect x="3" y="2" width="8" height="3" rx="1" {...s} />
          <rect x="3" y="5.5" width="8" height="3" rx="1" {...s} />
          <rect x="3" y="9" width="8" height="3" rx="1" {...s} />
        </>;
      case 'arbiter': // 天平
        return <>
          <path d="M7 2v9M4 12h6M3 4h8" {...s} />
          <path d="M3 4L1.4 7.6h3.2L3 4z" {...s} />
          <path d="M11 4L9.4 7.6h3.2L11 4z" {...s} />
        </>;
      case 'compute': // 芯片
        return <>
          <rect x="4" y="4" width="6" height="6" rx="1" {...s} />
          <path d="M7 1.2v2M7 10.8v2M1.2 7h2M10.8 7h2" {...s} />
        </>;
      default: // access 网关
        return <>
          <rect x="3" y="2.5" width="8" height="9" rx="1.2" {...s} />
          <path d="M6 11.5V8h2v3.5" {...s} />
        </>;
    }
  };
  return <g transform={`translate(${x},${y})`}>{body()}</g>;
}

/* 文本实测宽度（9.5px 字号，中西文混排）与超长截断——胶囊排版不再溢出 */
const textW = (s: string) => [...s].reduce((a, ch) => a + (ch.charCodeAt(0) > 0x2e80 ? 9.5 : 5.9), 0);
const ellipsize = (s: string, max: number) => {
  if (textW(s) <= max) return s;
  let out = '';
  for (const ch of s) {
    if (textW(out + ch) > max - 8) return out + '…';
    out += ch;
  }
  return out;
};

function Pill({ x, y, text, bg, fg, maxW }: { x: number; y: number; text: string; bg: string; fg: string; maxW?: number }) {
  const label = maxW ? ellipsize(text, maxW) : text;
  const w = textW(label) + 12;
  return (
    <g>
      <rect x={x} y={y} width={w} height={15} rx={7.5} fill={bg} />
      <text x={x + w / 2} y={y + 10.5} textAnchor="middle" fontSize={9.5} fill={fg} fontWeight={600}>{label}</text>
    </g>
  );
}

/* 高亮关联链：流量/复制上下游 + 租户落位（一跳，双向） */
function relatedSet(comps: TopoComponent[], id: string): Set<string> {
  const rel = new Set<string>([id]);
  const byId = new Map(comps.map(c => [c.id, c]));
  const me = byId.get(id);
  if (me) {
    if (me.traffic_upstream_id) rel.add(me.traffic_upstream_id);
    if (me.replication_upstream_id) rel.add(me.replication_upstream_id);
    me.extensions?.units?.forEach(u => rel.add(u.instance_id));
  }
  for (const c of comps) {
    if (c.traffic_upstream_id === id || c.replication_upstream_id === id) rel.add(c.id);
    if (c.extensions?.units?.some(u => u.instance_id === id)) rel.add(c.id);
  }
  return rel;
}

export default function TopoV2({ scenario, physicalDim, view = 'cluster', highlightId, onSelect }: TopoV2Props) {
  const layout = useMemo(() => {
    const comps = scenario.components;
    const hostMap = new Map(scenario.hosts.map(h => [h.host_ip, h]));
    const physOn = physicalDim !== 'off';
    const physKey = physicalDim === 'host_cluster' ? 'host_cluster' : 'az';
    // 物理标签 = region · az（或 region · host_cluster）；组件经 host_ip 定位，逻辑单元（无 host_ip）为 null
    const labelOf = (c: TopoComponent): string | null => {
      if (!physOn || !c.host_ip) return null;
      const h = hostMap.get(c.host_ip);
      return h ? `${h.region} · ${h[physKey]}` : null;
    };
    // 分栏：hosts 出现顺序去重（同 region 相邻归拢），仅保留有组件落入的标签；逻辑单元栏殿后（不画框）
    const seen: string[] = [];
    for (const h of scenario.hosts) {
      const l = `${h.region} · ${h[physKey]}`;
      if (!seen.includes(l)) seen.push(l);
    }
    const columns: (string | null)[] = seen.filter(l => comps.some(c => labelOf(c) === l));
    if (comps.some(c => labelOf(c) === null)) columns.push(null);

    // 层序：配置优先，未列出的 kind 追加（按出现顺序）
    const kinds = scenario.layerOrder.filter(k => comps.some(c => c.kind === k));
    for (const c of comps) if (!kinds.includes(c.kind)) kinds.push(c.kind);
    const layers = kinds.map(k => comps.filter(c => c.kind === k));

    // 间距判定统一：同组相邻 24、跨组 46（空 group_name 视为跨组），估算与摆放用同一规则
    const gapBetween = (a: TopoComponent, b: TopoComponent) =>
      a.group_name != null && a.group_name === b.group_name ? NODE_GAP : GROUP_GAP;

    // 每层每栏的内容宽 → 栏宽取各层最大值，保证所有层共享同一横向分栏（跨层对齐）
    const colLayerWidth = columns.map(col => layers.map(items => {
      const cs = items.filter(c => labelOf(c) === col);
      let w = 0;
      cs.forEach((c, i) => { if (i > 0) w += gapBetween(cs[i - 1], c); w += nodeSize(c, view).w; });
      return w;
    }));
    const colWidth = colLayerWidth.map(ws => Math.max(...ws));
    const colX: number[] = [];
    let cx = MARGIN;
    for (let i = 0; i < columns.length; i++) { colX.push(cx); cx += colWidth[i] + AZ_GAP; }
    const canvasW = Math.max(cx - AZ_GAP + MARGIN, 420);

    const pos = new Map<string, Pos>();
    const groups: { label: string; x: number; y: number; w: number }[] = [];
    const phys: PhysBox[] = [];
    let y = 26;
    const layerTops: number[] = [];
    const layerBottoms: number[] = [];
    layers.forEach((items, li) => {
      const layerH = Math.max(...items.map(c => nodeSize(c, view).h));
      layerTops.push(y);
      let prevGroup: string | undefined;
      let groupStart = -1;
      let endX = 0;
      columns.forEach((col, ci) => {
        const cs = items.filter(c => labelOf(c) === col);
        if (!cs.length) return;
        let x = colX[ci] + (colWidth[ci] - colLayerWidth[ci][li]) / 2;
        cs.forEach((c, i) => {
          if (!physOn && prevGroup !== undefined && (c.group_name ?? '') !== prevGroup) {
            groups.push({ label: prevGroup, x: groupStart - 6, y: y - 20, w: x - GROUP_GAP + NODE_GAP / 2 - groupStart + 12 });
            groupStart = -1;
          }
          if (!physOn && groupStart < 0) groupStart = x;
          const { w: nw, h: nh } = nodeSize(c, view);
          const ny = y + (layerH - nh) / 2;
          pos.set(c.id, { x, y: ny, w: nw, h: nh, cx: x + nw / 2, cy: ny + nh / 2 });
          x += nw + (i < cs.length - 1 ? gapBetween(cs[i], cs[i + 1]) : GROUP_GAP);
          prevGroup = c.group_name;
          endX = x;
        });
      });
      if (!physOn && groupStart >= 0 && prevGroup) {
        groups.push({ label: prevGroup, x: groupStart - 6, y: y - 20, w: endX - GROUP_GAP + NODE_GAP / 2 - groupStart + 12 });
      }
      layerBottoms.push(y + layerH);
      y += layerH + LAYER_GAP;
    });
    // 物理分区框：每栏一个贯穿所有层的纵向大框（栏间 x 互斥 → 永不重叠）；逻辑单元栏不画框
    if (physOn) {
      columns.forEach((col, ci) => {
        if (col === null) return;
        phys.push({
          label: col,
          x1: colX[ci] - PHYS_PAD,
          y1: layerTops[0] - 26,
          x2: colX[ci] + colWidth[ci] + PHYS_PAD,
          y2: layerBottoms[layerBottoms.length - 1] + 12,
        });
      });
    }
    const canvasH0 = y - LAYER_GAP + (layers.length > 1 ? 26 : 10);
    // 复制折线桥向下延伸（同主多备分层避让），画布高度需覆盖最深桥
    let replBottom = 0;
    const peerIdx = new Map<string, number>();
    for (const c of comps) {
      if (!c.replication_upstream_id) continue;
      const m = pos.get(c.replication_upstream_id);
      if (!m) continue;
      const idx = peerIdx.get(c.replication_upstream_id) ?? 0;
      peerIdx.set(c.replication_upstream_id, idx + 1);
      replBottom = Math.max(replBottom, m.y + m.h + 26 + idx * 14);
    }
    const canvasH = Math.max(canvasH0, replBottom + 20);
    // 数据流箭头按来源实例着色（首现顺序分配色板）
    const srcColor = new Map<string, string>();
    for (const c of comps) {
      if (c.traffic_upstream_id && !srcColor.has(c.traffic_upstream_id)) {
        srcColor.set(c.traffic_upstream_id, PALETTE[srcColor.size % PALETTE.length]);
      }
    }
    return { pos, groups, canvasW, canvasH, layers, layerTops, layerBottoms, phys, srcColor };
  }, [scenario, physicalDim, view]);

  const comps = scenario.components;
  const byId = new Map(comps.map(c => [c.id, c]));
  const rel = highlightId ? relatedSet(comps, highlightId) : null;
  const dim = (id: string) => (rel && !rel.has(id) ? 0.32 : 1);

  // 纵向折线（上→下，直角肘形）
  const elbowV = (a: Pos, b: Pos) => {
    const sy = a.y + a.h, ty = b.cy - b.h / 2, my = (sy + ty) / 2;
    return `M ${a.cx} ${sy} V ${my} H ${b.cx} V ${ty}`;
  };

  return (
    <div className="topo-wrap" style={{ overflowX: 'auto' }}>
      <svg width={layout.canvasW} height={layout.canvasH} viewBox={`0 0 ${layout.canvasW} ${layout.canvasH}`} style={{ display: 'block', margin: '0 auto' }}>
        <defs>
          {PALETTE.map((c, i) => (
            <marker key={i} id={`tv-f-${i}`} markerWidth="9" markerHeight="8" refX="7.5" refY="4" orient="auto"><path d="M0,0 L8,4 L0,8 Z" fill={c} /></marker>
          ))}
          <marker id="tv-repl" markerWidth="9" markerHeight="8" refX="7.5" refY="4" orient="auto"><path d="M0,0 L8,4 L0,8 Z" fill="#8d99ad" /></marker>
          <marker id="tv-repl-b" markerWidth="9" markerHeight="8" refX="0.5" refY="4" orient="auto"><path d="M8,0 L0,4 L8,8 Z" fill="#8d99ad" /></marker>
        </defs>
        <style>{`
          @keyframes tvPulse { 0%,100% { stroke-opacity: 1; } 50% { stroke-opacity: .35; } }
          .tv-node { transition: transform .15s ease; }
          .tv-node:hover { transform: translateY(-1.5px); }
        `}</style>

        {/* 物理分区背景框：每栏一个贯穿各层的纵向大框（轻盈线稿风：低饱和填充 + 淡描边） */}
        {layout.phys.map((b, i) => {
          const color = PALETTE[i % PALETTE.length];
          return (
            <g key={b.label}>
              <rect x={b.x1} y={b.y1} width={b.x2 - b.x1} height={b.y2 - b.y1} rx={12}
                fill={color} fillOpacity={0.03}
                stroke={color} strokeOpacity={0.45} strokeDasharray="7 5" strokeWidth={1.3} />
              <text x={b.x1 + 10} y={b.y1 + 15} fontSize={10} fill={color} fillOpacity={0.85} fontWeight={600}>{b.label}</text>
            </g>
          );
        })}

        {/* 分组标签 */}
        {layout.groups.filter(g => g.label).map(g => (
          <text key={`${g.label}-${g.x}`} x={g.x + g.w / 2} y={g.y + 4} textAnchor="middle" fontSize={10.5} fill="#98a3b5">{g.label}</text>
        ))}

        {/* ① 数据流（traffic_upstream：上游→我，实线折线，按来源实例着色） */}
        {comps.filter(c => c.traffic_upstream_id).map(c => {
          const up = layout.pos.get(c.traffic_upstream_id!);
          const p = layout.pos.get(c.id);
          if (!up || !p) return null;
          const color = layout.srcColor.get(c.traffic_upstream_id!) ?? PALETTE[0];
          return <path key={`t-${c.id}`} d={elbowV(up, p)} fill="none" stroke={color} strokeWidth={1.8}
            markerEnd={`url(#tv-f-${PALETTE.indexOf(color)})`}
            opacity={highlightId && !(rel?.has(c.id) || rel?.has(c.traffic_upstream_id!)) ? 0.3 : 0.85} />;
        })}

        {/* ② 复制（主→备，虚线折线桥：从主库底部下行、横跨、上行进备库；同主多备分层避让） */}
        {comps.filter(c => c.replication_upstream_id).map(c => {
          const m = layout.pos.get(c.replication_upstream_id!), p = layout.pos.get(c.id);
          if (!m || !p) return null;
          const peers = comps.filter(x => x.replication_upstream_id === c.replication_upstream_id);
          const lift = 26 + peers.indexOf(c) * 14;
          const d = `M ${m.cx} ${m.y + m.h} V ${m.y + m.h + lift} H ${p.cx} V ${p.y + p.h}`;
          return (
            <g key={`r-${c.id}`} opacity={highlightId && !(rel?.has(c.id) || rel?.has(c.replication_upstream_id!)) ? 0.35 : 1}>
              <path d={d} fill="none" stroke="#8d99ad" strokeWidth={1.5} strokeDasharray="6 4" markerEnd="url(#tv-repl)" />
              <text x={(m.cx + p.cx) / 2} y={m.y + m.h + lift - 4} textAnchor="middle" fontSize={9.5} fill="#8d99ad">
                {`${c.extensions?.sync ?? '复制'}${c.extensions?.delay_ms != null ? ` · ${c.extensions.delay_ms}ms` : ''}`}
              </text>
            </g>
          );
        })}

        {/* ③ Paxos（多主对称虚线，层底组间） */}
        {view === 'cluster' && (() => {
          const out: JSX.Element[] = [];
          layout.layers.forEach((items, li) => {
            const gnames = [...new Set(items.map(c => c.group_name).filter(Boolean))] as string[];
            const paxosGroups = gnames.filter(g => items.some(c => c.group_name === g && c.extensions?.paxos));
            if (paxosGroups.length < 2) return;
            const yLine = layout.layerBottoms[li] + 16;
            const cents = paxosGroups.map(g => {
              const ps = items.filter(c => c.group_name === g).map(c => layout.pos.get(c.id)!);
              return (Math.min(...ps.map(p => p.x)) + Math.max(...ps.map(p => p.x + p.w))) / 2;
            });
            for (let i = 0; i < cents.length - 1; i++) {
              out.push(<line key={`px-${li}-${i}`} x1={cents[i]} y1={yLine} x2={cents[i + 1]} y2={yLine}
                stroke="#8d99ad" strokeWidth={1.5} strokeDasharray="6 4" markerEnd="url(#tv-repl)" markerStart="url(#tv-repl-b)" opacity={0.85} />);
            }
            out.push(<text key={`pxt-${li}`} x={(cents[0] + cents[cents.length - 1]) / 2} y={yLine + 13} textAnchor="middle" fontSize={9.5} fill="#8d99ad">Paxos 多数派（多主，复制字段置空）</text>);
          });
          return out;
        })()}

        {/* ④ 租户落位（units → observer，点线；租户视图/集群视图均画） */}
        {comps.filter(c => c.kind === 'tenant').map(t =>
          (t.extensions?.units ?? []).map(u => {
            const tp = layout.pos.get(t.id), op = layout.pos.get(u.instance_id);
            if (!tp || !op) return null;
            return <path key={`u-${t.id}-${u.instance_id}`} d={elbowV(tp, op)} fill="none" stroke="#b9c2d0" strokeWidth={1.4} strokeDasharray="2 5"
              opacity={highlightId && !(rel?.has(t.id) || rel?.has(u.instance_id)) ? 0.3 : 0.95} />;
          })
        )}

        {/* 节点卡片：三行（名称 / role 胶囊 / 地址）+ kind 左色条 + 状态/来源圆点 */}
        {comps.map(c => {
          const p = layout.pos.get(c.id)!;
          const tenantBig = c.kind === 'tenant' && view === 'tenant';
          const isTarget = highlightId === c.id;
          const kindColor = KIND_COLOR[c.kind] ?? '#5a6f99';
          const roleLabel = physicalDim !== 'off' && c.group_name ? `${c.role ?? c.kind} · ${c.group_name}` : (c.role ?? c.kind);
          const [pbg, pfg] = ROLE_PILL[c.role ?? ''] ?? ['#f1f4f9', '#5a6779'];
          const srcDot = layout.srcColor.get(c.id); // 我作为流量来源的专属色（右下角圆点呼应箭头）
          return (
            <g key={c.id} className="tv-node" opacity={dim(c.id)} onClick={() => onSelect?.(c.id)} style={{ cursor: 'pointer' }}>
              <title>{`${c.name} · ${c.kind}${c.role ? ' / ' + c.role : ''}${c.group_name ? ' / ' + c.group_name : ''}${c.host_ip ? ` · ${c.host_ip}:${c.port}` : ''}${c.status && c.status !== 'ok' ? ` · ${c.status}` : ''}`}</title>
              {isTarget && <rect x={p.x - 4} y={p.y - 4} width={p.w + 8} height={p.h + 8} rx={10} fill="none" stroke="#006aff" strokeWidth={2.2} style={{ animation: 'tvPulse 1.6s infinite' }} />}
              <rect x={p.x} y={p.y} width={p.w} height={p.h} rx={8} fill="#fff" stroke={isTarget ? '#006aff' : '#dfe6f0'} strokeWidth={isTarget ? 1.8 : 1.2}
                style={{ filter: 'drop-shadow(0 1px 2px rgba(15,35,70,.06))' }} />
              <rect x={p.x} y={p.y + 8} width={3} height={p.h - 16} rx={1.5} fill={kindColor} />
              {c.status && c.status !== 'ok' && <circle cx={p.x + p.w - 10} cy={p.y + 10} r={2.8} fill={STATUS_DOT[c.status]} />}
              {srcDot && <circle cx={p.x + p.w - 10} cy={p.y + p.h - 10} r={3.4} fill={srcDot} />}
              <KindIcon kind={c.kind} x={p.x + 13} y={p.y + 9} color={kindColor} />
              <text x={p.x + 32} y={p.y + 21} fontSize={12} fontWeight={700} fill="#253248">{c.name}</text>
              {tenantBig ? (
                <>
                  <Pill x={p.x + 12} y={p.y + 30} text={c.extensions?.mode ?? '-'} bg="#f3efff" fg="#7c5cff" />
                  <Pill x={p.x + 12 + textW(c.extensions?.mode ?? '-') + 18} y={p.y + 30} text={c.extensions?.unit ?? '-'} bg="#e8f2ff" fg="#006aff" />
                  <Pill x={p.x + 12} y={p.y + 52} text={`unit 落位 ×${c.extensions?.units?.length ?? 0} 副本（点线）`} bg="#f1f4f9" fg="#5a6779" />
                </>
              ) : (
                <>
                  <Pill x={p.x + 12} y={p.y + 30} text={roleLabel} bg={pbg} fg={pfg} maxW={p.w - 24} />
                  <text x={p.x + p.w - 10} y={p.y + 60} textAnchor="end" fontSize={10} fill="#98a3b5" fontFamily="Menlo,monospace">
                    {c.host_ip ? `${c.host_ip}:${c.port}` : '逻辑单元'}
                  </text>
                </>
              )}
            </g>
          );
        })}
      </svg>
    </div>
  );
}

export function TopoV2Legend() {
  const items = [
    { t: '实线箭头', d: '数据流（纵向 · 按来源实例着色）', color: '#006aff', dash: '' },
    { t: '虚线箭头', d: '复制 主→备（横向）', color: '#8d99ad', dash: '6 4' },
    { t: '点线', d: '租户 unit 落位', color: '#b9c2d0', dash: '2 5' },
    { t: '虚线框', d: '物理分区（region/AZ/主机集群）', color: '#006aff', dash: '7 5', box: true },
  ];
  return (
    <div style={{ display: 'flex', gap: 18, flexWrap: 'wrap', fontSize: 11.5, color: '#8a97ad', padding: '8px 4px' }}>
      {items.map(i => (
        <span key={i.t} style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}>
          <svg width={30} height={12}>{i.box
            ? <rect x={1} y={1} width={28} height={10} rx={3} fill="none" stroke={i.color} strokeDasharray={i.dash} />
            : <line x1={1} y1={6} x2={29} y2={6} stroke={i.color} strokeWidth={2} strokeDasharray={i.dash} />}</svg>
          {i.t} = {i.d}
        </span>
      ))}
      <span>左上圆点 = 异常状态 · 右下圆点 = 该节点流出流量的颜色 · 点击节点切换高亮</span>
    </div>
  );
}
