import { useEffect, useRef, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { PanelView, ZoomOverlay } from './dashboardParts';
import { PanelEditor } from './PanelEditor';
import { ConfirmDialog } from '../components/dialogs';
import { blankPanel, getDashboard, updateDashboard, serverInit, serverUpdate } from '../lib/dashboards';
import type { Dashboard, Panel } from '../lib/types';
import { useBreadcrumb } from '../App';

const REFRESH_MS: Record<string, number> = { '10s': 10000, '30s': 30000, '1m': 60000, '5m': 300000 };

export default function DashboardView() {
  const { id } = useParams();
  const navigate = useNavigate();
  const [dash, setDash] = useState<Dashboard | null>(() => (id ? getDashboard(id) : null));
  const [editorFor, setEditorFor] = useState<{ panel: Panel; editing: boolean } | null>(null);
  const [zoomPanel, setZoomPanel] = useState<Panel | null>(null);
  const [delPanel, setDelPanel] = useState<Panel | null>(null);
  const [reloadKey, setReloadKey] = useState(0);
  const timerRef = useRef<any>(null);

  useBreadcrumb(
    dash
      ? [{ label: '首页' }, { label: '监控大盘', hash: '#/dashboards/list' }, { label: dash.title }]
      : [{ label: '首页' }, { label: '监控大盘' }],
  );

  useEffect(() => {
    let alive = true;
    serverInit().then(list => {
      if (!alive) return;
      const d = id ? list.find(x => x.id === id) : null;
      if (!d) { navigate('/dashboards', { replace: true }); return; }
      setDash(d);
    }).catch(() => {
      const d = id ? getDashboard(id) : null;
      if (!d) { navigate('/dashboards', { replace: true }); return; }
      setDash(d);
    });
    return () => { alive = false; };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id]);

  /* 自动刷新定时器 */
  useEffect(() => {
    if (timerRef.current) { clearInterval(timerRef.current); timerRef.current = null; }
    const ms = dash ? REFRESH_MS[dash.cfg.refresh] : undefined;
    if (ms) timerRef.current = setInterval(() => setReloadKey(k => k + 1), ms);
    return () => { if (timerRef.current) clearInterval(timerRef.current); };
  }, [dash?.cfg.refresh, dash]);

  if (!dash) return null;

  const persist = (patch: Partial<Dashboard>) => {
    const next = updateDashboard(dash.id, patch); // 本地镜像先行
    if (next) setDash(next);
    serverUpdate(dash.id, patch).catch(() => { /* 离线时本地镜像仍生效 */ });
  };

  const setCfg = (patch: Partial<Dashboard['cfg']>) => {
    const cfg = { ...dash.cfg, ...patch };
    persist({ cfg });
    if (patch.refresh !== undefined) setReloadKey(k => k + 1);
  };

  const visible = dash.panels.filter(p => p.visible);

  const savePanel = (draft: Panel) => {
    const editing = !!draft.id && dash.panels.some(p => p.id === draft.id);
    const panels = editing
      ? dash.panels.map(p => (p.id === draft.id ? { ...draft } : p))
      : [...dash.panels, { ...draft, id: 'p' + Date.now(), visible: true }];
    persist({ panels });
    setEditorFor(null);
    setReloadKey(k => k + 1);
  };

  return (
    <>
      <div className="page-title">
        <Link className="dash-back" to="/dashboards/list" title="返回大盘列表">‹</Link>
        <span className="dash-title-name">{dash.title}</span>
        <button className="btn sm" style={{ marginLeft: 12 }} onClick={() => navigate('/dashboards/list')}>☰ 查看大盘列表</button>
      </div>
      <div className="page-desc">自建大盘 · 复刻 Grafana 面板模型：多指标同轴叠加、双 Y 轴、序列级样式覆盖，配置经 apiserver 持久化（离线回退本地）</div>

      <div className="card dash-toolbar-card">
        <div className="dash-toolbar">
          <div className="select-wrap">
            <select value={dash.cfg.range} onChange={e => setCfg({ range: e.target.value })}>
              {['1h', '6h', '24h', '7d'].map(r => <option key={r} value={r}>近 {r}</option>)}
            </select>
          </div>
          <div className="select-wrap">
            <select value={dash.cfg.refresh} onChange={e => setCfg({ refresh: e.target.value })}>
              {[['0', '不自动刷新'], ['10s', '每 10s'], ['30s', '每 30s'], ['1m', '每 1m'], ['5m', '每 5m']].map(([v, l]) => <option key={v} value={v}>{l}</option>)}
            </select>
          </div>
          <label className="dash-checkbox">
            <input type="checkbox" checked={dash.cfg.compareYesterday} onChange={e => setCfg({ compareYesterday: e.target.checked })} /> 对比昨日
          </label>
          <button className="btn sm primary" onClick={() => setEditorFor({ panel: blankPanel(), editing: false })}>+ 添加面板</button>
          <span className="card-sub">{visible.length} 个面板 · 布局与配置保存在本地浏览器</span>
        </div>
      </div>

      <div style={{ marginTop: 14 }}>
        {!visible.length ? (
          <div className="card"><div className="empty" style={{ padding: '60px 0' }}>暂无面板，请点击上方「+ 添加面板」创建</div></div>
        ) : (
          <div className="dash-grid">
            {visible.map(p => (
              <PanelView
                key={`${p.id}-${reloadKey}`}
                panel={p}
                cfg={dash.cfg}
                reloadKey={reloadKey}
                onEdit={() => setEditorFor({ panel: p, editing: true })}
                onDelete={() => setDelPanel(p)}
                onZoom={() => setZoomPanel(p)}
              />
            ))}
          </div>
        )}
      </div>

      {editorFor && (
        <PanelEditor
          panel={editorFor.panel}
          cfg={dash.cfg}
          editing={editorFor.editing}
          onOk={savePanel}
          onClose={() => setEditorFor(null)}
        />
      )}

      {zoomPanel && <ZoomOverlay panel={zoomPanel} cfg={dash.cfg} onClose={() => setZoomPanel(null)} />}

      {delPanel && (
        <ConfirmDialog
          title="删除面板" okText="删除" danger
          message={`确定删除面板「${delPanel.title}」吗？该操作不可恢复。`}
          onOk={() => { persist({ panels: dash.panels.filter(x => x.id !== delPanel.id) }); setDelPanel(null); setReloadKey(k => k + 1); }}
          onClose={() => setDelPanel(null)}
        />
      )}
    </>
  );
}
