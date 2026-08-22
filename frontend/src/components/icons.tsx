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
export function IconPlugin({ size }: IcoProps) {
  return <Svg size={size}><path d="M19.439 7.85c-.049.322.059.648.289.878l1.568 1.568c.47.47.706 1.087.706 1.704s-.235 1.233-.706 1.704l-1.611 1.611a.98.98 0 0 1-.837.276c-.47-.07-.802-.48-.968-.925a2.501 2.501 0 1 0-3.214 3.214c.446.166.855.497.925.968a.979.979 0 0 1-.276.837l-1.61 1.61a2.404 2.404 0 0 1-1.705.707 2.402 2.402 0 0 1-1.704-.706l-1.568-1.568a1.026 1.026 0 0 0-.877-.29c-.493.074-.84.504-1.02.968a2.5 2.5 0 1 1-3.237-3.237c.464-.18.894-.527.967-1.02a1.026 1.026 0 0 0-.289-.877l-1.568-1.568A2.402 2.402 0 0 1 1.998 12c0-.617.236-1.234.706-1.704L4.23 8.77c.24-.24.581-.353.917-.303.515.077.877.528 1.073 1.01a2.5 2.5 0 1 0 3.259-3.259c-.482-.196-.933-.558-1.01-1.073-.05-.336.062-.676.303-.917l1.525-1.525A2.402 2.402 0 0 1 12 1.998c.617 0 1.234.236 1.704.706l1.568 1.568c.23.23.556.338.877.29.493-.074.84-.504 1.02-.968a2.5 2.5 0 1 1 3.237 3.237c-.464.18-.894.527-.967 1.02Z" /></Svg>;
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
export function IconChevronDown({ size = 12 }: IcoProps) {
  return <Svg size={size}><polyline points="6 9 12 15 18 9" /></Svg>;
}
export function IconCheck({ size = 14 }: IcoProps) {
  return <Svg size={size}><polyline points="20 6 9 17 4 12" /></Svg>;
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

/* 左下角设置入口 / 关于 */
export function IconSettings({ size = 16 }: IcoProps) {
  return <Svg size={size}><circle cx="12" cy="12" r="3" /><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 1 1-4 0v-.09a1.65 1.65 0 0 0-1-1.51 1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 1 1 0-4h.09a1.65 1.65 0 0 0 1.51-1 1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33h.01a1.65 1.65 0 0 0 1-1.51V3a2 2 0 1 1 4 0v.09a1.65 1.65 0 0 0 1 1.51h.01a1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82v.01a1.65 1.65 0 0 0 1.51 1H21a2 2 0 1 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z" /></Svg>;
}
export function IconInfo({ size = 16 }: IcoProps) {
  return <Svg size={size}><circle cx="12" cy="12" r="10" /><line x1="12" y1="16" x2="12" y2="12" /><line x1="12" y1="8" x2="12.01" y2="8" /></Svg>;
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

/* 聊天输入框（ZCode 风格 composer） */
export function IconArrowUp({ size = 16 }: IcoProps) {
  return <Svg size={size}><line x1="12" y1="20" x2="12" y2="4" /><polyline points="5 11 12 4 19 11" /></Svg>;
}
export function IconStop({ size = 16 }: IcoProps) {
  return <Svg size={size}><rect x="6" y="6" width="12" height="12" rx="2" fill="currentColor" stroke="none" /></Svg>;
}
