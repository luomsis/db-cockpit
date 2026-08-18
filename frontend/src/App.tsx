import { Component, createContext, useContext, useEffect, useState } from 'react';
import type { ReactNode } from 'react';
import { HashRouter, Link, Navigate, Route, Routes, useLocation } from 'react-router-dom';
import Overview from './pages/Overview';
import Clusters from './pages/Clusters';
import ClusterDetail from './pages/ClusterDetail';
import InstanceDetail from './pages/InstanceDetail';
import Hosts from './pages/Hosts';
import Dashboards, { DashboardEntry } from './pages/Dashboards';
import DashboardView from './pages/DashboardView';
import TenantDetail from './pages/TenantDetail';
import ChatPage from './pages/ChatPage';
import { ChatPanel } from './components/ChatPanel';
import { ApiProvider, setQueryProvider } from './lib/query';
import { apiGet, withFallback } from './lib/api';

/* 后端就绪：大盘查询切到 apiserver（失败自动回退 mock） */
setQueryProvider(ApiProvider);

/* 全局错误边界：运行时错误直接显示在页面上（开发期定位问题） */
class ErrorBoundary extends Component<{ children: ReactNode }, { error: Error | null }> {
  state = { error: null as Error | null };
  static getDerivedStateFromError(error: Error) { return { error }; }
  render() {
    if (this.state.error) {
      return (
        <div style={{ padding: 24, fontFamily: 'Menlo, monospace', whiteSpace: 'pre-wrap' }}>
          <h2 style={{ color: '#f53f3f' }}>页面渲染出错</h2>
          <div>{this.state.error.message}</div>
          <pre style={{ marginTop: 12, fontSize: 12, color: '#4e5d78' }}>{this.state.error.stack}</pre>
        </div>
      );
    }
    return this.props.children;
  }
}

/* ---------- 面包屑上下文 ---------- */
export interface CrumbItem { label: string; hash?: string; }
const CrumbCtx = createContext<{ items: CrumbItem[]; set: (i: CrumbItem[]) => void }>({
  items: [], set: () => { },
});

export function useBreadcrumb(items: CrumbItem[]) {
  const { set } = useContext(CrumbCtx);
  const key = JSON.stringify(items);
  useEffect(() => {
    set(JSON.parse(key));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key]);
}

const NAV = [
  { path: '/overview', ico: '◉', label: '概览' },
  { path: '/clusters', ico: '⛁', label: '集群' },
  { path: '/hosts', ico: '🖥', label: '主机' },
  { path: '/dashboards', ico: '📊', label: '监控大盘' },
  { path: '/chat', ico: '🤖', label: '智能对话' },
];

/* ---------- 左下角：compose 各服务 git 版本与构建时间 ---------- */
interface SvcVersion { name: string; gitSHA: string; buildTime: string }

function fmtTime(iso?: string): string {
  if (!iso || iso === 'unknown') return '';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '';
  const p = (n: number) => String(n).padStart(2, '0');
  return `${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`;
}

function ServiceVersions() {
  const [fe, setFe] = useState<SvcVersion | null>(null);
  const [api, setApi] = useState<SvcVersion | null>(null);
  useEffect(() => {
    fetch('/version.json').then(r => r.json()).then(setFe).catch(() => { /* 离线忽略 */ });
    fetch('/api/version')
      .then(r => r.json())
      .then((env: { code: number; data: SvcVersion }) => { if (env.code === 0) setApi(env.data); })
      .catch(() => { /* 离线忽略 */ });
  }, []);
  return (
    <div className="svc-versions">
      <div className="svc-version" title="frontend 构建信息">
        <span className="svc-name">frontend</span>
        <span className="svc-sha">{fe?.gitSHA || 'unknown'}</span>
        <span className="svc-time">{fmtTime(fe?.buildTime)}</span>
      </div>
      <div className="svc-version" title="apiserver 构建信息">
        <span className="svc-name">apiserver</span>
        <span className="svc-sha">{api?.gitSHA || 'unknown'}</span>
        <span className="svc-time">{fmtTime(api?.buildTime)}</span>
      </div>
    </div>
  );
}

