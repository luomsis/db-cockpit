import { useEffect, useState } from 'react';
import { Pill } from '../components/bits';
import { HOSTS, type HostRow } from '../lib/mockData';
import { apiGet, withFallback } from '../lib/api';
import { useBreadcrumb } from '../App';
import { FilterChips, Pagination, SearchInput, Th, uniqOpts } from '../components/tableKit';
import type { ActiveFilter } from '../components/tableKit';

/* 规格串 '16C / 64G' → 核数 / 内存 GB */
function parseSpec(spec: string) {
  const m = spec.match(/(\d+)\s*C\s*\/\s*(\d+)\s*G/i);
  return m ? { cores: Number(m[1]), memGb: Number(m[2]) } : { cores: 0, memGb: 0 };
}
const fmtCap = (gb: number) => (gb >= 1024 ? (gb / 1024).toFixed(1) + ' T' : Math.round(gb) + ' G');

/* 资源水位：数值 + 百分比，按水位分色（<70% 常规 / 70-85% 琥珀 / ≥85% 红） */
function ResVal({ text, pct }: { text: string; pct: number }) {
  const cls = pct >= 85 ? 'hot' : pct >= 70 ? 'warn' : '';
  return <span className={`res-val ${cls}`}>{text}</span>;
}

/* 图标统计卡 */
function HostStat({ ico, num, lbl, sub, warn }: { ico: string; num: any; lbl: string; sub?: string; warn?: boolean }) {
  return (
    <div className={`host-stat${warn ? ' warn' : ''}`}>
      <div className="hs-ico">{ico}</div>
      <div className="hs-body">
        <div className="hs-num">{num}</div>
        <div className="hs-lbl">{lbl}</div>
        {sub && <div className="hs-sub">{sub}</div>}
      </div>
    </div>
  );
}

const ST_LABEL: Record<string, string> = { ok: '正常', warn: '警告', err: '异常' };

