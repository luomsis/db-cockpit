/* ================= 核心类型定义 ================= */

export interface ParamItem { name: string; value: string; range: string; desc: string; status: string; }
export interface Instance {
  id: string; name: string; role: string; ip: string; port: number;
  status: 'ok' | 'warn' | 'err'; cpu: number; mem: number; conn: number; ver: string;
  zone?: string;               /* OceanBase OBServer 所属 Zone */
}
/* ---------- PostgreSQL 扩展 ---------- */
export interface PgDatabase {
  name: string; owner: string; size: string; tables: number;
  conn: number; connLimit: number; status: string;
}
export interface PgReplica {
  instance: string; role: string; delayMs: number; walLag: string; status: string;
}
/* ---------- OceanBase 租户体系 ---------- */
export interface ObUnit {
  zone: string; observer: string;
  maxCpu: number; usedCpu: number;    /* C 核 */
  maxMemGb: number; usedMemGb: number; /* GB */
}
export interface ObTenantDb { name: string; tables: number; size: string; conn: number; status: string; }
export interface ObTenant {
  id: string; name: string;
  kind: 'sys' | 'user';              /* 系统租户 / 用户租户 */
  mode: 'mysql' | 'oracle';          /* 租户兼容模式 */
  primaryZone: string;               /* PRIMARY_ZONE */
  locality: string;                  /* 副本分布描述 */
  unitNum: number;                   /* 每 Zone 的 Unit 数 */
  maxCpu: number; usedCpu: number;   /* 资源池合计（C） */
  maxMemGb: number; usedMemGb: number;
  storageUsed: string; storageTotal: string;
  units: ObUnit[];                   /* Unit 资源池分布 */
  databases: ObTenantDb[];           /* 租户内 Database */
  whitelist: string[];               /* 连接白名单 */
  connHint: string;                  /* 连接示例 */
  status: 'ok' | 'warn' | 'err';
}
export interface Cluster {
  id: string; name: string; type: string; version: string; desc: string;
  az: string;                         /* 可用区 */
  biz: string;                        /* 业务属性 */
  nodes: number; mode: string; cpu: number; mem: number; conn: number; qps: number;
  instances: Instance[]; params: ParamItem[];
  zones?: string[];                  /* OB：Zone 列表 */
  tenants?: ObTenant[];              /* OB：租户列表 */
  databases?: PgDatabase[];          /* PG：集群内 Database */
  replicas?: PgReplica[];            /* PG：备库复制状态 */
  syncMode?: string;                 /* PG：同步模式（Patroni/quorum 等） */
}

/* ---------- 监控大盘（对齐 Grafana 面板模型） ---------- */
export type PanelType = 'timeseries' | 'bar' | 'stat' | 'gauge' | 'table';
export interface PanelTarget {
  metric: string; name: string; color: string;
  axis: 'left' | 'right'; type: 'line' | 'bar' | 'area' | 'points';
  unit?: string; agg?: string; groupBy?: string; dbType?: string;
}
export interface ThresholdStep { value: number; color: string; }
export interface PanelStyle {
  lineWidth: number; fill: number; points: number; connectNulls: boolean;
  stack: 'none' | 'normal' | 'percent'; legend: 'top' | 'bottom' | 'right' | 'hide';
  yMin: number | null; yMax: number | null; decimals: number | null; unit: string;
}
export interface Panel {
  id: string; title: string; type: PanelType; legend: boolean; scope: string;
  visible: boolean; w: number; style: PanelStyle;
  thresholds: { steps: ThresholdStep[] };
  annotations: { enable: boolean; types: string[] };
  drilldown: { kind: 'cluster' | 'instance'; targetId: string } | null;
  targets: PanelTarget[];
}
export interface DashCfg { range: string; refresh: string; compareYesterday: boolean; }
export interface Dashboard {
  id: string; title: string; description: string; cfg: DashCfg;
  panels: Panel[]; createdAt: number; updatedAt: number;
}
export interface MetricDef { id: string; name: string; unit: string; base: number; jitter: number; }

