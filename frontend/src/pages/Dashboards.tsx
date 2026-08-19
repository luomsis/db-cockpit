import { useEffect, useMemo, useState } from 'react';
import { Navigate, useNavigate } from 'react-router-dom';
import { InfoDialog, ConfirmDialog, MenuPopover } from '../components/dialogs';
import { loadDashboards, saveDashboards, normalizeDashboard, relTime, serverInit, serverCreate, serverUpdate, serverDelete } from '../lib/dashboards';
import type { Dashboard } from '../lib/types';
import { IconSearch } from '../components/icons';
import { toast } from '../lib/toast';
import { useBreadcrumb } from '../App';

/* /dashboards 入口：默认直达「默认大盘」（服务端同步后跳转） */
export function DashboardEntry() {
  const [target, setTarget] = useState<string | null>(null);
  useEffect(() => {
    serverInit().then(list => {
      const t = list.find(d => d.id === 'd-default') || list[0];
      setTarget(t?.id ?? '');
    }).catch(() => setTarget(''));
  }, []);
  if (target === null) return null;
  return target ? <Navigate to={`/dashboard/${target}`} replace /> : <Navigate to="/dashboards/list" replace />;
}

export default function Dashboards() {
  useBreadcrumb([{ label: '首页' }, { label: '监控大盘' }]);
  const navigate = useNavigate();
  const [list, setList] = useState<Dashboard[]>(() => loadDashboards());
  const [kw, setKw] = useState('');
  const [metaDialog, setMetaDialog] = useState<{ mode: 'create' } | { mode: 'edit'; dash: Dashboard } | null>(null);
  const [delDash, setDelDash] = useState<Dashboard | null>(null);

  /* 服务端同步（失败保持本地镜像，可继续操作） */
  const refresh = () => serverInit().then(setList).catch(() => { });
  useEffect(() => { refresh(); /* eslint-disable-next-line */ }, []);

  const filtered = useMemo(() => {
    const sorted = list.slice().sort((a, b) => b.updatedAt - a.updatedAt);
    return kw ? sorted.filter(d => (d.title + ' ' + d.description).toLowerCase().includes(kw.toLowerCase())) : sorted;
  }, [list, kw]);

  const createDashboard = async (title: string, description: string) => {
    try {
      const d = await serverCreate(title, description);
      if (d) { toast.success(`大盘「${title}」已创建`); refresh(); navigate(`/dashboard/${d.id}`); return; }
    } catch (e) { /* 落回本地 */ toast.info('apiserver 不可达，大盘已保存到本地'); }
    const now = Date.now();
    const d = normalizeDashboard({ id: 'd' + now, title, description, cfg: { range: '24h', refresh: '0', compareYesterday: false }, panels: [], createdAt: now, updatedAt: now })!;
    const arr = loadDashboards(); arr.push(d); saveDashboards(arr);
    setList(arr);
    navigate(`/dashboard/${d.id}`);
  };

  const duplicate = async (d: Dashboard) => {
    try {
      const copy = await serverCreate(d.title + ' 副本', d.description, JSON.parse(JSON.stringify(d.cfg)), JSON.parse(JSON.stringify(d.panels)));
      if (copy) { toast.success(`已创建「${d.title}」副本`); refresh(); return; }
    } catch (e) { /* 落回本地 */ }
    const now = Date.now();
    const copy = normalizeDashboard({ id: 'd' + now, title: d.title + ' 副本', description: d.description, cfg: JSON.parse(JSON.stringify(d.cfg)), panels: JSON.parse(JSON.stringify(d.panels)), createdAt: now, updatedAt: now })!;
    const arr = loadDashboards(); arr.push(copy); saveDashboards(arr);
    toast.success(`已创建「${d.title}」副本（本地）`);
    refresh();
  };

  return (
    <>
      <div className="page-title">监控大盘 <span className="pill info"><i></i>共 {list.length} 个大盘</span></div>
      <div className="page-desc">点击卡片进入大盘；支持新建、编辑信息、制作副本、删除。大盘配置存储于 apiserver（离线时回退本地浏览器）。</div>
      <div className="dash-list-head">
        <div className="dash-search-wrap">
          <span className="ico"><IconSearch /></span>
          <input className="dash-search" placeholder="搜索大盘标题或描述…" value={kw} onChange={e => setKw(e.target.value)} />
        </div>
        <button className="btn sm primary" onClick={() => setMetaDialog({ mode: 'create' })}>+ 新建大盘</button>
      </div>
      <div className="dash-list-grid">
        {filtered.map(d => {
          const n = (d.panels || []).filter(p => p.visible !== false).length;
          return (
            <div className="dash-card" key={d.id} title="点击进入" onClick={() => navigate(`/dashboard/${d.id}`)}>
              <div className="dash-card-accent"></div>
              <div className="dash-card-body">
                <div className="dash-card-title">{d.title}</div>
                <div className="dash-card-desc">{d.description || <span className="muted">暂无描述</span>}</div>
                <div className="dash-card-meta">
                  <span>{n} 个面板 · {relTime(d.updatedAt)}</span>
                  <CardMenu items={[
                    { label: '打开', onClick: () => navigate(`/dashboard/${d.id}`) },
                    { label: '编辑信息', onClick: () => setMetaDialog({ mode: 'edit', dash: d }) },
                    { label: '制作副本', onClick: () => duplicate(d) },
                    { label: '删除', onClick: () => setDelDash(d) },
                  ]} />
                </div>
              </div>
            </div>
          );
        })}
        {!filtered.length && kw && <div className="dash-empty-hint">没有找到匹配的大盘</div>}
        <div className="dash-card dash-new-card" onClick={() => setMetaDialog({ mode: 'create' })}>
          <span className="dash-new-plus">＋</span>新建大盘
        </div>
      </div>

      {metaDialog?.mode === 'create' && (
        <InfoDialog title="新建大盘" okText="创建并进入" onOk={createDashboard} onClose={() => setMetaDialog(null)} />
      )}
      {metaDialog?.mode === 'edit' && (
        <InfoDialog title="编辑大盘信息" okText="保存"
          initial={{ title: metaDialog.dash.title, description: metaDialog.dash.description }}
          onOk={(title, description) => {
            const arr = loadDashboards();
            const idx = arr.findIndex(x => x.id === metaDialog.dash.id);
            if (idx >= 0) { arr[idx] = { ...arr[idx], title, description, updatedAt: Date.now() }; saveDashboards(arr); }
            serverUpdate(metaDialog.dash.id, { title, description })
              .then(() => toast.success('大盘信息已更新'))
              .catch(() => toast.info('apiserver 不可达，信息已保存到本地'))
              .finally(() => refresh());
            setMetaDialog(null);
          }}
          onClose={() => setMetaDialog(null)} />
      )}
      {delDash && (
        <ConfirmDialog title="删除大盘" okText="删除" danger
          message={`确定删除大盘「${delDash.title}」吗？该操作不可恢复。`}
          onOk={() => {
            saveDashboards(loadDashboards().filter(x => x.id !== delDash.id));
            serverDelete(delDash.id)
              .then(() => toast.success(`已删除大盘「${delDash.title}」`))
              .catch(() => { /* 本地已删，服务端失败静默 */ })
              .finally(() => refresh());
            setDelDash(null);
          }}
          onClose={() => setDelDash(null)} />
      )}
    </>
  );
}

/* 卡片 ⋯ 菜单（复用 MenuPopover：点击外部自动关闭） */
function CardMenu({ items }: { items: { label: string; onClick: () => void }[] }) {
  const [open, setOpen] = useState(false);
  return (
    <span style={{ position: 'relative' }} onClick={e => e.stopPropagation()}>
      {open && <MenuPopover items={items} onClose={() => setOpen(false)} style={{ position: 'absolute', top: 20, right: 0 }} />}
      <span className="dash-card-menu" title="管理" onClick={e => { e.stopPropagation(); setOpen(o => !o); }}>⋯</span>
    </span>
  );
}
