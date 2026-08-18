import { useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { Pill, TypeTag } from '../components/bits';
import { CLUSTERS, DB_TYPES } from '../lib/mockData';
import { apiGet, withFallback } from '../lib/api';
import { useBreadcrumb } from '../App';
import { EyeIcon, FilterChips, Pagination, SearchInput, Th, uniqOpts } from '../components/tableKit';
import type { ActiveFilter } from '../components/tableKit';
import type { Cluster } from '../lib/types';

const verText = (c: Cluster) => c.version.split(' ').slice(1).join(' ') || c.version;
const clusterStatus = (c: Cluster) =>
  c.instances.some(i => i.status === 'err') ? 'err'
    : c.instances.some(i => i.status === 'warn') ? 'warn' : 'ok';

const ST_OPTS = [{ value: 'ok', label: '正常' }, { value: 'warn', label: '警告' }, { value: 'err', label: '异常' }];

export default function Clusters() {
  const { '*': focusParam } = useParams();
  useBreadcrumb([{ label: '首页' }, { label: '集群管理' }]);

  /* apiserver 集群列表（失败回退本地 mock 常量，筛选逻辑保持客户端不变） */
  const [clusters, setClusters] = useState<Cluster[]>(CLUSTERS);
  useEffect(() => {
    let alive = true;
    withFallback(
      apiGet<{ items: Cluster[] }>('/api/clusters?page=1&pageSize=100'),
      () => null,
    ).then(d => { if (alive && d?.items?.length) setClusters(d.items); });
    return () => { alive = false; };
  }, []);

  const typeOpts = Array.from(new Set(clusters.map(c => c.type)))
    .map(t => ({ value: t, label: DB_TYPES.find(d => d.type === t)?.name || t }));

  const [kw, setKw] = useState(() => (focusParam ? clusters.find(c => c.id === focusParam)?.name || '' : ''));
  const [fType, setFType] = useState('');
  const [fVer, setFVer] = useState('');
  const [fAz, setFAz] = useState('');
  const [fSt, setFSt] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);

  const filtered = clusters.filter(c =>
    (!kw.trim() || (c.name + ' ' + c.desc + ' ' + c.biz).toLowerCase().includes(kw.trim().toLowerCase()))
    && (!fType || c.type === fType)
    && (!fVer || verText(c) === fVer)
    && (!fAz || c.az === fAz)
    && (!fSt || clusterStatus(c) === fSt));

  /* 任一筛选变化时回到第 1 页 */
  useEffect(() => { setPage(1); }, [kw, fType, fVer, fAz, fSt]);

  const chips: ActiveFilter[] = [
    fType && { key: 'type', label: '数据库类型', value: typeOpts.find(o => o.value === fType)?.label || fType, onRemove: () => setFType('') },
    fVer && { key: 'ver', label: '版本号', value: fVer, onRemove: () => setFVer('') },
    fAz && { key: 'az', label: '可用区', value: fAz, onRemove: () => setFAz('') },
    fSt && { key: 'st', label: '状态', value: ST_OPTS.find(o => o.value === fSt)?.label || fSt, onRemove: () => setFSt('') },
  ].filter(Boolean) as ActiveFilter[];

  const total = filtered.length;
  const pageCount = Math.max(1, Math.ceil(total / pageSize));
  const cur = Math.min(page, pageCount);
  const rows = filtered.slice((cur - 1) * pageSize, cur * pageSize);

  return (
    <>
      <div className="page-title">集群管理</div>
      <div className="page-desc">共 {clusters.length} 个集群 · {clusters.reduce((a, c) => a + c.instances.length, 0)} 个实例，点击操作列图标查看集群详情</div>
      <div className="filter-bar">
        <SearchInput value={kw} onChange={setKw} placeholder="搜索集群名称 / 描述…" />
        <FilterChips items={chips} />
      </div>
      <div className="card">
        <table className="tbl">
          <thead>
            <tr>
              <Th>集群名称</Th>
              <Th filter={{ value: fType, options: typeOpts, onChange: setFType }}>数据库类型</Th>
              <Th filter={{ value: fVer, options: uniqOpts(clusters.map(verText)), onChange: setFVer }}>版本号</Th>
              <Th filter={{ value: fAz, options: uniqOpts(clusters.map(c => c.az)), onChange: setFAz }}>可用区</Th>
              <Th filter={{ value: fSt, options: ST_OPTS, onChange: setFSt }}>状态</Th>
              <th style={{ width: 72 }}>操作</th>
            </tr>
          </thead>
          <tbody>
            {rows.map(c => (
              <tr key={c.id}>
                <td className="mono"><Link to={`/cluster/${c.id}`}>{c.name}</Link></td>
                <td><TypeTag t={c.type} /></td>
                <td className="mono">{verText(c)}</td>
                <td>{c.az}</td>
                <td><Pill st={clusterStatus(c)} /></td>
                <td><Link className="icon-op" to={`/cluster/${c.id}`} title="查看详情"><EyeIcon /></Link></td>
              </tr>
            ))}
            {!rows.length && <tr><td colSpan={6}><div className="empty" style={{ padding: '40px 0' }}>没有匹配的集群</div></td></tr>}
          </tbody>
        </table>
        <Pagination total={total} page={cur} pageSize={pageSize} onPage={setPage} onPageSize={setPageSize} />
      </div>
    </>
  );
}