/* ---------- 统一查询协议 ---------- */
export interface QueryRequest {
  metric: string; scope?: string; range: string; shift?: string;
  name?: string; color?: string; axis?: string; type?: string; unit?: string;
  agg?: string; groupBy?: string; dbType?: string;
}
export interface ResolvedTarget {
  name: string; data: number[]; color: string; unit: string;
  axis: 'left' | 'right'; type: string; dashed?: boolean;
}
export interface Annotation { time: string; title: string; type: 'release' | 'switch' | 'alert'; }

/* ---------- 卡片协议（card-protocol/1.0 MVP 子集） ---------- */
export type CardType = 'text' | 'metric_chart' | 'data_table' | 'task_progress' | 'diagnosis_report';
export interface Interaction {
  id: string; label: string;
  kind: 'ask' | 'drilldown' | 'open_link' | 'confirm_action' | 'cancel_task' | 'retry_task';
  payload?: Record<string, unknown>;
}
export interface CardEnvelope {
  card_id: string;
  card_type: CardType;
  protocol_version: string;
  title: string;
  status: 'streaming' | 'final';
  source?: { session_id?: string; turn_id?: string; agent?: string; tool_call_id?: string | null };
  context?: { instance_id?: number | string; cluster_id?: number | string; time_range?: { start: string; end: string } } | null;
  payload: any;
  interactions?: Interaction[];
  fallback_text: string;
}

/* metric_chart payload */
export interface MetricChartPayload {
  chart_type: 'line' | 'bar' | 'area';
  metrics: { name: string; label?: string; unit?: string; agg?: string }[];
  data: { points: [string, number][] };
  time_range?: { start: string; end: string };
  thresholds?: { value: number; label?: string; severity?: 'warning' | 'critical' }[];
  anomalies?: { start: string; end: string; note?: string }[];
}

/* data_table payload */
export interface DataTablePayload {
  columns: { key: string; label: string; type?: 'string' | 'number' | 'time' | 'duration' | 'sql' | 'status'; sortable?: boolean }[];
  rows: Record<string, any>[];
  total?: number;
  row_actions?: ('ask' | 'drilldown' | 'explain_sql')[];
}

/* task_progress payload */
export interface TaskProgressPayload {
  task_id: string; task_type: string;
  status: 'pending' | 'running' | 'done' | 'failed' | 'cancelled';
  progress?: number; stage?: string;
  stages?: { name: string; status: 'pending' | 'running' | 'done' | 'failed' | 'skipped' }[];
  eta_seconds?: number;
  result_card_id?: string;
  error?: string;
  cancellable?: boolean;
}

/* diagnosis_report payload */
export interface DiagnosisReportPayload {
  summary: string;
  severity: 'normal' | 'notice' | 'warning' | 'critical';
  root_causes?: { hypothesis: string; confidence: number; evidence_refs?: string[] }[];
  findings: { category: string; detail: string; metric?: string; evidence_card?: string }[];
  suggestions: string[];
  provider?: 'builtin' | 'vendor_agent';
  external_report_id?: string;
}

/* ---------- Chat 会话 ---------- */
export interface Thought { tool_name: string; status: 'running' | 'success' | 'failed'; }
export interface ChatMessage {
  id: string; role: 'user' | 'assistant';
  text: string; thoughts: Thought[]; cards: CardEnvelope[];
  status: 'streaming' | 'final' | 'error';
}
export interface ChatSession {
  id: string; title: string; createdAt: number; updatedAt: number;
  messages: ChatMessage[];
}

/* ---------- SSE 事件（对齐执行框架 §9，mock 侧） ---------- */
export type AgentEvent =
  | { type: 'thought'; step: number; tool_name: string; status: 'running' | 'success' | 'failed' }
  | { type: 'token'; text_delta: string }
  | { type: 'card'; card: CardEnvelope; mode: 'create' | 'update' }
  | { type: 'progress'; task_id: string; progress: number; stage: string }
  | { type: 'done' }
  | { type: 'error'; code: string; message: string };