export default function Hosts() {
  useBreadcrumb([{ label: '首页' }, { label: '主机' }]);

  /* apiserver 主机列表（失败回退本地 mock） */
  const [hosts, setHosts] = useState<HostRow[]>(HOSTS);
  useEffect(() => {
    let alive = true;
    withFallback(apiGet<{ items: HostRow[] }>('/api/hosts?page=1&pageSize=100'), () => null)
      .then(d => { if (alive && d?.items?.length) setHosts(d.items); });
    return () => { alive = false; };
  }, []);
  const statusOpts = Array.from(new Set(hosts.map(h => h.status))).map(s => ({ value: s, label: ST_LABEL[s] || s }));

  const [kw, setKw] = useState('');
  const [fZone, setFZone] = useState('');
  const [fOs, setFOs] = useState('');
  const [fSt, setFSt] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);

  const filtered = hosts.filter(h =>
    (!kw.trim() || (h.ip + ' ' + h.cluster + ' ' + h.insts.join(' ')).includes(kw.trim()))
    && (!fZone || h.zone === fZone)
    && (!fOs || h.os === fOs)
    && (!fSt || h.status === fSt));

  useEffect(() => { setPage(1); }, [kw, fZone, fOs, fSt]);

  const total = filtered.length;
  const pageCount = Math.max(1, Math.ceil(total / pageSize));
  const cur = Math.min(page, pageCount);
  const rows = filtered.slice((cur - 1) * pageSize, cur * pageSize);

  const chips: ActiveFilter[] = [
    fZone && { key: 'zone', label: '可用区', value: fZone, onRemove: () => setFZone('') },
    fOs && { key: 'os', label: '操作系统', value: fOs, onRemove: () => setFOs('') },
    fSt && { key: 'st', label: '状态', value: ST_LABEL[fSt] || fSt, onRemove: () => setFSt('') },
  ].filter(Boolean) as ActiveFilter[];

  const abn = hosts.filter(h => h.status !== 'ok').length;
  const avg = (k: 'cpu' | 'mem' | 'disk') => Math.round(hosts.reduce((a, h) => a + h[k], 0) / hosts.length);
  const top = (k: 'cpu' | 'mem' | 'disk') => hosts.reduce((a, b) => (b[k] > a[k] ? b : a));

  return (
    <>
      <div className="page-title">主机</div>
      <div className="page-desc">实例宿主的资源水位视图（物理机 / 虚机），共 {hosts.length} 台</div>
      <div className="host-stat-row">
        <HostStat ico="🖥" num={hosts.length} lbl="主机总数" sub={`${hosts.filter(h => h.status === 'ok').length} 台运行正常`} />
        <HostStat ico="⚠️" num={abn} lbl="异常/警告主机" sub={abn ? '需要关注' : '全部健康'} warn={!!abn} />
        <HostStat ico="🔥" num={`${avg('cpu')}%`} lbl="平均 CPU" sub={`最高 ${top('cpu').cpu}% · ${top('cpu').ip}`} />
        <HostStat ico="💾" num={`${avg('mem')}%`} lbl="平均内存" sub={`最高 ${top('mem').mem}% · ${top('mem').ip}`} />
        <HostStat ico="🗄️" num={`${avg('disk')}%`} lbl="平均磁盘水位" sub={`最高 ${top('disk').disk}% · ${top('disk').ip}`} />
      </div>
      <div className="filter-bar">
        <SearchInput value={kw} onChange={setKw} placeholder="搜索 IP / 集群 / 实例…" />
        <FilterChips items={chips} />
      </div>
      <div className="card">
        <div className="card-head">
          <div className="card-title">主机列表</div>
          <div className="card-sub">共 {total} 台</div>
        </div>
        <table className="tbl hosts-tbl">
          <thead>
            <tr>
              <Th>IP</Th>
              <Th filter={{ value: fZone, options: uniqOpts(hosts.map(h => h.zone)), onChange: setFZone }}>可用区</Th>
              <Th>规格</Th>
              <Th filter={{ value: fOs, options: uniqOpts(hosts.map(h => h.os)), onChange: setFOs }}>操作系统</Th>
              <Th>CPU</Th><Th>内存</Th><Th>磁盘</Th><Th>实例数</Th>
              <Th filter={{ value: fSt, options: statusOpts, onChange: setFSt }}>状态</Th>
            </tr>
          </thead>
          <tbody>
            {rows.map(h => {
              const { cores, memGb } = parseSpec(h.spec);
              return (
                <tr key={h.ip}>
                  <td className="mono"><span className={`hdot ${h.status}`}></span>{h.ip}</td>
                  <td><span className="tag zone">{h.zone}</span></td>
                  <td className="mono">{h.spec}</td>
                  <td><span className={`os-dot ${h.os.toLowerCase().includes('ubuntu') ? 'ubuntu' : 'centos'}`}></span>{h.os}</td>
                  <td><ResVal text={`${(cores * h.cpu / 100).toFixed(1)} / ${cores} C（${h.cpu}%）`} pct={h.cpu} /></td>
                  <td><ResVal text={`${(memGb * h.mem / 100).toFixed(1)} / ${memGb} G（${h.mem}%）`} pct={h.mem} /></td>
                  <td><ResVal text={`${fmtCap(h.diskTotal * h.disk / 100)} / ${fmtCap(h.diskTotal)}（${h.disk}%）`} pct={h.disk} /></td>
                  <td><span className="inst-badge">{h.insts.length}</span></td>
                  <td><Pill st={h.status} /></td>
                </tr>
              );
            })}
            {!rows.length && <tr><td colSpan={9}><div className="empty" style={{ padding: '40px 0' }}>没有匹配的主机</div></td></tr>}
          </tbody>
        </table>
        <Pagination total={total} page={cur} pageSize={pageSize} onPage={setPage} onPageSize={setPageSize} />
      </div>
    </>
  );
}
