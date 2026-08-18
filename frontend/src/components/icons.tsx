/* ================= 统一 SVG 线性图标库 =================
   24 viewBox · 2px 描边 · currentColor 着色（随文字颜色变化） */
interface IcoProps { size?: number }

function Svg({ size = 18, children }: { size?: number; children: React.ReactNode }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor"
      strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden>
      {children}
    </svg>
  );
}

/* 导航 */
export function IconOverview({ size }: IcoProps) { return <Svg size={size}><circle cx="12" cy="12" r="10" /><circle cx="12" cy="12" r="4" /></Svg>; }
export function IconCluster({ size }: IcoProps) {
  return <Svg size={size}><rect x="2" y="2" width="20" height="8" rx="2" /><rect x="2" y="14" width="20" height="8" rx="2" /><line x1="6" y1="6" x2="6.01" y2="6" /><line x1="6" y1="18" x2="6.01" y2="18" /></Svg>;
}
export function IconHost({ size }: IcoProps) {
  return <Svg size={size}><rect x="2" y="3" width="20" height="14" rx="2" /><line x1="8" y1="21" x2="16" y2="21" /><line x1="12" y1="17" x2="12" y2="21" /></Svg>;
}
export function IconDashboard({ size }: IcoProps) {
  return <Svg size={size}><line x1="18" y1="20" x2="18" y2="10" /><line x1="12" y1="20" x2="12" y2="4" /><line x1="6" y1="20" x2="6" y2="14" /></Svg>;
}
export function IconChat({ size }: IcoProps) {
  return <Svg size={size}><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" /></Svg>;
}

/* 通用 */
export function IconSearch({ size = 15 }: IcoProps) {
  return <Svg size={size}><circle cx="11" cy="11" r="8" /><line x1="21" y1="21" x2="16.65" y2="16.65" /></Svg>;
}
export function IconBell({ size = 17 }: IcoProps) {
  return <Svg size={size}><path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9" /><path d="M13.73 21a2 2 0 0 1-3.46 0" /></Svg>;
}
export function IconRefresh({ size = 14 }: IcoProps) {
  return <Svg size={size}><polyline points="23 4 23 10 17 10" /><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10" /></Svg>;
}
export function IconBolt({ size = 14 }: IcoProps) {
  return <Svg size={size}><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2" /></Svg>;
}
export function IconChevronRight({ size = 12 }: IcoProps) {
  return <Svg size={size}><polyline points="9 18 15 12 9 6" /></Svg>;
}
export function IconChevronLeft({ size = 12 }: IcoProps) {
  return <Svg size={size}><polyline points="15 18 9 12 15 6" /></Svg>;
}
export function IconPlus({ size = 13 }: IcoProps) {
  return <Svg size={size}><line x1="12" y1="5" x2="12" y2="19" /><line x1="5" y1="12" x2="19" y2="12" /></Svg>;
}
export function IconClose({ size = 14 }: IcoProps) {
  return <Svg size={size}><line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" /></Svg>;
}
export function IconEye({ size = 15 }: IcoProps) {
  return <Svg size={size}><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" /><circle cx="12" cy="12" r="3" /></Svg>;
}
export function IconFunnel({ size = 12 }: IcoProps) {
  return <Svg size={size}><path d="M22 3H2l8 9.46V19l4 2v-8.54L22 3z" /></Svg>;
}
export function IconEdit({ size = 13 }: IcoProps) {
  return <Svg size={size}><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" /><path d="M18.5 2.5a2.12 2.12 0 0 1 3 3L12 15l-4 1 1-4Z" /></Svg>;
}
export function IconTrash({ size = 13 }: IcoProps) {
  return <Svg size={size}><polyline points="3 6 5 6 21 6" /><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" /></Svg>;
}
export function IconHistory({ size = 15 }: IcoProps) {
  return <Svg size={size}><path d="M3 3v5h5" /><path d="M3.05 13A9 9 0 1 0 6 5.3L3 8" /><polyline points="12 7 12 12 15 15" /></Svg>;
}
export function IconMonitor({ size = 20 }: IcoProps) {
  return <Svg size={size}><rect x="2" y="3" width="20" height="14" rx="2" /><line x1="8" y1="21" x2="16" y2="21" /><line x1="12" y1="17" x2="12" y2="21" /></Svg>;
}

/* 主机统计卡 */
export function IconAlertTriangle({ size = 20 }: IcoProps) {
  return <Svg size={size}><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" /><line x1="12" y1="9" x2="12" y2="13" /><line x1="12" y1="17" x2="12.01" y2="17" /></Svg>;
}
export function IconCpu({ size = 20 }: IcoProps) {
  return <Svg size={size}><path d="M12 20V10" /><path d="M18 20V4" /><path d="M6 20v-4" /></Svg>;
}
export function IconMemory({ size = 20 }: IcoProps) {
  return <Svg size={size}><path d="M21 12V7H5a2 2 0 0 1 0-4h14v4" /><path d="M3 5v14a2 2 0 0 0 2 2h16v-5" /><path d="M18 12a2 2 0 0 0 0 4h4v-4Z" /></Svg>;
}
export function IconDisk({ size = 20 }: IcoProps) {
  return <Svg size={size}><ellipse cx="12" cy="5" rx="9" ry="3" /><path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3" /><path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5" /></Svg>;
}
export function IconRobot({ size = 34 }: IcoProps) {
  return <Svg size={size}><rect x="4" y="8" width="16" height="12" rx="3" /><line x1="12" y1="8" x2="12" y2="3" /><circle cx="12" cy="3" r="1" /><circle cx="9" cy="14" r="1" /><circle cx="15" cy="14" r="1" /><line x1="9" y1="17" x2="15" y2="17" /></Svg>;
}