function Shell() {
  const location = useLocation();
  const [items, setItems] = useState<CrumbItem[]>([{ label: '首页' }, { label: '运维概览' }]);
  const [chatOpen, setChatOpen] = useState(false);
  const [alertTotal, setAlertTotal] = useState(12); // 后端不可达时保持演示值

  /* 告警铃铛：真实告警实例数（mock 兜底） */
  useEffect(() => {
    let alive = true;
    withFallback(
      apiGet<{ items: unknown[]; total: number }>('/api/alerts'),
      () => ({ items: [], total: 12 }),
    ).then(r => { if (alive && r) setAlertTotal(r.total); }).catch(() => { /* keep */ });
    return () => { alive = false; };
  }, []);

  /* 路由切换时重置面包屑（页面挂载后会再设置） */
  useEffect(() => { setItems([{ label: '首页' }]); }, [location.pathname]);

  /* 面包屑不展示首位的「首页」 */
  const crumbs = items.filter((it, i) => !(i === 0 && it.label === '首页'));

  return (
    <CrumbCtx.Provider value={{ items, set: setItems }}>
      <div className="app">
        <aside className="sidebar">
          <div className="logo">
            <div className="logo-mark">DB</div>
            <div className="logo-text">
              <b>DB Cockpit</b>
              <span>数据库智能驾驶仓</span>
            </div>
          </div>
          <nav className="nav">
            {NAV.map(n => (
              <Link key={n.path} to={n.path}
                className={location.pathname === n.path || (n.path === '/dashboards' && location.pathname.startsWith('/dashboard')) ? 'active' : ''}>
                <span className="ico">{n.ico}</span>{n.label}
              </Link>
            ))}
          </nav>
          <div className="sidebar-foot">
            <div className="sf-env">
              <div className="env-dot"></div>
              <div>
                <div className="env-name">开发环境 DEV</div>
                <div className="env-sub">v0.1.0 · apiserver 已接线</div>
              </div>
            </div>
            <ServiceVersions />
          </div>
        </aside>

        <div className="main">
          <header className="topbar">
            <div className="crumb">
              {crumbs.map((it, i) => {
                const last = i === crumbs.length - 1;
                return (
                  <span key={i} style={{ display: 'inline-flex', gap: 8, alignItems: 'center' }}>
                    {i > 0 && <span className="crumb-sep">/</span>}
                    {!last && it.hash
                      ? <a href={it.hash}>{it.label}</a>
                      : !last ? <span>{it.label}</span>
                        : <span className="crumb-cur">{it.label}</span>}
                  </span>
                );
              })}
            </div>
            <div className="topbar-right">
              <div className="search"><span className="ico">⌕</span><input placeholder="搜索集群 / 实例 / SQL…" /></div>
              <button className="icon-btn" title="告警中心"><span className="bell">🔔</span><i className="badge">{alertTotal}</i></button>
              <div className="avatar">运</div>
            </div>
          </header>
          <main className="content">
            <ErrorBoundary>
              <Routes>
              <Route path="/overview" element={<Overview />} />
              <Route path="/clusters/*" element={<Clusters />} />
              <Route path="/cluster/:cid" element={<ClusterDetail />} />
              <Route path="/tenant/:cid/:tid" element={<TenantDetail />} />
              <Route path="/instance/:cid/:iid" element={<InstanceDetail />} />
              <Route path="/hosts" element={<Hosts />} />
              <Route path="/dashboards" element={<DashboardEntry />} />
              <Route path="/dashboards/list" element={<Dashboards />} />
              <Route path="/dashboard/:id" element={<DashboardView />} />
              <Route path="/dashboard" element={<Navigate to="/dashboards" replace />} />
              <Route path="/chat" element={<ChatPage />} />
              <Route path="*" element={<Navigate to="/overview" replace />} />
              </Routes>
            </ErrorBoundary>
          </main>
          {location.pathname !== '/chat' && !chatOpen && (
            <button className="chat-fab" title="打开智能对话" onClick={() => setChatOpen(true)}>
              <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor"
                strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden>
                <path d="M21 11.5a8.38 8.38 0 0 1-.9 3.8 8.5 8.5 0 0 1-7.6 4.7 8.38 8.38 0 0 1-3.8-.9L3 21l1.9-5.7a8.38 8.38 0 0 1-.9-3.8 8.5 8.5 0 0 1 4.7-7.6 8.38 8.38 0 0 1 3.8-.9h.5a8.48 8.48 0 0 1 8 8v.5z" />
              </svg>
            </button>
          )}
          {chatOpen && <ChatPanel onClose={() => setChatOpen(false)} />}
        </div>
      </div>
    </CrumbCtx.Provider>
  );
}

export default function App() {
  return (
    <HashRouter>
      <Shell />
    </HashRouter>
  );
}
