import { Component, createContext, useContext, useEffect, useState } from 'react';
import type { ReactNode } from 'react';
import { HashRouter, Link, Navigate, Route, Routes, useLocation } from 'react-router-dom';
import Overview from './pages/Overview';
import Clusters from './pages/Clusters';
import ClusterDetail from './pages/ClusterDetail';
import InstanceDetail from './pages/InstanceDetail';
import Hosts from './pages/Hosts';
import Alerts from './pages/Alerts';
import Dashboards, { DashboardEntry } from './pages/Dashboards';
import DashboardView from './pages/DashboardView';
import TenantDetail from './pages/TenantDetail';
import ChatPage from './pages/ChatPage';
import { ChatPanel } from './components/ChatPanel';
import { ApiProvider, setQueryProvider } from './lib/query';
import { apiGet, withFallback } from './lib/api';
import { ToastHost } from './lib/toast';
import { IconOverview, IconCluster, IconHost, IconDashboard, IconChat, IconSearch, IconBell } from './components/icons';

/* 后端就绪：大盘查询切到 apiserver（失败自动回退 mock） */
setQueryProvider(ApiProvider);

/* 离线演示徽标：任一请求回退 mock 时显示，恢复后隐藏 */
function OfflineBadge() {
  const [offline, setOffline] = useState(false);
  useEffect(() => {
    let timer: any = null;
    const onOff = () => setOffline(true);
    const onOn = () => { if (timer) clearTimeout(timer); timer = setTimeout(() => setOffline(false), 1500); };
    window.addEventListener('api-offline', onOff);
    window.addEventListener('api-online', onOn);
    return () => {
      window.removeEventListener('api-offline', onOff);
      window.removeEventListener('api-online', onOn);
      if (timer) clearTimeout(timer);
    };
  }, []);
  if (!offline) return null;
  return <span className="offline-badge" title="apiserver 不可达，当前展示本地演示数据">离线演示数据</span>;
}

/* 全局错误边界：错误信息 + 重试 / 返回首页 */
class ErrorBoundary extends Component<{ children: ReactNode }, { error: Error | null }> {
  state = { error: null as Error | null };
  static getDerivedStateFromError(error: Error) { return { error }; }
  render() {
    if (this.state.error) {
      return (
        <div style={{ padding: 24, fontFamily: 'Menlo, monospace', whiteSpace: 'pre-wrap' }}>
          <h2 style={{ color: '#f53f3f' }}>页面渲染出错</h2>
          <div>{this.state.error.message}</div>
          <pre style={{ marginTop: 12, fontSize: 12, color: '#4e5d78', maxHeight: 200, overflow: 'auto' }}>{this.state.error.stack}</pre>
          <div style={{ display: 'flex', gap: 10, marginTop: 16 }}>
            <button className="btn sm primary" onClick={() => location.reload()}>↻ 重试</button>
            <a className="btn sm" href="#/overview" onClick={() => location.reload()}>返回首页</a>
          </div>
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
  { path: '/overview', Ico: IconOverview, label: '概览' },
  { path: '/clusters', Ico: IconCluster, label: '集群' },
  { path: '/hosts', Ico: IconHost, label: '主机' },
  { path: '/alerts', Ico: IconBell, label: '告警' },
  { path: '/dashboards', Ico: IconDashboard, label: '监控大盘' },
  { path: '/chat', Ico: IconChat, label: '智能对话' },
];

/* ---------- 左下角：compose 各服务 git 版本 ---------- */
interface SvcVersion { name: string; gitSHA: string; buildTime: string }

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
      <div className="svc-version" title="frontend 构建版本">
        <span className="svc-name">frontend</span>
        <span className="svc-sha">{fe?.gitSHA || 'unknown'}</span>
      </div>
      <div className="svc-version" title="apiserver 构建版本">
        <span className="svc-name">apiserver</span>
        <span className="svc-sha">{api?.gitSHA || 'unknown'}</span>
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

  /* 互斥：进入智能对话页自动收起侧边栏聊天抽屉（悬浮球在该页本就隐藏） */
  useEffect(() => {
    if (location.pathname === '/chat') setChatOpen(false);
  }, [location.pathname]);

  /* 全局「问 AI」入口：页面派发 open-chat-drawer 事件唤起抽屉 */
  useEffect(() => {
    const onOpen = () => setChatOpen(true);
    window.addEventListener('open-chat-drawer', onOpen);
    return () => window.removeEventListener('open-chat-drawer', onOpen);
  }, []);

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
                <span className="ico"><n.Ico size={17} /></span><span className="nav-label">{n.label}</span>
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
              <div className="search"><span className="ico"><IconSearch /></span><input placeholder="搜索集群 / 实例 / SQL…" /></div>
              <OfflineBadge />
              <Link className="icon-btn" to="/alerts" title="告警中心"><span className="bell"><IconBell /></span><i className="badge">{alertTotal}</i></Link>
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
                <Route path="/alerts" element={<Alerts />} />
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
        <ToastHost />
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
