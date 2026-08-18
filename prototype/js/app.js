/* ================= DB Copilot · 前端原型 ================= */
/* ---------- Mock 数据 ---------- */
const DB_TYPES = [
  { type: 'mysql', name: 'MySQL', icon: '🐬', total: 46, alert: 6 },
  { type: 'pg', name: 'PostgreSQL', icon: '🐘', total: 28, alert: 3 },
  { type: 'oracle', name: 'Oracle', icon: '🔶', total: 12, alert: 2 },
  { type: 'redis', name: 'Redis', icon: '⚡', total: 30, alert: 1 },
  { type: 'mongodb', name: 'MongoDB', icon: '🍃', total: 14, alert: 0 },
  { type: 'sqlserver', name: 'SQL Server', icon: '🪟', total: 8, alert: 1 },
];

const TOP_ANOMALY = [
  { name: 'mysql-prod-order-01', cluster: 'prod-mysql-cluster-01', score: 96, issue: 'CPU 98% · 全表扫描激增', inst: 'in-2f8a1' },
  { name: 'pg-prod-report-02', cluster: 'prod-pg-cluster-01', score: 88, issue: '活跃会话堆积 · 锁等待', inst: 'in-9c3d7' },
  { name: 'mysql-prod-user-03', cluster: 'prod-mysql-cluster-01', score: 82, issue: '内存命中率跌至 71%', inst: 'in-7b2e9' },
  { name: 'redis-cache-01', cluster: 'prod-redis-cluster-01', score: 74, issue: '大 Key 访问 · 带宽打满', inst: 'in-e41a2' },
  { name: 'oracle-erp-01', cluster: 'prod-oracle-rac-01', score: 68, issue: '归档日志堆积 · IO 等待', inst: 'in-5d8c0' },
];

const SQL_ISSUES = [
  { name: '全表扫描', cnt: 23 },
  { name: '缺失索引', cnt: 18 },
  { name: '隐式类型转换', cnt: 12 },
  { name: '过度排序', cnt: 9 },
  { name: '深分页', cnt: 7 },
  { name: '冗余 JOIN', cnt: 5 },
];

const SLOW_SQLS = [
  { sql: 'SELECT o.*, u.name FROM trade_order o JOIN user u ON o.uid = u.id WHERE o.status = ?', db: 'trade_center', time: '12.8s', rows: '4,380,012', count: 342 },
  { sql: 'UPDATE stock_record SET qty = qty - ? WHERE sku_id = ? AND warehouse_id = ?', db: 'inventory', time: '9.6s', rows: '1,203,550', count: 187 },
  { sql: 'SELECT COUNT(*) FROM access_log WHERE create_time BETWEEN ? AND ? GROUP BY path', db: 'analytics', time: '8.2s', rows: '9,881,204', count: 96 },
  { sql: 'SELECT * FROM payment_bill WHERE bill_no LIKE ? ORDER BY ctime DESC LIMIT ?', db: 'payment', time: '6.4s', rows: '760,332', count: 64 },
  { sql: 'DELETE FROM session_token WHERE expire_at < ? AND app_id IN (?, ?, ?)', db: 'auth', time: '5.1s', rows: '2,310,778', count: 41 },
];

const CLUSTERS = [
  {
    id: 'c1', name: 'prod-mysql-cluster-01', type: 'mysql', version: 'MySQL 8.0.36',
    desc: '交易核心集群 · 华东-可用区A', nodes: 5, mode: '主从复制（1主3从 + MHA）',
    cpu: 62, mem: 71, conn: 1843, qps: 24600,
    instances: [
      { id: 'in-2f8a1', name: 'mysql-prod-order-01', role: '主实例', ip: '10.20.1.11', port: 3306, status: 'err', cpu: 98, mem: 86, conn: 512, ver: '8.0.36' },
      { id: 'in-8d4b2', name: 'mysql-prod-order-02', role: '从实例', ip: '10.20.1.12', port: 3306, status: 'ok', cpu: 41, mem: 63, conn: 210, ver: '8.0.36' },
      { id: 'in-7b2e9', name: 'mysql-prod-user-03', role: '从实例', ip: '10.20.1.13', port: 3306, status: 'warn', cpu: 72, mem: 88, conn: 480, ver: '8.0.36' },
      { id: 'in-3a6c5', name: 'mysql-prod-pay-04', role: '从实例', ip: '10.20.1.14', port: 3306, status: 'ok', cpu: 28, mem: 54, conn: 130, ver: '8.0.36' },
      { id: 'in-mha01', name: 'mha-manager-01', role: 'MHA', ip: '10.20.1.15', port: 22, status: 'ok', cpu: 8, mem: 22, conn: 12, ver: '0.58' },
    ],
    tenants: [
      { name: 'trade_center', inst: 2, cpu: 62, mem: 71, storage: '1.2 TB', status: 'ok' },
      { name: 'user_center', inst: 1, cpu: 72, mem: 88, storage: '680 GB', status: 'warn' },
      { name: 'payment', inst: 1, cpu: 28, mem: 54, storage: '420 GB', status: 'ok' },
    ],
    params: [
      { name: 'innodb_buffer_pool_size', value: '48G', range: '1G - 128G', desc: 'InnoDB 缓冲池大小', status: 'ok' },
      { name: 'max_connections', value: '2000', range: '100 - 10000', desc: '最大连接数', status: 'ok' },
      { name: 'slow_query_log', value: 'ON', range: 'ON / OFF', desc: '慢查询日志开关', status: 'ok' },
      { name: 'long_query_time', value: '1.0', range: '0.1 - 600', desc: '慢查询阈值（秒）', status: 'pending' },
      { name: 'innodb_flush_log_at_trx_commit', value: '1', range: '0 / 1 / 2', desc: '事务日志刷盘策略', status: 'ok' },
    ],
  },
  {
    id: 'c2', name: 'prod-pg-cluster-01', type: 'pg', version: 'PostgreSQL 15.6',
    desc: '报表分析集群 · 华东-可用区B', nodes: 3, mode: '流复制（1主2备 + Patroni）',
    cpu: 48, mem: 66, conn: 926, qps: 8400,
    instances: [
      { id: 'in-1e7f3', name: 'pg-prod-report-01', role: '主实例', ip: '10.20.2.11', port: 5432, status: 'ok', cpu: 45, mem: 62, conn: 320, ver: '15.6' },
      { id: 'in-9c3d7', name: 'pg-prod-report-02', role: '备实例', ip: '10.20.2.12', port: 5432, status: 'warn', cpu: 78, mem: 81, conn: 410, ver: '15.6' },
      { id: 'in-6b1a8', name: 'pg-prod-report-03', role: '备实例', ip: '10.20.2.13', port: 5432, status: 'ok', cpu: 33, mem: 58, conn: 196, ver: '15.6' },
    ],
    tenants: [
      { name: 'analytics', inst: 2, cpu: 51, mem: 69, storage: '2.8 TB', status: 'ok' },
      { name: 'bi_report', inst: 1, cpu: 78, mem: 81, storage: '1.1 TB', status: 'warn' },
    ],
    params: [
      { name: 'shared_buffers', value: '32G', range: '128MB - 128G', desc: '共享缓冲区大小', status: 'ok' },
      { name: 'max_connections', value: '1000', range: '1 - 262143', desc: '最大连接数', status: 'ok' },
      { name: 'work_mem', value: '64MB', range: '64kB - 2G', desc: '排序/哈希操作内存', status: 'pending' },
      { name: 'autovacuum', value: 'on', range: 'on / off', desc: '自动清理开关', status: 'ok' },
    ],
  },
  {
    id: 'c3', name: 'prod-redis-cluster-01', type: 'redis', version: 'Redis 7.2.4',
    desc: '缓存集群 · 华东-可用区A', nodes: 6, mode: 'Cluster（3主3从）',
    cpu: 35, mem: 58, conn: 3200, qps: 152000,
    instances: [
      { id: 'in-e41a2', name: 'redis-cache-01', role: '分片主', ip: '10.20.3.11', port: 6379, status: 'warn', cpu: 82, mem: 76, conn: 1100, ver: '7.2.4' },
      { id: 'in-e41a3', name: 'redis-cache-02', role: '分片主', ip: '10.20.3.12', port: 6379, status: 'ok', cpu: 31, mem: 55, conn: 900, ver: '7.2.4' },
      { id: 'in-e41a4', name: 'redis-cache-03', role: '分片主', ip: '10.20.3.13', port: 6379, status: 'ok', cpu: 27, mem: 49, conn: 820, ver: '7.2.4' },
      { id: 'in-e41b1', name: 'redis-cache-04', role: '分片从', ip: '10.20.3.14', port: 6379, status: 'ok', cpu: 18, mem: 52, conn: 200, ver: '7.2.4' },
      { id: 'in-e41b2', name: 'redis-cache-05', role: '分片从', ip: '10.20.3.15', port: 6379, status: 'ok', cpu: 15, mem: 48, conn: 110, ver: '7.2.4' },
      { id: 'in-e41b3', name: 'redis-cache-06', role: '分片从', ip: '10.20.3.16', port: 6379, status: 'ok', cpu: 16, mem: 47, conn: 70, ver: '7.2.4' },
    ],
    tenants: [
      { name: 'session_cache', inst: 3, cpu: 44, mem: 61, storage: '96 GB', status: 'ok' },
      { name: 'hot_data', inst: 3, cpu: 82, mem: 76, storage: '210 GB', status: 'warn' },
    ],
    params: [
      { name: 'maxmemory', value: '16gb', range: '0 - 512gb', desc: '最大内存限制', status: 'ok' },
      { name: 'maxmemory-policy', value: 'allkeys-lru', range: '8 种策略', desc: '内存淘汰策略', status: 'ok' },
      { name: 'appendonly', value: 'yes', range: 'yes / no', desc: 'AOF 持久化开关', status: 'ok' },
    ],
  },
  {
    id: 'c4', name: 'prod-oracle-rac-01', type: 'oracle', version: 'Oracle 19c',
    desc: 'ERP 核心集群 · 华北-可用区C', nodes: 2, mode: 'RAC 双节点',
    cpu: 55, mem: 74, conn: 460, qps: 5200,
    instances: [
      { id: 'in-5d8c0', name: 'oracle-erp-01', role: 'RAC 节点1', ip: '10.30.1.11', port: 1521, status: 'warn', cpu: 68, mem: 82, conn: 280, ver: '19.22' },
      { id: 'in-5d8c1', name: 'oracle-erp-02', role: 'RAC 节点2', ip: '10.30.1.12', port: 1521, status: 'ok', cpu: 42, mem: 66, conn: 180, ver: '19.22' },
    ],
    tenants: [
      { name: 'erp_core', inst: 2, cpu: 55, mem: 74, storage: '4.6 TB', status: 'ok' },
    ],
    params: [
      { name: 'sga_target', value: '64G', range: '1G - 512G', desc: 'SGA 目标大小', status: 'ok' },
      { name: 'processes', value: '2000', range: '6 - 65535', desc: '最大进程数', status: 'ok' },
      { name: 'db_files', value: '1024', range: '1 - 65535', desc: '最大数据文件数', status: 'pending' },
    ],
  },
  {
    id: 'c5', name: 'prod-oceanbase-01', type: 'oceanbase', version: 'OceanBase 4.2.1',
    desc: '核心账务集群 · 华东-可用区A/B/C', nodes: 6, mode: '3 Zone × 2 OBServer · Paxos',
    cpu: 58, mem: 69, conn: 1240, qps: 18600,
    instances: [
      { id: 'ob-a1', name: 'observer-zone1-01', role: 'OBServer · Zone1', ip: '10.40.1.11', port: 2881, status: 'ok', cpu: 61, mem: 72, conn: 420, ver: '4.2.1' },
      { id: 'ob-a2', name: 'observer-zone1-02', role: 'OBServer · Zone1', ip: '10.40.1.12', port: 2881, status: 'ok', cpu: 55, mem: 66, conn: 380, ver: '4.2.1' },
      { id: 'ob-b1', name: 'observer-zone2-01', role: 'OBServer · Zone2', ip: '10.40.2.11', port: 2881, status: 'warn', cpu: 78, mem: 83, conn: 300, ver: '4.2.1' },
      { id: 'ob-b2', name: 'observer-zone2-02', role: 'OBServer · Zone2', ip: '10.40.2.12', port: 2881, status: 'ok', cpu: 49, mem: 63, conn: 90, ver: '4.2.1' },
      { id: 'ob-c1', name: 'observer-zone3-01', role: 'OBServer · Zone3', ip: '10.40.3.11', port: 2881, status: 'ok', cpu: 52, mem: 68, conn: 30, ver: '4.2.1' },
      { id: 'ob-c2', name: 'observer-zone3-02', role: 'OBServer · Zone3', ip: '10.40.3.12', port: 2881, status: 'ok', cpu: 47, mem: 61, conn: 20, ver: '4.2.1' },
    ],
    tenants: [
      { name: 'sys', inst: 6, cpu: 12, mem: 18, storage: '40 GB', status: 'ok' },
      { name: 'trade_tenant', inst: 6, cpu: 64, mem: 71, storage: '2.1 TB', status: 'ok' },
      { name: 'pay_tenant', inst: 6, cpu: 78, mem: 91, storage: '860 GB', status: 'warn' },
    ],
    params: [
      { name: 'memory_limit', value: '110G', range: '0 - 物理内存', desc: 'OBServer 内存上限', status: 'ok' },
      { name: 'system_memory', value: '10G', range: '0 - memory_limit', desc: '系统预留内存', status: 'ok' },
      { name: 'cpu_count', value: '0', range: '0 - 核数', desc: '0 表示自动探测', status: 'ok' },
      { name: 'enable_syslog_recycle', value: 'true', range: 'true / false', desc: '系统日志自动回收', status: 'pending' },
    ],
  },
];

window.CLUSTERS = CLUSTERS; // 供 query.js resolveGroups 使用

const INSTANCE_USERS = [
  { user: 'app_rw', host: '10.20.%.%', priv: 'SELECT, INSERT, UPDATE', lastLogin: '2026-08-11 15:42', status: 'ok' },
  { user: 'app_ro', host: '10.20.%.%', priv: 'SELECT', lastLogin: '2026-08-11 15:39', status: 'ok' },
  { user: 'dba_admin', host: '%', priv: 'ALL PRIVILEGES', lastLogin: '2026-08-10 22:10', status: 'ok' },
  { user: 'report_etl', host: '10.21.0.%', priv: 'SELECT, LOAD', lastLogin: '2026-08-09 03:00', status: 'warn' },
  { user: 'tmp_debug', host: '10.99.1.5', priv: 'SELECT', lastLogin: '2026-07-28 11:20', status: 'err' },
];

const SESSIONS = [
  { id: 88231, user: 'app_rw', host: '10.20.4.21:40218', db: 'trade_center', cmd: 'Query', time: '28s', state: 'Sending data', lock: '行锁等待', status: 'err' },
  { id: 88190, user: 'app_rw', host: '10.20.4.22:40871', db: 'trade_center', cmd: 'Query', time: '12s', state: 'update', lock: '—', status: 'warn' },
  { id: 88102, user: 'report_etl', host: '10.21.0.8:38452', db: 'analytics', cmd: 'Query', time: '96s', state: 'Copying to tmp table', lock: '—', status: 'warn' },
  { id: 87955, user: 'app_ro', host: '10.20.5.10:41002', db: 'user_center', cmd: 'Sleep', time: '300s', state: '—', lock: '—', status: 'ok' },
  { id: 87901, user: 'app_rw', host: '10.20.4.21:40233', db: 'payment', cmd: 'Query', time: '3s', state: 'executing', lock: '—', status: 'ok' },
];

const TRANSACTIONS = [
  { id: 'TRX-998231', session: 88231, user: 'app_rw', dur: '1m 28s', undo: '14.2 MB', lockRows: '38,201', waiting: '是', sql: 'UPDATE stock_record SET qty = qty - ? …', status: 'err' },
  { id: 'TRX-998190', session: 88190, user: 'app_rw', dur: '42s', undo: '6.8 MB', lockRows: '12,044', waiting: '否', sql: 'INSERT INTO trade_order (…) VALUES (…)', status: 'warn' },
  { id: 'TRX-998102', session: 88102, user: 'report_etl', dur: '96s', undo: '0.2 MB', lockRows: '0', waiting: '否', sql: 'SELECT COUNT(*) FROM access_log …', status: 'ok' },
];

const REPORTS = [
  { ico: '📊', title: '性能周报 · 第 32 周', desc: '集群整体负载、TOP SQL、容量趋势', date: '2026-08-10 08:00', size: '2.4 MB' },
  { ico: '🔍', title: '慢 SQL 专项治理报告', desc: '本周新增 23 条慢 SQL，已优化 15 条', date: '2026-08-09 18:00', size: '1.1 MB' },
  { ico: '📈', title: '容量预测报告（8月）', desc: '存储预计 47 天后触达 85% 水位', date: '2026-08-08 08:00', size: '3.2 MB' },
  { ico: '🛡️', title: '巡检报告 · 每日', desc: '参数基线、备份校验、账号审计', date: '2026-08-11 06:00', size: '860 KB' },
];

/* ---------- 主机 Mock ---------- */
const HOSTS = [
  { ip: '10.20.1.11', zone: '华东-AZ-A', spec: '32C / 128G', os: 'CentOS 7.9', cpu: 92, mem: 86, disk: 71, insts: ['mysql-prod-order-01'], cluster: 'prod-mysql-cluster-01', cid: 'c1', status: 'err' },
  { ip: '10.20.1.12', zone: '华东-AZ-A', spec: '32C / 128G', os: 'CentOS 7.9', cpu: 45, mem: 63, disk: 64, insts: ['mysql-prod-order-02', 'mysql-prod-pay-04'], cluster: 'prod-mysql-cluster-01', cid: 'c1', status: 'ok' },
  { ip: '10.20.2.11', zone: '华东-AZ-B', spec: '16C / 64G', os: 'CentOS 7.9', cpu: 48, mem: 62, disk: 58, insts: ['pg-prod-report-01'], cluster: 'prod-pg-cluster-01', cid: 'c2', status: 'ok' },
  { ip: '10.20.3.11', zone: '华东-AZ-A', spec: '16C / 64G', os: 'Ubuntu 22.04', cpu: 82, mem: 76, disk: 55, insts: ['redis-cache-01', 'redis-cache-04'], cluster: 'prod-redis-cluster-01', cid: 'c3', status: 'warn' },
  { ip: '10.30.1.11', zone: '华北-AZ-C', spec: '64C / 256G', os: 'RHEL 8.6', cpu: 68, mem: 82, disk: 91, insts: ['oracle-erp-01'], cluster: 'prod-oracle-rac-01', cid: 'c4', status: 'warn' },
  { ip: '10.40.1.11', zone: '华东-AZ-A', spec: '64C / 256G', os: 'CentOS 7.9', cpu: 61, mem: 72, disk: 66, insts: ['observer-zone1-01', 'observer-zone1-02'], cluster: 'prod-oceanbase-01', cid: 'c5', status: 'ok' },
  { ip: '10.40.2.11', zone: '华东-AZ-B', spec: '64C / 256G', os: 'CentOS 7.9', cpu: 78, mem: 83, disk: 69, insts: ['observer-zone2-01', 'observer-zone2-02'], cluster: 'prod-oceanbase-01', cid: 'c5', status: 'ok' },
];

const LINE_COLORS = ['#006aff', '#00b365', '#ff9500', '#7a5af8', '#00a3e0', '#f53f3f'];

const STATUS_MAP = { ok: ['ok', '正常'], warn: ['warn', '警告'], err: ['err', '异常'], info: ['info', '提示'] };
const TYPE_ICON = { mysql: '🐬', pg: '🐘', oracle: '🔶', redis: '⚡', mongodb: '🍃', sqlserver: '🪟', oceanbase: '🌊', tidb: '🧰' };
const EXTRA_TYPE_NAME = { oceanbase: 'OceanBase', tidb: 'TiDB' };

/* ---------- 工具函数 ---------- */
const $ = (s, p) => (p || document).querySelector(s);
const pill = (st, text) => `<span class="pill ${STATUS_MAP[st][0]}"><i></i>${text || STATUS_MAP[st][1]}</span>`;
const typeTag = (t) => {
  const d = DB_TYPES.find(x => x.type === t);
  const name = d ? d.name : (EXTRA_TYPE_NAME[t] || t);
  return `<span class="tag ${t}">${name}</span>`;
};

const chartInstances = [];
function mkChart(el, option) {
  const c = echarts.init(el);
  c.setOption(option);
  chartInstances.push(c);
  return c;
}
const axisStyle = {
  axisLine: { lineStyle: { color: '#d9dee8' } },
  axisLabel: { color: '#8a97ad', fontSize: 11 },
  splitLine: { lineStyle: { color: '#eef1f8' } },
};
const TIP = { backgroundColor: '#fff', borderColor: '#e5e9f2', textStyle: { color: '#1d2b45', fontSize: 12 }, extraCssText: 'box-shadow:0 4px 16px rgba(29,43,69,0.12)' };
function lineOpt(title, xs, series, unit) {
  return {
    tooltip: { trigger: 'axis', ...TIP },
    grid: { left: 42, right: 14, top: 30, bottom: 24 },
    title: { text: title, textStyle: { color: '#4e5d78', fontSize: 12, fontWeight: 500 }, left: 4, top: 2 },
    xAxis: { type: 'category', data: xs, boundaryGap: false, ...axisStyle },
    yAxis: { type: 'value', ...axisStyle, axisLabel: { ...axisStyle.axisLabel, formatter: `{value}${unit || ''}` } },
    series: series.map(s => ({
      type: 'line', smooth: true, symbol: 'none', data: s.data, name: s.name,
      lineStyle: { width: 2, color: s.color },
      areaStyle: { color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
        { offset: 0, color: s.color + '44' }, { offset: 1, color: s.color + '05' }]) },
    })),
  };
}
/* 格式化 Y 轴标签：小数位 + 单位 */
function axisFmt(unit, decimals) {
  return (val) => {
    const num = decimals != null ? Number(val).toFixed(decimals) : val;
    return `${num}${unit || ''}`;
  };
}

/* Grafana 风格时序图：多指标共享时间轴 + 双 Y 轴 + 序列级样式覆盖 + 面板级样式
 * targets: [{ name, data, color, axis, type, unit, dashed }]
 * cfg: { legend, thresholds: [{value,color}], annotations: [{time,title,type}], style: {...} } */
function tsChartOpt(title, xs, targets, cfg) {
  const st = Object.assign({}, DEFAULT_PANEL_STYLE, cfg.style || {});
  const hasRight = targets.some(t => t.axis === 'right');
  const unitOvr = st.unit || '';
  let leftUnit = unitOvr, rightUnit = unitOvr;
  if (!unitOvr) {
    targets.forEach(t => {
      if (t.axis === 'right') { if (!rightUnit) rightUnit = t.unit; }
      else if (!leftUnit) leftUnit = t.unit;
    });
  }
  const isStack = st.stack === 'normal' || st.stack === 'percent';
  const isPercent = st.stack === 'percent';
  const series = targets.map(t => {
    const lw = st.lineWidth || 2;
    const s = {
      name: t.name, data: t.data, yAxisIndex: t.axis === 'right' ? 1 : 0,
      type: 'line', smooth: true, symbol: 'none', connectNulls: st.connectNulls,
      lineStyle: { width: lw, color: t.color }, itemStyle: { color: t.color },
    };
    if (t.dashed) s.lineStyle = Object.assign({}, s.lineStyle, { type: 'dashed' });
    /* 序列级类型 */
    if (t.type === 'bar') {
      s.type = 'bar'; s.barWidth = 10; s.smooth = undefined; s.lineStyle = undefined;
      s.itemStyle = { color: t.color, borderRadius: [3, 3, 0, 0] };
    } else if (t.type === 'points') {
      s.symbol = 'circle'; s.symbolSize = st.points > 0 ? st.points : 5;
    }
    /* 面板级填充：line/area 且 fill > 0 时叠加面积 */
    if (st.fill > 0 && t.type !== 'bar') {
      const op = Math.round(st.fill / 100 * 255).toString(16).padStart(2, '0');
      s.areaStyle = { color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
        { offset: 0, color: t.color + op }, { offset: 1, color: t.color + '05' }]) };
    } else if (t.type === 'area') {
      s.areaStyle = { color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
        { offset: 0, color: t.color + '44' }, { offset: 1, color: t.color + '05' }]) };
    }
    /* 面板级点 */
    if (st.points > 0 && t.type !== 'bar' && t.type !== 'points') {
      s.symbol = 'circle'; s.symbolSize = st.points;
    }
    /* 堆叠 */
    if (isStack && t.type !== 'bar') s.stack = isPercent ? 'percent' : 'total';
    if (isStack && t.type === 'bar') s.stack = isPercent ? 'percent' : 'total';
    return s;
  });
  const yAxisBase = {
    type: 'value', ...axisStyle,
    axisLabel: { ...axisStyle.axisLabel, formatter: axisFmt(leftUnit, st.decimals) },
  };
  if (st.yMin != null && st.yMin !== '') yAxisBase.min = +st.yMin;
  if (st.yMax != null && st.yMax !== '') yAxisBase.max = +st.yMax;
  if (isPercent) yAxisBase.max = 100;
  const yAxes = [yAxisBase];
  if (hasRight) {
    const yr = {
      type: 'value', position: 'right', ...axisStyle, splitLine: { show: false },
      axisLabel: { ...axisStyle.axisLabel, formatter: axisFmt(rightUnit, st.decimals) },
    };
    if (isPercent) yr.max = 100;
    yAxes.push(yr);
  }
  /* 图例位置 */
  const legPos = st.legend || 'top';
  const showLegend = cfg.legend !== false && legPos !== 'hide';
  const legCfg = { show: showLegend, textStyle: { color: '#4e5d78', fontSize: 11 }, itemWidth: 12, itemHeight: 8 };
  const grid = { left: 52, right: hasRight ? 52 : 14, top: 12, bottom: 24 };
  if (showLegend) {
    if (legPos === 'bottom') { legCfg.bottom = 0; grid.bottom = 36; }
    else if (legPos === 'right') { legCfg.right = 0; legCfg.top = 'center'; legCfg.orient = 'vertical'; grid.right = 100; }
    else { legCfg.top = 0; grid.top = 30; } // top
  }
  const option = {
    tooltip: { trigger: 'axis', ...TIP },
    legend: legCfg,
    grid,
    title: { text: title, textStyle: { color: '#4e5d78', fontSize: 12, fontWeight: 500 }, left: 4, top: 2 },
    xAxis: { type: 'category', data: xs, boundaryGap: false, ...axisStyle },
    yAxis: yAxes,
    series,
  };

  /* 阈值：Grafana steps 语义（升序分段着色）+ 阈值虚线 */
  const mkData = [];
  const steps = (cfg.thresholds || []).slice().sort((a, b) => a.value - b.value);
  if (steps.length) {
    const pieces = [{ lt: steps[0].value, color: '#8a97ad' }];
    steps.forEach((s, i) => pieces.push({
      gte: s.value, lt: i + 1 < steps.length ? steps[i + 1].value : Infinity, color: s.color,
    }));
    option.visualMap = { show: false, dimension: 1, pieces };
    steps.forEach(s => mkData.push({ yAxis: s.value, lineStyle: { color: s.color, width: 1, type: 'dashed' } }));
  }

  /* 事件标注：发布/切换/告警叠加为时间轴竖线 */
  const annColors = { release: '#006aff', switch: '#ff9500', alert: '#f53f3f' };
  (cfg.annotations || []).forEach(a => {
    mkData.push({
      xAxis: a.time, title: a.title,
      lineStyle: { color: annColors[a.type] || '#8a97ad', width: 1.5 },
      label: { show: true, formatter: (p) => p.data.title, position: 'insideEndTop', color: annColors[a.type] || '#8a97ad', fontSize: 10 },
    });
  });
  if (mkData.length && series.length) series[0].markLine = { silent: true, symbol: 'none', data: mkData };
  return option;
}

/* genSeries / HOURS / rangeTicks / rangeSpike 已迁入 query.js（统一查询数据层） */

/* ---------- 侧边栏导航 ---------- */
const NAV = [
  { hash: '#/overview', ico: '◉', label: '概览' },
  { hash: '#/clusters', ico: '⛁', label: '集群' },
  { hash: '#/hosts', ico: '🖥', label: '主机' },
  { hash: '#/dashboards', ico: '📊', label: '监控大盘' },
];
function renderNav(activeHash) {
  $('#nav').innerHTML = NAV.map(n => n.group
    ? `<div class="nav-group">${n.group}</div>`
    : `<a href="${n.disabled ? 'javascript:void(0)' : n.hash}" data-hash="${n.hash}"
         class="${!n.disabled && (activeHash.startsWith(n.hash) || (n.hash === '#/dashboards' && activeHash.startsWith('#/dashboard'))) ? 'active' : ''}"
         ${n.disabled ? 'style="opacity:.45;cursor:not-allowed"' : ''}>
         <span class="ico">${n.ico}</span>${n.label}</a>`).join('');
}

/* ---------- 面包屑 ---------- */
function setCrumb(items) {
  $('#crumb').innerHTML = items.map((it, i) => {
    const last = i === items.length - 1;
    const txt = it.hash ? `<a href="${it.hash}">${it.label}</a>` : it.label;
    return `${i ? '<span class="crumb-sep">/</span>' : ''}${last ? `<span class="crumb-cur">${it.label}</span>` : txt}`;
  }).join('');
}

/* ================= 页面：运维概览 ================= */
function renderOverview() {
  setCrumb([{ label: '首页' }, { label: '运维概览' }]);
  const totalInst = DB_TYPES.reduce((a, b) => a + b.total, 0);
  const totalAlert = DB_TYPES.reduce((a, b) => a + b.alert, 0);

  $('#content').innerHTML = `
  <div class="page-title">运维概览 <span class="pill info"><i></i>实时 · 每 15s 刷新</span></div>
  <div class="page-desc">数据库智能运维平台 · 全局健康视图（数据截至 ${new Date().toLocaleTimeString('zh-CN', { hour12: false })}）</div>
  <div class="ov-grid">

    <!-- 卡片1：数据库类型及告警概览 -->
    <div class="card c1">
      <div class="card-head">
        <div class="card-title"><span class="t-ico"></span>数据库类型及告警概览</div>
        <a class="card-link" href="#/clusters">查看全部 ›</a>
      </div>
      <div class="stat-row">
        <div class="stat"><div class="num ok">${DB_TYPES.length}</div><div class="lbl">数据库类型</div></div>
        <div class="stat"><div class="num warn">${totalAlert}</div><div class="lbl">告警实例数</div></div>
        <div class="stat"><div class="num">${totalInst}</div><div class="lbl">总实例数</div></div>
      </div>
      <div id="chart-dbtype" class="chart-box"></div>
    </div>

    <!-- 卡片2：性能异常实例 TOP -->
    <div class="card c2">
      <div class="card-head">
        <div class="card-title"><span class="t-ico"></span>性能异常实例 TOP5</div>
        <span class="card-sub">健康分越低越危险</span>
      </div>
      <div class="rank-list">
        ${TOP_ANOMALY.map((t, i) => `
        <div class="rank-item" onclick="location.hash='#/clusters'">
          <span class="rank-no ${i < 2 ? 'hot' : ''}">${i + 1}</span>
          <div class="rank-main">
            <div class="rank-name">${t.name}</div>
            <div class="rank-meta">${t.cluster} · ${t.issue}</div>
          </div>
          <span class="rank-score ${t.score >= 90 ? 'score-danger' : t.score >= 75 ? 'score-warn' : 'score-ok'}">${t.score}</span>
        </div>`).join('')}
      </div>
    </div>

    <!-- 卡片3：SQL 性能问题概览 -->
    <div class="card c3">
      <div class="card-head">
        <div class="card-title"><span class="t-ico"></span>SQL 性能问题概览</div>
        <span class="card-sub">近 24 小时</span>
      </div>
      <div id="chart-sqlissue" class="chart-box lg" style="height:252px"></div>
    </div>

    <!-- 卡片4：锁分析 -->
    <div class="card c4">
      <div class="card-head">
        <div class="card-title"><span class="t-ico"></span>锁分析</div>
        <span class="card-sub">全局锁态势</span>
      </div>
      <div class="lock-flex">
        <div class="lock-gauge"><div id="chart-lockgauge" style="height:170px"></div></div>
        <div class="lock-right">
          <div class="lock-kv"><span class="k">当前锁等待会话</span><span class="v" style="color:var(--red)">7</span></div>
          <div class="lock-kv"><span class="k">今日死锁次数</span><span class="v" style="color:var(--amber)">2</span></div>
          <div class="lock-kv"><span class="k">最长锁等待</span><span class="v">1m 28s · TRX-998231</span></div>
          <div class="lock-kv"><span class="k">元数据锁（MDL）阻塞</span><span class="v">1</span></div>
          <div class="lock-kv"><span class="k">热点争用表</span><span class="v">stock_record</span></div>
        </div>
      </div>
    </div>

    <!-- 卡片5：慢 SQL -->
    <div class="card c5">
      <div class="card-head">
        <div class="card-title"><span class="t-ico"></span>慢 SQL TOP5</div>
        <a class="card-link">诊断 ›</a>
      </div>
      <div class="sql-list">
        ${SLOW_SQLS.map(s => `
        <div class="sql-item">
          <div class="sql-text">${s.sql}</div>
          <div class="sql-meta">
            <span>库 <b>${s.db}</b></span>
            <span>耗时 <b style="color:var(--amber)">${s.time}</b></span>
            <span>扫描 <b>${s.rows}</b> 行</span>
            <span>次数 <b>${s.count}</b></span>
          </div>
        </div>`).join('')}
      </div>
    </div>
  </div>`;

  /* 卡片1 图表：类型分布 + 告警 */
  mkChart($('#chart-dbtype'), {
    tooltip: { trigger: 'axis', ...TIP },
    legend: { data: ['实例数', '告警数'], textStyle: { color: '#4e5d78', fontSize: 11 }, top: 0, right: 0, itemWidth: 12, itemHeight: 8 },
    grid: { left: 36, right: 10, top: 30, bottom: 24 },
    xAxis: { type: 'category', data: DB_TYPES.map(d => d.name), ...axisStyle },
    yAxis: { type: 'value', ...axisStyle },
    series: [
      { name: '实例数', type: 'bar', barWidth: 14, itemStyle: { borderRadius: [4, 4, 0, 0], color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [{ offset: 0, color: '#338bff' }, { offset: 1, color: '#006aff' }]) }, data: DB_TYPES.map(d => d.total) },
      { name: '告警数', type: 'bar', barWidth: 14, itemStyle: { borderRadius: [4, 4, 0, 0], color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [{ offset: 0, color: '#ffb056' }, { offset: 1, color: '#ff9500' }]) }, data: DB_TYPES.map(d => d.alert) },
    ],
  });

  /* 卡片3 图表：SQL 问题分类 */
  mkChart($('#chart-sqlissue'), {
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' }, ...TIP },
    grid: { left: 92, right: 30, top: 8, bottom: 8 },
    xAxis: { type: 'value', ...axisStyle, splitLine: { lineStyle: { color: '#eef1f8' } } },
    yAxis: { type: 'category', data: SQL_ISSUES.map(s => s.name).reverse(), ...axisStyle, axisLabel: { color: '#4e5d78', fontSize: 11.5 } },
    series: [{
      type: 'bar', barWidth: 12, data: SQL_ISSUES.map(s => s.cnt).reverse(),
      label: { show: true, position: 'right', color: '#006aff', fontSize: 11 },
      itemStyle: { borderRadius: [0, 4, 4, 0], color: new echarts.graphic.LinearGradient(1, 0, 0, 0, [{ offset: 0, color: '#00a3e0' }, { offset: 1, color: '#006aff' }]) },
    }],
  });

  /* 卡片4 图表：锁等待率仪表 */
  mkChart($('#chart-lockgauge'), {
    series: [{
      type: 'gauge', startAngle: 210, endAngle: -30, min: 0, max: 100,
      radius: '98%', center: ['50%', '58%'],
      progress: { show: true, width: 12, itemStyle: { color: new echarts.graphic.LinearGradient(0, 0, 1, 0, [{ offset: 0, color: '#006aff' }, { offset: 1, color: '#f53f3f' }]) } },
      axisLine: { lineStyle: { width: 12, color: [[1, '#eef1f8']] } },
      axisTick: { show: false }, splitLine: { show: false },
      axisLabel: { show: false }, pointer: { show: false },
      detail: { valueAnimation: true, formatter: '{value}%', color: '#ff9500', fontSize: 22, fontWeight: 700, offsetCenter: [0, '12%'] },
      title: { show: true, offsetCenter: [0, '48%'], color: '#8a97ad', fontSize: 11 },
      data: [{ value: 38, name: '锁等待率' }],
    }],
  });
}

/* ================= 页面：集群列表 ================= */
function renderClusters(focusClusterId) {
  setCrumb([{ label: '首页' }, { label: '集群管理' }]);
  $('#content').innerHTML = `
  <div class="page-title">集群管理</div>
  <div class="page-desc">共 ${CLUSTERS.length} 个集群 · ${CLUSTERS.reduce((a, c) => a + c.instances.length, 0)} 个实例，选择集群可下拉展开实例列表</div>
  <div class="toolbar">
    <div class="toolbar-left">
      <div class="select-wrap">
        <select id="clusterSelect">
          <option value="">— 选择集群查看实例 —</option>
          ${CLUSTERS.map(c => `<option value="${c.id}">${c.name}（${c.instances.length} 实例）</option>`).join('')}
        </select>
      </div>
      <button class="btn" id="btnExpandAll">全部展开</button>
      <button class="btn" id="btnCollapseAll">全部收起</button>
    </div>
    <button class="btn primary">＋ 接入新集群</button>
  </div>
  <div class="cluster-list" id="clusterList">
    ${CLUSTERS.map(c => clusterCardHTML(c)).join('')}
  </div>`;

  $('#clusterSelect').onchange = (e) => {
    const id = e.target.value;
    if (!id) return;
    const card = $(`#cluster-${id}`);
    card.classList.add('open');
    card.scrollIntoView({ behavior: 'smooth', block: 'center' });
  };
  $('#btnExpandAll').onclick = () => $$('.cluster-card').forEach(el => el.classList.add('open'));
  $('#btnCollapseAll').onclick = () => $$('.cluster-card').forEach(el => el.classList.remove('open'));

  if (focusClusterId) {
    const card = $(`#cluster-${focusClusterId}`);
    if (card) { card.classList.add('open'); setTimeout(() => card.scrollIntoView({ block: 'center' }), 60); }
    const sel = $('#clusterSelect'); if (sel) sel.value = focusClusterId;
  }
}
function clusterCardHTML(c) {
  return `
  <div class="card cluster-card" id="cluster-${c.id}">
    <div class="cluster-row" data-toggle="${c.id}">
      <div class="cluster-icon">${TYPE_ICON[c.type]}</div>
      <div class="cluster-info">
        <div class="cluster-name">${c.name} ${typeTag(c.type)} ${pill('ok')}</div>
        <div class="cluster-meta">${c.version} · ${c.mode} · ${c.desc}</div>
      </div>
      <div class="cluster-stats">
        <div class="cs-item"><div class="n">${c.nodes}</div><div class="l">节点数</div></div>
        <div class="cs-item"><div class="n">${c.instances.length}</div><div class="l">实例数</div></div>
        <div class="cs-item"><div class="n">${c.cpu}%</div><div class="l">CPU</div></div>
        <div class="cs-item"><div class="n">${c.mem}%</div><div class="l">内存</div></div>
        <div class="cs-item"><div class="n">${(c.qps / 1000).toFixed(1)}k</div><div class="l">QPS</div></div>
      </div>
      <a class="btn sm" href="#/cluster/${c.id}" onclick="event.stopPropagation()">集群详情</a>
      <div class="cluster-toggle">▾</div>
    </div>
    <div class="inst-panel">
      <table class="tbl">
        <thead><tr><th>实例名</th><th>角色</th><th>地址</th><th>版本</th><th>CPU</th><th>内存</th><th>连接数</th><th>状态</th><th>操作</th></tr></thead>
        <tbody>
          ${c.instances.map(inst => `
          <tr>
            <td class="mono">${inst.name}</td>
            <td>${inst.role}</td>
            <td class="mono">${inst.ip}:${inst.port}</td>
            <td>${inst.ver}</td>
            <td><div class="bar"><i class="${inst.cpu > 80 ? 'hot' : ''}" style="width:${inst.cpu}%"></i></div></td>
            <td><div class="bar"><i class="${inst.mem > 85 ? 'hot' : ''}" style="width:${inst.mem}%"></i></div></td>
            <td>${inst.conn}</td>
            <td>${pill(inst.status)}</td>
            <td><a href="#/instance/${c.id}/${inst.id}">实例详情</a></td>
          </tr>`).join('')}
        </tbody>
      </table>
    </div>
  </div>`;
}

/* ================= 通用：拓扑图 SVG ================= */
function topoSVG(cluster) {
  const W = 760, H = 300;
  const insts = cluster.instances;
  let masters = insts.filter(i => /主|节点1|分片主/.test(i.role));
  let slaves = insts.filter(i => !/主|节点1|分片主/.test(i.role));
  if (!masters.length) { // OBServer 等对等节点：上下两排展示
    masters = insts.slice(0, Math.ceil(insts.length / 2));
    slaves = insts.slice(Math.ceil(insts.length / 2));
  }
  const statusColor = { ok: '#00b365', warn: '#ff9500', err: '#f53f3f' };

  const masterY = 90, slaveY = 220;
  const mNodes = masters.map((m, i) => ({
    ...m, x: W / 2 - ((masters.length - 1) * 130) / 2 + i * 130, y: masterY,
  }));
  const sNodes = slaves.map((s, i) => ({
    ...s, x: W / 2 - ((slaves.length - 1) * 120) / 2 + i * 120, y: slaveY,
  }));

  const appNode = { x: W / 2, y: 28 };
  let edges = '';
  mNodes.forEach(m => {
    edges += `<line x1="${appNode.x}" y1="${appNode.y + 16}" x2="${m.x}" y2="${m.y - 26}" stroke="url(#lgEdge)" stroke-width="1.6" stroke-dasharray="5 4" opacity="0.75"/>`;
  });
  sNodes.forEach((s, i) => {
    const m = mNodes[i % Math.max(1, mNodes.length)] || mNodes[0];
    if (m) edges += `<line x1="${m.x}" y1="${m.y + 26}" x2="${s.x}" y2="${s.y - 26}" stroke="rgba(0,106,255,0.35)" stroke-width="1.4"/>`;
  });

  const nodeHTML = (n, isMaster) => `
  <a href="#/instance/${cluster.id}/${n.id}" class="topo-node">
    <circle class="halo" cx="${n.x}" cy="${n.y}" r="30" fill="${statusColor[n.status]}" opacity="0.09"/>
    <circle class="main" cx="${n.x}" cy="${n.y}" r="22" fill="#ffffff" stroke="${statusColor[n.status]}" stroke-width="2"/>
    <text x="${n.x}" y="${n.y + 4}" text-anchor="middle" font-size="13" fill="#1d2b45">${isMaster ? '主' : '从'}</text>
    <circle cx="${n.x + 16}" cy="${n.y - 16}" r="4" fill="${statusColor[n.status]}"/>
    <text x="${n.x}" y="${n.y + 42}" text-anchor="middle" font-size="10.5" fill="#4e5d78">${n.name}</text>
    <text x="${n.x}" y="${n.y + 56}" text-anchor="middle" font-size="9.5" fill="#8a97ad">${n.role} · ${n.ip}</text>
  </a>`;

  return `
  <svg viewBox="0 0 ${W} ${H}" width="100%" height="${H}">
    <defs>
      <linearGradient id="lgEdge" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0" stop-color="#006aff"/><stop offset="1" stop-color="#00a3e0"/>
      </linearGradient>
      <linearGradient id="lgApp" x1="0" y1="0" x2="1" y2="1">
        <stop offset="0" stop-color="#006aff"/><stop offset="1" stop-color="#00a3e0"/>
      </linearGradient>
    </defs>
    <g>
      <rect x="${appNode.x - 60}" y="${appNode.y - 16}" width="120" height="32" rx="8" fill="#e8f2ff" stroke="rgba(0,106,255,0.55)"/>
      <text x="${appNode.x}" y="${appNode.y + 4}" text-anchor="middle" font-size="12" fill="#006aff" font-weight="600">应用接入层 · Proxy</text>
    </g>
    ${edges}
    ${mNodes.map(n => nodeHTML(n, true)).join('')}
    ${sNodes.map(n => nodeHTML(n, false)).join('')}
  </svg>`;
}
const topoLegend = `
<div class="topo-legend">
  <span><i style="background:#00b365"></i>正常</span>
  <span><i style="background:#ff9500"></i>警告</span>
  <span><i style="background:#f53f3f"></i>异常</span>
  <span><i style="background:#006aff;width:16px;height:2px;border-radius:2px"></i>读写链路</span>
  <span><i style="background:rgba(0,106,255,.35);width:16px;height:2px;border-radius:2px"></i>复制链路</span>
</div>`;

/* ================= 通用：性能监控 Tab ================= */
function renderMonitorTab(container) {
  container.innerHTML = `
  <div class="mon-grid">
    <div class="card"><div id="mon-cpu" class="chart-box lg"></div></div>
    <div class="card"><div id="mon-mem" class="chart-box lg"></div></div>
    <div class="card"><div id="mon-qps" class="chart-box lg"></div></div>
    <div class="card"><div id="mon-sess" class="chart-box lg"></div></div>
  </div>`;
  mkChart($('#mon-cpu'), lineOpt('CPU 使用率（%）', HOURS, [{ name: 'CPU', data: genSeries(55, 14, 14), color: '#006aff' }], '%'));
  mkChart($('#mon-mem'), lineOpt('内存使用率（%）', HOURS, [{ name: '内存', data: genSeries(68, 8), color: '#7a5af8' }], '%'));
  mkChart($('#mon-qps'), lineOpt('QPS / TPS', HOURS, [
    { name: 'QPS', data: genSeries(18000, 5000, 14), color: '#00a3e0' },
    { name: 'TPS', data: genSeries(4200, 1200, 14), color: '#00b365' }]));
  mkChart($('#mon-sess'), lineOpt('活跃会话数', HOURS, [{ name: '会话', data: genSeries(120, 40, 15), color: '#ff9500' }]));
}

/* ================= 页面：集群详情 ================= */
const CLUSTER_TABS = [
  { id: 'overview', label: '集群概览' },
  { id: 'topo', label: '拓扑图' },
  { id: 'tenant', label: '租户管理' },
  { id: 'monitor', label: '性能监控' },
  { id: 'report', label: '性能报告' },
  { id: 'param', label: '参数管理' },
];
function renderClusterDetail(cid) {
  const c = CLUSTERS.find(x => x.id === cid);
  if (!c) { location.hash = '#/clusters'; return; }
  setCrumb([{ label: '首页' }, { label: '集群', hash: '#/clusters' }, { label: c.name }]);
  // 租户为 OceanBase 等原生多租户数据库的内嵌概念，仅对应类型集群展示
  const tabs = CLUSTER_TABS.filter(t => t.id !== 'tenant' || c.type === 'oceanbase');

  $('#content').innerHTML = `
  <div class="detail-head">
    <div class="cluster-icon">${TYPE_ICON[c.type]}</div>
    <div>
      <div class="detail-title">${c.name} ${typeTag(c.type)} ${pill('ok', '运行中')}</div>
      <div class="detail-sub">
        <span>${c.version}</span><span>${c.mode}</span><span>${c.desc}</span>
      </div>
    </div>
    <div class="detail-head-right">
      <button class="btn">↻ 刷新</button>
      <button class="btn primary">⚡ 智能巡检</button>
    </div>
  </div>
  <div class="tabs" id="tabs">
    ${tabs.map((t, i) => `<div class="tab ${i === 0 ? 'active' : ''}" data-tab="${t.id}">${t.label}</div>`).join('')}
  </div>
  <div id="tabBody"></div>`;

  switchTab('#tabs', (tabId) => {
    const body = $('#tabBody');
    if (tabId === 'overview') {
      body.innerHTML = `
      <div class="stat-row">
        <div class="stat"><div class="num ok">${c.instances.length}</div><div class="lbl">实例数</div></div>
        <div class="stat"><div class="num ${c.cpu > 80 ? 'warn' : ''}">${c.cpu}%</div><div class="lbl">CPU 均值</div></div>
        <div class="stat"><div class="num ${c.mem > 85 ? 'warn' : ''}">${c.mem}%</div><div class="lbl">内存均值</div></div>
        <div class="stat"><div class="num">${(c.qps / 1000).toFixed(1)}k</div><div class="lbl">QPS</div></div>
        <div class="stat"><div class="num">${c.conn}</div><div class="lbl">连接数</div></div>
      </div>
      <div class="card" style="margin-bottom:14px"><div class="topo-wrap">${topoSVG(c)}</div>${topoLegend}</div>
      <div class="card">
        <div class="card-head"><div class="card-title"><span class="t-ico"></span>实例健康摘要</div><a class="card-link" href="#/dashboards">监控大盘 ›</a></div>
        <table class="tbl">
          <thead><tr><th>实例</th><th>角色</th><th>CPU</th><th>内存</th><th>连接数</th><th>状态</th></tr></thead>
          <tbody>${c.instances.map(i => `<tr>
            <td class="mono">${i.name}</td><td>${i.role}</td>
            <td><div class="bar"><i class="${i.cpu > 85 ? 'hot' : ''}" style="width:${i.cpu}%"></i></div></td>
            <td><div class="bar"><i class="${i.mem > 85 ? 'hot' : ''}" style="width:${i.mem}%"></i></div></td>
            <td>${i.conn}</td><td>${pill(i.status)}</td>
          </tr>`).join('')}</tbody>
        </table>
      </div>`;
    } else if (tabId === 'topo') {
      body.innerHTML = `<div class="card"><div class="topo-wrap">${topoSVG(c)}</div>${topoLegend}</div>`;
    } else if (tabId === 'tenant') {
      body.innerHTML = `<div class="card">
        <div class="card-head"><div class="card-title"><span class="t-ico"></span>租户资源分配</div><button class="btn sm primary">＋ 新建租户</button></div>
        <table class="tbl">
          <thead><tr><th>租户名</th><th>实例数</th><th>CPU 用量</th><th>内存用量</th><th>存储用量</th><th>状态</th><th>操作</th></tr></thead>
          <tbody>${c.tenants.map(t => `
            <tr><td class="mono">${t.name}</td><td>${t.inst}</td>
            <td><div class="bar"><i class="${t.cpu > 80 ? 'hot' : ''}" style="width:${t.cpu}%"></i></div></td>
            <td><div class="bar"><i class="${t.mem > 80 ? 'hot' : ''}" style="width:${t.mem}%"></i></div></td>
            <td>${t.storage}</td><td>${pill(t.status)}</td>
            <td><a>配额调整</a> · <a>监控</a></td></tr>`).join('')}
          </tbody>
        </table></div>`;
    } else if (tabId === 'monitor') {
      body.innerHTML = '';
      renderMonitorTab(body);
    } else if (tabId === 'report') {
      body.innerHTML = `<div class="report-grid">${REPORTS.map(r => `
        <div class="report-card">
          <div class="r-ico">${r.ico}</div>
          <h4>${r.title}</h4><p>${r.desc}</p>
          <div class="r-foot"><span>${r.date}</span><a class="card-link">下载（${r.size}）</a></div>
        </div>`).join('')}</div>`;
    } else if (tabId === 'param') {
      body.innerHTML = `<div class="card">
        <div class="card-head"><div class="card-title"><span class="t-ico"></span>参数管理</div><span class="card-sub">修改参数将通过工单审批后下发</span></div>
        <table class="tbl">
          <thead><tr><th>参数名</th><th>当前值</th><th>可选范围</th><th>说明</th><th>状态</th><th>操作</th></tr></thead>
          <tbody>${c.params.map(p => `
            <tr><td class="mono">${p.name}</td><td class="mono" style="color:var(--cyan)">${p.value}</td>
            <td class="mono">${p.range}</td><td>${p.desc}</td>
            <td>${p.status === 'pending' ? pill('warn', '待下发') : pill('ok', '已生效')}</td>
            <td><a>修改</a> · <a>历史</a></td></tr>`).join('')}
          </tbody>
        </table></div>`;
    }
  });
}

/* ================= 页面：实例详情 ================= */
const INST_TABS = [
  { id: 'topo', label: '拓扑图' },
  { id: 'user', label: '用户管理' },
  { id: 'monitor', label: '性能监控' },
  { id: 'sql', label: 'SQL 诊断' },
  { id: 'trx', label: '事务诊断' },
  { id: 'session', label: '会话管理' },
];
function renderInstanceDetail(cid, iid) {
  const c = CLUSTERS.find(x => x.id === cid);
  const inst = c && c.instances.find(i => i.id === iid);
  if (!c || !inst) { location.hash = '#/clusters'; return; }
  setCrumb([{ label: '首页' }, { label: '集群', hash: '#/clusters' }, { label: c.name, hash: `#/cluster/${c.id}` }, { label: inst.name }]);

  $('#content').innerHTML = `
  <div class="detail-head">
    <div class="cluster-icon">${TYPE_ICON[c.type]}</div>
    <div>
      <div class="detail-title">${inst.name} ${pill(inst.status)} ${typeTag(c.type)}</div>
      <div class="detail-sub">
        <span class="mono" style="font-family:Menlo,monospace">${inst.ip}:${inst.port}</span>
        <span>角色：${inst.role}</span><span>版本：${inst.ver}</span><span>连接数：${inst.conn}</span>
      </div>
    </div>
    <div class="detail-head-right">
      <button class="btn">↻ 刷新</button>
      <button class="btn primary">🤖 AI 诊断</button>
    </div>
  </div>
  <div class="tabs" id="tabs">
    ${INST_TABS.map((t, i) => `<div class="tab ${i === 0 ? 'active' : ''}" data-tab="${t.id}">${t.label}</div>`).join('')}
  </div>
  <div id="tabBody"></div>`;

  switchTab('#tabs', (tabId) => {
    const body = $('#tabBody');
    if (tabId === 'topo') {
      body.innerHTML = `<div class="card"><div class="topo-wrap">${topoSVG(c)}</div>${topoLegend}</div>`;
    } else if (tabId === 'user') {
      body.innerHTML = `<div class="card">
        <div class="card-head"><div class="card-title"><span class="t-ico"></span>数据库账号</div><button class="btn sm primary">＋ 创建账号</button></div>
        <table class="tbl">
          <thead><tr><th>用户名</th><th>允许主机</th><th>权限</th><th>最近登录</th><th>状态</th><th>操作</th></tr></thead>
          <tbody>${INSTANCE_USERS.map(u => `
            <tr><td class="mono">${u.user}</td><td class="mono">${u.host}</td><td class="mono">${u.priv}</td>
            <td>${u.lastLogin}</td>
            <td>${u.status === 'ok' ? pill('ok') : u.status === 'warn' ? pill('warn', '长期未活跃') : pill('err', '建议回收')}</td>
            <td><a>授权</a> · <a>重置密码</a> · <a>锁定</a></td></tr>`).join('')}
          </tbody>
        </table></div>`;
    } else if (tabId === 'monitor') {
      body.innerHTML = '';
      renderMonitorTab(body);
    } else if (tabId === 'sql') {
      body.innerHTML = `
      <div class="card">
        <div class="card-head"><div class="card-title"><span class="t-ico"></span>慢 SQL 诊断（近 24h）</div><span class="card-sub">共 ${SLOW_SQLS.length} 条待优化</span></div>
        <table class="tbl">
          <thead><tr><th>SQL 指纹</th><th>库</th><th>平均耗时</th><th>扫描行数</th><th>执行次数</th><th>操作</th></tr></thead>
          <tbody>${SLOW_SQLS.map((s, i) => `
            <tr><td class="mono" style="max-width:380px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">${s.sql}</td>
            <td>${s.db}</td><td style="color:var(--amber)">${s.time}</td><td>${s.rows}</td><td>${s.count}</td>
            <td><a data-diag="${i}">AI 诊断</a></td></tr>`).join('')}
          </tbody>
        </table>
        <div class="advice" id="sqlAdvice" style="display:none">
          <h4>🤖 AI 优化建议</h4>
          <ul id="adviceList"></ul>
        </div>
      </div>`;
      body.querySelectorAll('[data-diag]').forEach(a => a.onclick = () => {
        const s = SLOW_SQLS[+a.dataset.diag];
        $('#sqlAdvice').style.display = 'block';
        $('#adviceList').innerHTML = `
          <li>对 <code>${s.db}</code> 表缺少合适索引，建议添加联合索引 <code>idx_status_uid (status, uid)</code>，预计扫描行数下降 <b style="color:var(--green)">92%</b>；</li>
          <li>存在隐式类型转换导致索引失效，请核对字段类型与传参类型一致；</li>
          <li>该 SQL 日均执行 <b>${s.count}</b> 次，优化后预计集群 CPU 下降约 <b style="color:var(--green)">8%</b>；</li>
          <li>可一键生成索引变更工单，由参数管理通道灰度下发。</li>`;
      });
    } else if (tabId === 'trx') {
      body.innerHTML = `<div class="card">
        <div class="card-head"><div class="card-title"><span class="t-ico"></span>长事务 / 未提交事务</div><span class="card-sub">阈值 &gt; 30s</span></div>
        <table class="tbl">
          <thead><tr><th>事务 ID</th><th>会话</th><th>用户</th><th>持续时间</th><th>Undo 大小</th><th>锁行数</th><th>等待锁</th><th>当前 SQL</th><th>状态</th></tr></thead>
          <tbody>${TRANSACTIONS.map(t => `
            <tr><td class="mono">${t.id}</td><td>${t.session}</td><td class="mono">${t.user}</td>
            <td style="color:${t.status === 'err' ? 'var(--red)' : 'inherit'}">${t.dur}</td>
            <td>${t.undo}</td><td>${t.lockRows}</td><td>${t.waiting}</td>
            <td class="mono" style="max-width:220px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">${t.sql}</td>
            <td>${pill(t.status)}</td></tr>`).join('')}
          </tbody>
        </table>
        <div class="advice">
          <h4>🔗 阻塞链分析</h4>
          <ul>
            <li><code>TRX-998231</code>（会话 88231）持有 <code>stock_record</code> 行锁 1m28s，阻塞 3 个后续会话；</li>
            <li>根因：应用侧开启事务后未及时提交，建议联系 <code>app_rw</code> 业务方确认；</li>
            <li>若持续超过 5 分钟，可执行「Kill 会话」一键止血（会话管理中操作）。</li>
          </ul>
        </div></div>`;
    } else if (tabId === 'session') {
      body.innerHTML = `<div class="card">
        <div class="card-head"><div class="card-title"><span class="t-ico"></span>活跃会话</div>
          <div><span class="card-sub">共 ${SESSIONS.length} 个会话 · 2 个异常</span></div></div>
        <table class="tbl">
          <thead><tr><th>ID</th><th>用户</th><th>来源</th><th>库</th><th>命令</th><th>时长</th><th>状态</th><th>锁信息</th><th>操作</th></tr></thead>
          <tbody>${SESSIONS.map(s => `
            <tr><td class="mono">${s.id}</td><td class="mono">${s.user}</td><td class="mono">${s.host}</td>
            <td>${s.db}</td><td>${s.cmd}</td>
            <td style="color:${s.status !== 'ok' ? 'var(--amber)' : 'inherit'}">${s.time}</td>
            <td>${s.state}</td><td>${s.lock === '—' ? '—' : `<span style="color:var(--red)">${s.lock}</span>`}</td>
            <td>${s.status !== 'ok' ? '<button class="btn sm danger">Kill</button>' : '<span class="card-sub">—</span>'}</td></tr>`).join('')}
          </tbody>
        </table></div>`;
    }
  });
}

/* ---------- Tab 切换 ---------- */
function switchTab(selector, cb) {
  const tabs = $$(selector + ' .tab');
  tabs.forEach(t => t.onclick = () => {
    tabs.forEach(x => x.classList.remove('active'));
    t.classList.add('active');
    chartInstances.length = 0;
    cb(t.dataset.tab);
  });
  cb(tabs[0].dataset.tab);
}

/* ================= 浮窗 Chat ================= */
const QUICK_QS = ['当前有哪些告警实例？', '分析 TOP1 慢 SQL', 'stock_record 锁等待原因'];
const BOT_KB = [
  { k: ['告警'], a: '当前共有 12 个告警实例，其中 3 个为严重级别：\n\n1. mysql-prod-order-01：CPU 98%，疑似全表扫描导致（健康分 96）\n2. pg-prod-report-02：活跃会话堆积 + 锁等待（健康分 88）\n3. mysql-prod-user-03：Buffer 命中率跌至 71%（健康分 82）\n\n建议优先处理 #1，可在「集群管理」中定位实例详情。' },
  { k: ['慢 SQL', '慢SQL', 'TOP1'], a: 'TOP1 慢 SQL：\n\nSELECT o.*, u.name FROM trade_order o JOIN user u …\n\n📊 分析结论：\n• trade_order.status 字段无索引，导致全表扫描 438 万行\n• 日均执行 342 次，平均耗时 12.8s\n\n💡 优化建议：\n添加联合索引 idx_trade_order_status_uid (status, uid)，预计耗时降至 0.3s 以内。是否需要我生成索引变更工单？' },
  { k: ['锁', 'stock_record'], a: '🔗 锁等待链路分析：\n\n会话 88231（app_rw@10.20.4.21）持有 stock_record 表的行锁已超过 1m28s，事务 TRX-998231 未提交，阻塞了 3 个后续会话。\n\n根因：应用开启事务后长时间未提交（疑似等待外部接口返回）。\n\n建议：\n1. 联系业务方确认事务边界；\n2. 如影响扩大，可在「会话管理」中 Kill 会话 88231 止血；\n3. 长期方案：拆分热点行扣减逻辑（如合并扣减 / 队列化）。' },
  { k: ['你好', 'hi', 'hello'], a: '你好！我是 DB Copilot 智能运维助手 🤖\n\n我可以帮你：\n• 分析慢 SQL / 锁等待 / 性能异常根因\n• 解读监控指标与容量趋势\n• 生成巡检报告与优化建议\n\n试着问我："当前有哪些告警实例？"' },
];
function botReply(q) {
  const hit = BOT_KB.find(e => e.k.some(k => q.includes(k)));
  return hit ? hit.a : `已收到问题：「${q}」\n\n作为原型演示，我的知识库目前覆盖：告警概览、慢 SQL 分析、锁等待诊断。\n\n正式版本将接入诊断 Agent（可编程 API），支持实时根因分析与自动修复建议。`;
}
function addMsg(role, text) {
  const box = $('#chatBody');
  const div = document.createElement('div');
  div.className = 'msg ' + role;
  const av = document.createElement('div');
  av.className = 'm-avatar';
  av.textContent = role === 'bot' ? 'AI' : '我';
  const bub = document.createElement('div');
  bub.className = 'm-bubble';
  bub.textContent = text;
  div.appendChild(av); div.appendChild(bub);
  box.appendChild(div);
  box.scrollTop = box.scrollHeight;
  return div;
}
function initChat() {
  const panel = $('#chatPanel'), fab = $('#chatFab');
  fab.onclick = () => {
    panel.classList.toggle('open');
    if (panel.classList.contains('open') && !$('#chatBody').children.length) {
      addMsg('bot', '你好，我是 DB Copilot 智能运维助手 🤖\n可以问我告警、慢 SQL、锁等待等问题，或点击下方快捷问题。');
    }
  };
  $('#chatClose').onclick = () => panel.classList.remove('open');
  $('#chatQuick').innerHTML = QUICK_QS.map(q => `<button>${q}</button>`).join('');
  $('#chatQuick').querySelectorAll('button').forEach(b => b.onclick = () => sendChat(b.textContent));

  const send = () => sendChat($('#chatInput').value);
  $('#chatSend').onclick = send;
  $('#chatInput').onkeydown = (e) => { if (e.key === 'Enter') send(); };
}
function sendChat(text) {
  text = (text || '').trim();
  if (!text) return;
  $('#chatInput').value = '';
  $('#chatPanel').classList.add('open');
  addMsg('user', text);
  const typing = addMsg('bot', '');
  typing.querySelector('.m-bubble').innerHTML = '<span class="typing"><i></i><i></i><i></i></span>';
  setTimeout(() => {
    typing.querySelector('.m-bubble').textContent = botReply(text);
    $('#chatBody').scrollTop = $('#chatBody').scrollHeight;
  }, 900);
}

/* ================= 页面：主机 ================= */
function renderHosts() {
  setCrumb([{ label: '首页' }, { label: '主机' }]);
  const abn = HOSTS.filter(h => h.status !== 'ok').length;
  const avg = (k) => Math.round(HOSTS.reduce((a, h) => a + h[k], 0) / HOSTS.length);
  $('#content').innerHTML = `
  <div class="page-title">主机</div>
  <div class="page-desc">实例宿主的资源水位视图（物理机 / 虚机），共 ${HOSTS.length} 台</div>
  <div class="stat-row">
    <div class="stat"><div class="num">${HOSTS.length}</div><div class="lbl">主机总数</div></div>
    <div class="stat"><div class="num ${abn ? 'warn' : 'ok'}">${abn}</div><div class="lbl">异常/警告主机</div></div>
    <div class="stat"><div class="num">${avg('cpu')}%</div><div class="lbl">平均 CPU</div></div>
    <div class="stat"><div class="num">${avg('mem')}%</div><div class="lbl">平均内存</div></div>
    <div class="stat"><div class="num">${avg('disk')}%</div><div class="lbl">平均磁盘水位</div></div>
  </div>
  <div class="card">
    <table class="tbl">
      <thead><tr><th>IP</th><th>可用区</th><th>规格</th><th>操作系统</th><th>CPU</th><th>内存</th><th>磁盘</th><th>实例数</th><th>承载实例</th><th>状态</th></tr></thead>
      <tbody>
        ${HOSTS.map(h => `
        <tr>
          <td class="mono">${h.ip}</td><td>${h.zone}</td><td>${h.spec}</td><td>${h.os}</td>
          <td><div class="bar"><i class="${h.cpu > 85 ? 'hot' : ''}" style="width:${h.cpu}%"></i></div></td>
          <td><div class="bar"><i class="${h.mem > 85 ? 'hot' : ''}" style="width:${h.mem}%"></i></div></td>
          <td><div class="bar"><i class="${h.disk > 90 ? 'hot' : ''}" style="width:${h.disk}%"></i></div></td>
          <td>${h.insts.length}</td>
          <td style="max-width:220px"><div style="white-space:nowrap;overflow:hidden;text-overflow:ellipsis" title="${h.insts.join(', ')}">${h.insts.map(i => `<a href="#/cluster/${h.cid}">${i}</a>`).join('、')}</div></td>
          <td>${pill(h.status)}</td>
        </tr>`).join('')}
      </tbody>
    </table>
  </div>`;
}

/* ================= 页面：监控大盘（本地自建，复刻 Grafana 面板模型） ================= */
const DASH_KEY = 'dbCopilotDashCfg';
const PANELS_KEY = 'dbCopilotDashPanels';
const DASHBOARDS_KEY = 'dbCopilotDashboards'; // 多大盘集合

/* 预置指标库（集群级） */
const METRIC_LIB = [
  { id: 'qps',        name: 'QPS',           unit: '',   base: 210, jitter: 60 },
  { id: 'cpu',        name: 'CPU 使用率',     unit: '%',  base: 55,  jitter: 14 },
  { id: 'mem',        name: '内存使用率',     unit: '%',  base: 68,  jitter: 10 },
  { id: 'sessions',   name: '活跃会话数',     unit: '',   base: 120, jitter: 40 },
  { id: 'slow_sql',   name: '慢 SQL 趋势',    unit: '',   base: 18,  jitter: 8  },
  { id: 'lock_wait',  name: '锁等待会话',     unit: '',   base: 6,   jitter: 4  },
  { id: 'disk',       name: '磁盘使用率',     unit: '%',  base: 66,  jitter: 3  },
  { id: 'repl_delay', name: '复制延迟',       unit: 's',  base: 2,   jitter: 3  },
];

window.METRIC_LIB = METRIC_LIB; // 供 query.js MockProvider 使用

/* 图表样式默认值（大盘面板编辑器共用）*/
const DEFAULT_PANEL_STYLE = {
  lineWidth: 2, fill: 0, points: 0, connectNulls: true,
  stack: 'none', legend: 'top', yMin: null, yMax: null, decimals: null, unit: '',
};

/* 面板数据模型（对齐 Grafana）：panel = 多 targets（query）→ 多序列同轴叠加
 * 每个 target 支持序列级覆盖：名称 / 颜色 / 左·右 Y 轴 / 图表类型 / 单位 / 聚合 / 分组
 * 面板级：type（timeseries/bar/gauge/stat/table）、legend、thresholds（阈值 steps）、
 *         annotations（事件标注）、drilldown（下钻目标）、w（12 列网格宽度）、style（样式） */
const DEFAULT_PANELS = [
  { id: 'p1', title: 'QPS 与活跃会话', type: 'timeseries', legend: true, scope: 'global', visible: true, w: 6,
    thresholds: { steps: [] }, annotations: { enable: true, types: ['release', 'switch', 'alert'] }, drilldown: null,
    targets: [
      { metric: 'qps', name: 'QPS', color: '#006aff', axis: 'left', type: 'line' },
      { metric: 'sessions', name: '活跃会话', color: '#ff9500', axis: 'right', type: 'line' },
    ] },
  { id: 'p2', title: 'CPU / 内存同轴对比', type: 'timeseries', legend: true, scope: 'global', visible: true, w: 6,
    thresholds: { steps: [{ value: 85, color: '#ff9500' }, { value: 95, color: '#f53f3f' }] },
    annotations: { enable: false, types: ['release', 'switch', 'alert'] }, drilldown: null,
    targets: [
      { metric: 'cpu', name: 'CPU 使用率', color: '#00a3e0', axis: 'left', type: 'area' },
      { metric: 'mem', name: '内存使用率', color: '#7a5af8', axis: 'left', type: 'line' },
    ] },
  { id: 'p3', title: '慢 SQL 与锁等待', type: 'timeseries', legend: true, scope: 'global', visible: true, w: 6,
    thresholds: { steps: [] }, annotations: { enable: false, types: ['release', 'switch', 'alert'] }, drilldown: null,
    targets: [
      { metric: 'slow_sql', name: '慢 SQL', color: '#7a5af8', axis: 'left', type: 'bar' },
      { metric: 'lock_wait', name: '锁等待', color: '#f53f3f', axis: 'left', type: 'line' },
    ] },
  { id: 'p4', title: '当前活跃会话', type: 'stat', legend: false, scope: 'global', visible: true, w: 3,
    thresholds: { steps: [{ value: 150, color: '#ff9500' }, { value: 200, color: '#f53f3f' }] },
    annotations: { enable: false, types: ['release', 'switch', 'alert'] }, drilldown: null,
    targets: [{ metric: 'sessions', name: '活跃会话', color: '#ff9500', axis: 'left', type: 'line' }] },
  { id: 'p5', title: '磁盘使用率', type: 'gauge', legend: false, scope: 'global', visible: true, w: 3,
    thresholds: { steps: [] }, annotations: { enable: false, types: ['release', 'switch', 'alert'] }, drilldown: null,
    targets: [{ metric: 'disk', name: '磁盘使用率', color: '#00b365', axis: 'left', type: 'line' }] },
  { id: 'p6', title: '复制延迟', type: 'timeseries', legend: false, scope: 'global', visible: true, w: 6,
    thresholds: { steps: [{ value: 8, color: '#f53f3f' }] }, annotations: { enable: false, types: ['release', 'switch', 'alert'] }, drilldown: null,
    targets: [{ metric: 'repl_delay', name: '复制延迟', color: '#f53f3f', axis: 'left', type: 'line' }] },
];

function loadDashCfg() {
  let cfg = { range: '24h', refresh: '0' };
  try { cfg = Object.assign(cfg, JSON.parse(localStorage.getItem(DASH_KEY) || '{}')); } catch (e) {}
  return { range: cfg.range, refresh: cfg.refresh, compareYesterday: !!cfg.compareYesterday }; // 白名单，仅保留本地大盘所需配置
}

/* 补齐面板 style 默认值（与 DEFAULT_PANEL_STYLE 合并，旧值优先）*/
function normalizeStyle(s) {
  return Object.assign({}, DEFAULT_PANEL_STYLE, s || {});
}

/* 旧版本（单指标面板）迁移：metric/chart → targets/type；新字段补默认值 */
function normalizePanel(p) {
  if (!p) return null;
  if (!p.targets && p.metric) {
    const m = METRIC_LIB.find(x => x.id === p.metric) || {};
    return {
      id: p.id, title: p.title, scope: p.scope || 'global', visible: p.visible !== false,
      type: { line: 'timeseries', bar: 'bar', gauge: 'gauge' }[p.chart] || 'timeseries', legend: true,
      w: p.w || 6,
      style: normalizeStyle(p.style),
      thresholds: p.thresholds || { steps: [] },
      annotations: p.annotations || { enable: false, types: ['release', 'switch', 'alert'] },
      drilldown: p.drilldown || null,
      targets: [{ metric: p.metric, name: m.name || p.title, color: '', axis: 'left', type: 'line', agg: '', groupBy: '' }],
    };
  }
  if (Array.isArray(p.targets)) {
    p.targets = p.targets.map(t => ({
      metric: t.metric, name: t.name || '', color: t.color || '',
      axis: t.axis === 'right' ? 'right' : 'left', type: t.type || 'line', unit: t.unit || '',
      agg: t.agg || '', groupBy: t.groupBy || '', dbType: t.dbType || '',
    }));
  }
  if (!p.type) p.type = 'timeseries';
  p.w = p.w || 6;
  p.style = normalizeStyle(p.style);
  p.thresholds = p.thresholds || { steps: [] };
  p.annotations = p.annotations || { enable: false, types: ['release', 'switch', 'alert'] };
  p.drilldown = p.drilldown || null;
  return p;
}
function loadPanels() {
  try {
    const arr = JSON.parse(localStorage.getItem(PANELS_KEY) || 'null');
    if (Array.isArray(arr)) {
      const norm = arr.map(normalizePanel).filter(Boolean);
      localStorage.setItem(PANELS_KEY, JSON.stringify(norm)); // 迁移结果回写
      return norm; // 空数组也尊重（用户删光了所有面板）
    }
  } catch (e) {}
  localStorage.setItem(PANELS_KEY, JSON.stringify(DEFAULT_PANELS));
  return JSON.parse(JSON.stringify(DEFAULT_PANELS));
}
function savePanels(arr) {
  if (dashState && dashState.dashId) updateDashboard(dashState.dashId, { panels: arr });
  else localStorage.setItem(PANELS_KEY, JSON.stringify(arr));
}

/* ---------- 多大盘：集合读写 + 旧数据迁移 ---------- */
function normalizeDashboard(d) {
  if (!d || typeof d !== 'object') return null;
  return {
    id: d.id || ('d' + Date.now()),
    title: d.title || '未命名大盘',
    description: d.description || '',
    cfg: d.cfg || { range: '24h', refresh: '0', compareYesterday: false },
    panels: Array.isArray(d.panels) ? d.panels.map(normalizePanel).filter(Boolean) : [],
    createdAt: d.createdAt || Date.now(),
    updatedAt: d.updatedAt || Date.now(),
  };
}
function makeDefaultDashboard() {
  const now = Date.now();
  return {
    id: 'd-default', title: '数据库综合监控大盘',
    description: '默认大盘 · QPS、CPU/内存、慢 SQL、磁盘、复制延迟等核心指标',
    cfg: { range: '24h', refresh: '0', compareYesterday: false },
    panels: JSON.parse(JSON.stringify(DEFAULT_PANELS)), createdAt: now, updatedAt: now,
  };
}
function migrateOldDashboard() {
  const oldCfg = localStorage.getItem(DASH_KEY);
  const oldPanels = localStorage.getItem(PANELS_KEY);
  if (!oldCfg && !oldPanels) return null;
  let cfg = { range: '24h', refresh: '0', compareYesterday: false };
  try { cfg = Object.assign(cfg, JSON.parse(oldCfg || '{}')); } catch (e) {}
  let panels = JSON.parse(JSON.stringify(DEFAULT_PANELS));
  try {
    const arr = JSON.parse(oldPanels || 'null');
    if (Array.isArray(arr)) panels = arr.map(normalizePanel).filter(Boolean);
  } catch (e) {}
  const now = Date.now();
  return { id: 'd-default', title: '数据库综合监控大盘', description: '由历史配置迁移 · 核心指标监控', cfg, panels, createdAt: now, updatedAt: now };
}
function loadDashboards() {
  try {
    const arr = JSON.parse(localStorage.getItem(DASHBOARDS_KEY) || 'null');
    if (Array.isArray(arr)) {
      const norm = arr.map(normalizeDashboard).filter(Boolean);
      if (norm.length) { if (norm.length !== arr.length) saveDashboards(norm); return norm; }
    }
  } catch (e) {}
  const list = [migrateOldDashboard() || makeDefaultDashboard()];
  saveDashboards(list);
  return list;
}
function saveDashboards(arr) { localStorage.setItem(DASHBOARDS_KEY, JSON.stringify(arr)); }
function getDashboard(id) { return loadDashboards().find(d => d.id === id) || null; }
function updateDashboard(id, patch) {
  const arr = loadDashboards();
  const idx = arr.findIndex(d => d.id === id);
  if (idx < 0) return null;
  arr[idx] = Object.assign({}, arr[idx], patch, { updatedAt: Date.now() });
  saveDashboards(arr);
  return arr[idx];
}

function escapeHtml(s) { return String(s == null ? '' : s).replace(/[&<>"]/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c])); }
function relTime(ts) {
  const d = Date.now() - (ts || 0);
  if (d < 60000) return '刚刚';
  if (d < 3600000) return Math.floor(d / 60000) + ' 分钟前';
  if (d < 86400000) return Math.floor(d / 3600000) + ' 小时前';
  if (d < 2592000000) return Math.floor(d / 86400000) + ' 天前';
  try { return new Date(ts).toLocaleDateString(); } catch (e) { return ''; }
}

/* ---------- 大盘列表页 ---------- */
function dashCardHTML(d) {
  const n = (d.panels || []).filter(p => p.visible !== false).length;
  return `<div class="dash-card" data-id="${d.id}" title="点击进入">
    <div class="dash-card-accent"></div>
    <div class="dash-card-body">
      <div class="dash-card-title">${escapeHtml(d.title)}</div>
      <div class="dash-card-desc">${d.description ? escapeHtml(d.description) : '<span class="muted">暂无描述</span>'}</div>
      <div class="dash-card-meta">
        <span>${n} 个面板 · ${relTime(d.updatedAt)}</span>
        <span class="dash-card-menu" data-id="${d.id}" title="管理">⋯</span>
      </div>
    </div>
  </div>`;
}
function renderDashboards() {
  setCrumb([{ label: '首页' }, { label: '监控大盘' }]);
  $('#content').innerHTML = `
    <div class="page-title">监控大盘 <span class="pill info"><i></i>共 ${loadDashboards().length} 个大盘</span></div>
    <div class="page-desc">点击卡片进入大盘；支持新建、编辑信息、制作副本、删除。各大盘配置独立存储于本地浏览器。</div>
    <div class="dash-list-head">
      <input class="dash-search" id="dashSearch" placeholder="搜索大盘标题或描述…">
      <button class="btn sm primary" id="dashNew">+ 新建大盘</button>
    </div>
    <div class="dash-list-grid" id="dashGrid"></div>`;
  const grid = $('#dashGrid');
  /* 只重渲染网格，搜索框只创建一次，避免输入焦点丢失与重渲染竞态 */
  const draw = (kw) => {
    const list = loadDashboards().slice().sort((a, b) => b.updatedAt - a.updatedAt);
    const filtered = kw ? list.filter(d => (d.title + ' ' + d.description).toLowerCase().includes(kw.toLowerCase())) : list;
    grid.innerHTML = (filtered.length ? filtered.map(dashCardHTML).join('')
      : (kw ? '<div class="dash-empty-hint">没有找到匹配的大盘</div>' : ''))
      + `<div class="dash-card dash-new-card" id="dashNewCard"><span class="dash-new-plus">＋</span>新建大盘</div>`;
    $$('#dashGrid .dash-card').forEach(card => {
      if (card.id === 'dashNewCard') return;
      card.onclick = () => { location.hash = '#/dashboard/' + card.dataset.id; };
      const mb = card.querySelector('.dash-card-menu');
      if (mb) mb.onclick = (e) => { e.stopPropagation(); openDashCardMenu(card.dataset.id, mb); };
    });
    $('#dashNewCard').onclick = createDashboard;
  };
  $('#dashNew').onclick = createDashboard;
  $('#dashSearch').oninput = (e) => draw(e.target.value);
  draw('');
}

/* ---------- 大盘生命周期 CRUD ---------- */
function openDashDialog(opts) {
  const overlay = document.createElement('div');
  overlay.className = 'dash-popover-overlay';
  overlay.innerHTML = `<div class="dash-dialog">
    <div class="dap-head">${escapeHtml(opts.title)}</div>
    <div class="dap-body">
      <label>标题<input id="dgTitle" value="${escapeHtml((opts.initial && opts.initial.title) || '')}" placeholder="如：交易核心大盘"></label>
      <label>描述<textarea id="dgDesc" rows="3" placeholder="可选，简述该大盘的用途">${escapeHtml((opts.initial && opts.initial.description) || '')}</textarea></label>
    </div>
    <div class="dap-foot">
      <button class="btn sm" id="dgCancel">取消</button>
      <button class="btn sm primary" id="dgOk">${escapeHtml(opts.okText || '确定')}</button>
    </div>
  </div>`;
  document.body.appendChild(overlay);
  const close = () => overlay.remove();
  overlay.onclick = (e) => { if (e.target === overlay) close(); };
  $('#dgCancel').onclick = close;
  const tIn = $('#dgTitle'); tIn.focus(); tIn.select();
  const submit = () => {
    const title = tIn.value.trim();
    if (!title) { tIn.focus(); return; }
    const description = $('#dgDesc').value.trim();
    close();
    opts.onOk(title, description);
  };
  $('#dgOk').onclick = submit;
  tIn.onkeydown = (e) => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); submit(); } };
}
function createDashboard() {
  openDashDialog({ title: '新建大盘', okText: '创建并进入', onOk: (title, description) => {
    const now = Date.now();
    const d = normalizeDashboard({ id: 'd' + now, title, description, cfg: { range: '24h', refresh: '0', compareYesterday: false }, panels: [], createdAt: now, updatedAt: now });
    const arr = loadDashboards(); arr.push(d); saveDashboards(arr);
    location.hash = '#/dashboard/' + d.id;
  } });
}
function editDashboardMeta(id) {
  const d = getDashboard(id); if (!d) return;
  openDashDialog({ title: '编辑大盘信息', okText: '保存', initial: { title: d.title, description: d.description }, onOk: (title, description) => {
    updateDashboard(id, { title, description }); renderDashboards();
  } });
}
function duplicateDashboard(id) {
  const d = getDashboard(id); if (!d) return;
  const now = Date.now();
  const copy = normalizeDashboard({ id: 'd' + now, title: d.title + ' 副本', description: d.description, cfg: JSON.parse(JSON.stringify(d.cfg)), panels: JSON.parse(JSON.stringify(d.panels)), createdAt: now, updatedAt: now });
  const arr = loadDashboards(); arr.push(copy); saveDashboards(arr);
  renderDashboards();
}
function confirmDialog(opts) {
  const overlay = document.createElement('div');
  overlay.className = 'dash-popover-overlay';
  overlay.innerHTML = `<div class="dash-dialog">
    <div class="dap-head">${escapeHtml(opts.title || '请确认')}</div>
    <div class="dap-body"><div class="dash-confirm-msg">${escapeHtml(opts.message || '')}</div></div>
    <div class="dap-foot">
      <button class="btn sm" id="cfCancel">取消</button>
      <button class="btn sm ${opts.danger ? 'danger' : 'primary'}" id="cfOk">${escapeHtml(opts.okText || '确认')}</button>
    </div>
  </div>`;
  document.body.appendChild(overlay);
  const close = () => overlay.remove();
  overlay.onclick = (e) => { if (e.target === overlay) close(); };
  $('#cfCancel').onclick = close;
  $('#cfOk').onclick = () => { close(); if (opts.onOk) opts.onOk(); };
}
function deleteDashboard(id) {
  const d = getDashboard(id); if (!d) return;
  confirmDialog({
    title: '删除大盘', okText: '删除', danger: true,
    message: '确定删除大盘「' + d.title + '」吗？该操作不可恢复。',
    onOk: () => {
      saveDashboards(loadDashboards().filter(x => x.id !== id));
      if (dashState && dashState.dashId === id) dashState = null;
      renderDashboards();
    },
  });
}
function closeDashCardMenu() { const m = document.getElementById('dashCardPopover'); if (m) m.remove(); }
function openDashCardMenu(id, anchor) {
  closeDashCardMenu();
  const r = anchor.getBoundingClientRect();
  const menu = document.createElement('div');
  menu.className = 'dash-card-popover';
  menu.id = 'dashCardPopover';
  menu.style.top = (r.bottom + window.scrollY + 4) + 'px';
  menu.style.left = Math.max(8, r.right + window.scrollX - 156) + 'px';
  menu.innerHTML = `
    <button class="exp-more-item" data-act="open">打开</button>
    <button class="exp-more-item" data-act="edit">编辑信息</button>
    <button class="exp-more-item" data-act="dup">制作副本</button>
    <button class="exp-more-item" data-act="del">删除</button>`;
  document.body.appendChild(menu);
  menu.querySelectorAll('[data-act]').forEach(b => b.onclick = (e) => {
    e.stopPropagation(); const act = b.dataset.act; closeDashCardMenu();
    if (act === 'open') location.hash = '#/dashboard/' + id;
    else if (act === 'edit') editDashboardMeta(id);
    else if (act === 'dup') duplicateDashboard(id);
    else if (act === 'del') deleteDashboard(id);
  });
  setTimeout(() => {
    const onDown = (e) => { if (!menu.contains(e.target) && e.target !== anchor) { closeDashCardMenu(); document.removeEventListener('mousedown', onDown); } };
    document.addEventListener('mousedown', onDown);
  }, 0);
}

/* 大盘页面状态 */
let dashState = null;
let dashTimer = null;

function renderDashboard(dashId) {
  const dash = dashId ? getDashboard(dashId) : null;
  if (!dash) { location.hash = '#/dashboards'; return; }
  setCrumb([{ label: '首页' }, { label: '监控大盘', hash: '#/dashboards' }, { label: dash.title }]);
  if (!dashState) dashState = {};
  dashState.cfg = dash.cfg; dashState.panels = dash.panels;
  dashState.dashId = dash.id; dashState.title = dash.title;

  const renderShell = () => {
    const cfg = dashState.cfg;
    const visibleCnt = dashState.panels.filter(p => p.visible).length;
    $('#content').innerHTML = `
    <div class="page-title"><a class="dash-back" href="#/dashboards" title="返回大盘列表">‹</a><span class="dash-title-name">${dashState.title || '监控大盘'}</span></div>
    <div class="page-desc">本地自建大盘 · 复刻 Grafana 面板模型：多指标同轴叠加、双 Y 轴、序列级样式覆盖、面板配置本地保存</div>
    <div class="card dash-toolbar-card">
      <div class="dash-toolbar">
        <div class="select-wrap"><select id="dashRange">
          ${['1h', '6h', '24h', '7d'].map(r => `<option value="${r}" ${cfg.range === r ? 'selected' : ''}>近 ${r}</option>`).join('')}
        </select></div>
        <div class="select-wrap"><select id="dashRefresh">
          ${[['0', '不自动刷新'], ['10s', '每 10s'], ['30s', '每 30s'], ['1m', '每 1m'], ['5m', '每 5m']].map(([v, l]) => `<option value="${v}" ${cfg.refresh === v ? 'selected' : ''}>${l}</option>`).join('')}
        </select></div>
        <label class="dash-checkbox"><input type="checkbox" id="dashCompare" ${cfg.compareYesterday ? 'checked' : ''}> 对比昨日</label>
        <button class="btn sm primary" id="dashAddPanel">+ 添加面板</button>
        <span class="card-sub">${visibleCnt} 个面板 · 布局与配置保存在本地浏览器</span>
      </div>
    </div>
    <div id="dashBody" style="margin-top:14px"></div>`;

    $('#dashRange').onchange = (e) => { dashState.cfg.range = e.target.value; persistCfg(); drawDashBody(); };
    $('#dashRefresh').onchange = (e) => { dashState.cfg.refresh = e.target.value; persistCfg(); setupTimer(); };
    $('#dashCompare').onchange = (e) => { dashState.cfg.compareYesterday = e.target.checked; persistCfg(); drawDashBody(); };
    $('#dashAddPanel').onclick = () => openAddPanel();
    drawDashBody();
    setupTimer();
  };

  const persistCfg = () => { if (dashState.dashId) updateDashboard(dashState.dashId, { cfg: dashState.cfg }); };

  const setupTimer = () => {
    if (dashTimer) { clearInterval(dashTimer); dashTimer = null; }
    const ms = { '10s': 10000, '30s': 30000, '1m': 60000, '5m': 300000 }[dashState.cfg.refresh];
    if (ms) dashTimer = setInterval(() => { if ($('#dashBody')) drawDashBody(); }, ms);
  };

  const drawDashBody = () => {
    const body = $('#dashBody');
    chartInstances.forEach(c => c.dispose());
    chartInstances.length = 0;
    drawLocalDash(body);
  };

  const drawLocalDash = (body) => {
    const visible = dashState.panels.filter(p => p.visible);
    if (!visible.length) {
      body.innerHTML = '<div class="card"><div class="empty" style="padding:60px 0">暂无面板，请点击上方「+ 添加面板」创建</div></div>';
      return;
    }
    body.innerHTML = `<div class="dash-grid" id="dashGrid">
      ${visible.map(p => panelHTML(p)).join('')}
    </div>`;
    visible.forEach(p => renderPanelChart(p));
    bindPanelOps();
  };

  /* 面板数据解析（异步）：targets → 查询层取数 → 渲染序列；支持对比昨日 + 事件标注 */
  const resolveTargets = (p, cb) => {
    const xs = rangeTicks(dashState.cfg.range);
    const tgs = (p.targets || []).filter(t => t.metric);
    const compare = !!dashState.cfg.compareYesterday && p.type !== 'gauge' && p.type !== 'stat';
    const out = [];
    let pending = tgs.length * (compare ? 2 : 1);
    let annotations = null;
    let finished = false;
    const finish = () => {
      if (finished || pending > 0 || annotations === null) return;
      finished = true;
      cb({ xs, targets: out, annotations });
    };
    const onSeries = (t, suffix, dashed) => (err, res) => {
      (res && res.series ? res.series : []).forEach(s => {
        out.push({
          name: suffix ? s.name + suffix : s.name,
          data: s.data,
          color: s.color || panelColor(t.metric),
          axis: s.axis === 'right' ? 'right' : 'left',
          type: s.type || 'line',
          unit: s.unit || '',
          dashed,
        });
      });
      pending -= 1;
      finish();
    };
    if (!tgs.length) { pending = 0; annotations = []; finish(); return; }
    tgs.forEach(t => {
      const q = { metric: t.metric, scope: p.scope, range: dashState.cfg.range, shift: '', name: t.name, color: t.color, axis: t.axis, type: t.type, unit: t.unit, agg: t.agg || '', groupBy: t.groupBy || '', dbType: t.dbType || '' };
      fetchSeries(q, onSeries(t, '', false));
      if (compare) fetchSeries({ ...q, shift: '24h' }, onSeries(t, '（昨日）', true));
    });
    fetchAnnotations(dashState.cfg.range, (err, anns) => {
      const annCfg = p.annotations || {};
      annotations = annCfg.enable && Array.isArray(annCfg.types)
        ? (anns || []).filter(a => annCfg.types.includes(a.type))
        : [];
      finish();
    });
  };

  const panelHTML = (p) => {
    const tgs = (p.targets || []).filter(t => t.metric);
    const scopeName = p.scope === 'global' ? '全局' : ((CLUSTERS.find(c => c.id === p.scope) || {}).name || p.scope);
    const drill = p.drilldown && p.drilldown.targetId;
    return `<div class="panel" style="--w:${p.w || 6}" data-id="${p.id}">
      <div class="panel-bar">
        <span class="panel-title">${p.title}</span>
        ${drill ? '<span class="panel-drill" title="点击图表可下钻">↗</span>' : ''}
        <span class="panel-meta">${tgs.length} 个指标 · ${scopeName}</span>
        <button class="panel-menu-btn" title="面板操作">⋯</button>
      </div>
      <div class="panel-chart" id="pc-${p.id}"></div>
    </div>`;
  };

  /* 阈值命中颜色（Grafana steps 语义：取满足 val >= value 的最高档） */
  const thresholdColor = (steps, val) => {
    let c = '#1d2b45';
    (steps || []).forEach(s => { if (val >= s.value) c = s.color; });
    return c;
  };

  /* Stat 单值面板：大数值（阈值色）+ 迷你 sparkline；idSuffix 用于隔离预览区与大盘本体的元素 id */
  const renderStatPanel = (p, targets, el, idSuffix, sparkInit) => {
    const t0 = targets[0];
    const val = t0.data[t0.data.length - 1];
    const color = thresholdColor((p.thresholds && p.thresholds.steps) || [], val);
    el.innerHTML = `
      <div class="stat-panel">
        <div class="stat-val" style="color:${color}">${val}<span class="stat-unit">${t0.unit || ''}</span></div>
        <div class="stat-spark" id="pc-${p.id}${idSuffix || ''}-spark"></div>
      </div>`;
    (sparkInit || mkChart)($('#pc-' + p.id + (idSuffix || '') + '-spark'), {
      grid: { left: 2, right: 2, top: 4, bottom: 2 },
      xAxis: { type: 'category', show: false, boundaryGap: false, data: Array.from({ length: t0.data.length }, (_, i) => i) },
      yAxis: { type: 'value', show: false },
      tooltip: { show: false },
      series: [{
        type: 'line', smooth: true, symbol: 'none', data: t0.data,
        lineStyle: { width: 1.5, color: t0.color },
        areaStyle: { color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [{ offset: 0, color: t0.color + '33' }, { offset: 1, color: t0.color + '05' }]) },
      }],
    });
  };

  /* 下钻：点击图表跳转集群/实例详情，携带时间上下文 */
  const bindDrilldown = (chart, p) => {
    const d = p.drilldown;
    if (!d || !d.targetId) return;
    chart.on('click', () => {
      window._drillContext = { range: dashState.cfg.range, scope: p.scope };
      if (d.kind === 'cluster') location.hash = `#/cluster/${d.targetId}`;
      else if (d.kind === 'instance') {
        const cl = CLUSTERS.find(c => c.instances.some(i => i.id === d.targetId));
        if (cl) location.hash = `#/instance/${cl.id}/${d.targetId}`;
      }
    });
  };

  /* 表格面板：指标汇总（当前 / 最小 / 最大 / 平均），阈值着色 */
  const renderTablePanel = (p, targets, el) => {
    const steps = (p.thresholds && p.thresholds.steps) || [];
    const rows = targets.map(t => {
      const cur = t.data[t.data.length - 1];
      const min = Math.min(...t.data);
      const max = Math.max(...t.data);
      const avg = Math.round(t.data.reduce((a, b) => a + b, 0) / t.data.length);
      const color = thresholdColor(steps, cur);
      return `<tr>
        <td><span class="tp-dot" style="background:${t.color}"></span>${t.name}</td>
        <td class="tp-cur" style="color:${color}">${cur}${t.unit || ''}</td>
        <td>${min}</td><td>${max}</td><td>${avg}</td>
      </tr>`;
    });
    el.innerHTML = `<div class="table-panel"><table>
      <thead><tr><th>指标</th><th>当前值</th><th>最小</th><th>最大</th><th>平均</th></tr></thead>
      <tbody>${rows.join('')}</tbody>
    </table></div>`;
  };

  const renderPanelChart = (p) => {
    const el = $('#pc-' + p.id); if (!el) return;
    const type = p.type;
    resolveTargets(p, ({ xs, targets, annotations }) => {
      if (!el.isConnected || !targets.length) return; // 异步返回时面板可能已重绘
      if (type === 'gauge') {
        const t0 = targets[0];
        mkChart(el, gaugeOpt(p.title, t0.data[t0.data.length - 1], t0.unit));
        return;
      }
      if (type === 'stat') { renderStatPanel(p, targets, el, ''); return; }
      if (type === 'table') { renderTablePanel(p, targets, el); return; }
      if (type === 'bar') targets.forEach(t => { t.type = 'bar'; }); // 面板级柱状图覆盖所有序列
      const chart = mkChart(el, tsChartOpt(p.title, xs, targets, {
        legend: p.legend !== false,
        thresholds: (p.thresholds && p.thresholds.steps) || [],
        annotations,
        style: p.style || {},
      }));
      bindDrilldown(chart, p);
    });
  };

  const panelColor = (metric) => ({
    qps: '#006aff', cpu: '#00a3e0', mem: '#7a5af8', sessions: '#ff9500',
    slow_sql: '#7a5af8', lock_wait: '#f53f3f', disk: '#00b365', repl_delay: '#f53f3f',
  }[metric] || '#006aff');

  const gaugeOpt = (title, val, unit) => ({
    series: [{ type: 'gauge', startAngle: 210, endAngle: -30, min: 0, max: 100,
      radius: '92%', center: ['50%', '58%'],
      progress: { show: true, width: 10, itemStyle: { color: new echarts.graphic.LinearGradient(0, 0, 1, 0, [{ offset: 0, color: '#006aff' }, { offset: 1, color: '#f53f3f' }]) } },
      axisLine: { lineStyle: { width: 10, color: [[1, '#eef1f8']] } },
      axisTick: { show: false }, splitLine: { show: false }, axisLabel: { show: false }, pointer: { show: false },
      detail: { valueAnimation: true, formatter: `{value}${unit || ''}`, color: '#1d2b45', fontSize: 20, fontWeight: 700, offsetCenter: [0, '12%'] },
      title: { show: true, offsetCenter: [0, '46%'], color: '#8a97ad', fontSize: 11 },
      data: [{ value: val, name: title }],
    }],
  });

  function closePanelMenu() { const m = document.getElementById('panelPopover'); if (m) m.remove(); }
  function openPanelMenu(id, anchor) {
    closePanelMenu();
    const r = anchor.getBoundingClientRect();
    const menu = document.createElement('div');
    menu.className = 'dash-card-popover';
    menu.id = 'panelPopover';
    menu.style.top = (r.bottom + window.scrollY + 4) + 'px';
    menu.style.left = Math.max(8, r.right + window.scrollX - 156) + 'px';
    menu.innerHTML = `
      <button class="exp-more-item" data-act="view">查看</button>
      <button class="exp-more-item" data-act="edit">编辑</button>
      <button class="exp-more-item" data-act="del">删除</button>`;
    document.body.appendChild(menu);
    menu.querySelectorAll('[data-act]').forEach(b => b.onclick = (e) => {
      e.stopPropagation(); const act = b.dataset.act; closePanelMenu();
      if (act === 'view') openZoom(id);
      else if (act === 'edit') openAddPanel(id);
      else if (act === 'del') {
        const p = dashState.panels.find(x => x.id === id);
        confirmDialog({
          title: '删除面板', okText: '删除', danger: true,
          message: '确定删除面板「' + ((p && p.title) || '') + '」吗？该操作不可恢复。',
          onOk: () => {
            const idx = dashState.panels.findIndex(x => x.id === id);
            if (idx >= 0) dashState.panels.splice(idx, 1);
            savePanels(dashState.panels); renderShell();
          },
        });
      }
    });
    setTimeout(() => {
      const onDown = (e) => { if (!menu.contains(e.target) && e.target !== anchor) { closePanelMenu(); document.removeEventListener('mousedown', onDown); } };
      document.addEventListener('mousedown', onDown);
    }, 0);
  }

  const bindPanelOps = () => {
    const grid = $('#dashGrid'); if (!grid) return;
    grid.querySelectorAll('.panel').forEach(card => {
      const id = card.dataset.id;
      const mb = card.querySelector('.panel-menu-btn');
      if (mb) mb.onclick = (e) => { e.stopPropagation(); openPanelMenu(id, mb); };
    });
  };

  const targetRowHTML = (t, i) => `
    <div class="dap-target" data-i="${i}">
      <span class="dap-t-idx">${i + 1}</span>
      <select data-f="metric">${METRIC_LIB.map(m => `<option value="${m.id}" ${t.metric === m.id ? 'selected' : ''}>${m.name}</option>`).join('')}</select>
      <input data-f="name" placeholder="名称" value="${t.name || ''}">
      <input type="color" data-f="color" value="${t.color || '#006aff'}" title="颜色">
      <select data-f="axis">
        <option value="left" ${t.axis !== 'right' ? 'selected' : ''}>左 Y 轴</option>
        <option value="right" ${t.axis === 'right' ? 'selected' : ''}>右 Y 轴</option>
      </select>
      <select data-f="type">
        ${[['line', '折线'], ['bar', '柱状'], ['area', '面积'], ['points', '点']].map(([v, l]) => `<option value="${v}" ${t.type === v ? 'selected' : ''}>${l}</option>`).join('')}
      </select>
      <select data-f="agg" title="聚合">
        ${[['', '原始'], ['avg', '平均'], ['max', '最大'], ['min', '最小'], ['last', '末值'], ['p95', 'P95']].map(([v, l]) => `<option value="${v}" ${(t.agg || '') === v ? 'selected' : ''}>${l}</option>`).join('')}
      </select>
      <select data-f="groupBy" title="分组">
        ${[['', '不分组'], ['cluster', '按集群'], ['instance', '按实例']].map(([v, l]) => `<option value="${v}" ${(t.groupBy || '') === v ? 'selected' : ''}>${l}</option>`).join('')}
      </select>
      <button class="dap-t-del" data-del="${i}" title="删除指标">✕</button>
    </div>`;

  const ANN_TYPES = [['release', '发布'], ['switch', '主备切换'], ['alert', '告警']];

  const thresholdRowHTML = (s) => `
    <div class="dap-threshold">
      <input type="number" data-f="tval" value="${s.value}" placeholder="阈值" step="any">
      <input type="color" data-f="tcolor" value="${s.color || '#ff9500'}" title="颜色">
      <button class="dap-t-del" title="删除阈值">✕</button>
    </div>`;

  const openAddPanel = (editId) => {
    const editing = !!editId;
    const p = editing
      ? dashState.panels.find(x => x.id === editId)
      : { title: '', type: 'timeseries', legend: true, scope: 'global', w: 6, style: Object.assign({}, DEFAULT_PANEL_STYLE), thresholds: { steps: [] }, annotations: { enable: false, types: ['release', 'switch', 'alert'] }, drilldown: null };
    const targets = (p.targets && p.targets.length)
      ? p.targets.map(t => ({ metric: t.metric, name: t.name || '', color: t.color || '', axis: t.axis || 'left', type: t.type || 'line', agg: t.agg || '', groupBy: t.groupBy || '' }))
      : [{ metric: 'qps', name: '', color: '', axis: 'left', type: 'line', agg: '', groupBy: '' }];
    const steps = (p.thresholds && p.thresholds.steps) || [];
    const ann = p.annotations || { enable: false, types: ['release', 'switch', 'alert'] };
    const drill = p.drilldown || null;
    const pst = normalizeStyle(p.style);
    /* 分组折叠初始态：编辑时含非默认配置的区块自动展开 */
    const advOpen = !!(steps.length || ann.enable || drill);
    const styleOpen = Object.keys(DEFAULT_PANEL_STYLE).some(k => pst[k] !== DEFAULT_PANEL_STYLE[k]);
    const secCls = (open) => 'dap-section' + (open ? '' : ' closed');

    const overlay = document.createElement('div');
    overlay.className = 'dash-popover-overlay';
    overlay.innerHTML = `<div class="dash-add-popover">
      <div class="dap-head">${editing ? '编辑面板' : '添加面板'}</div>
      <div class="dap-layout" id="dapLayout">
      <div class="dap-body">
        <div class="${secCls(true)}">
          <div class="dap-section-head"><span class="dap-section-arrow">▾</span>基础设置</div>
          <div class="dap-section-body">
            <label>标题<input id="dapTitle" value="${p.title || ''}" placeholder="如：交易库 CPU 与 QPS"></label>
            <div class="dap-row">
              <label>图表类型<select id="dapType">
                <option value="timeseries" ${p.type === 'timeseries' ? 'selected' : ''}>时序图（多指标同轴）</option>
                <option value="bar" ${p.type === 'bar' ? 'selected' : ''}>柱状图</option>
                <option value="stat" ${p.type === 'stat' ? 'selected' : ''}>单值 Stat（取第 1 个指标）</option>
                <option value="gauge" ${p.type === 'gauge' ? 'selected' : ''}>仪表盘（取第 1 个指标）</option>
                <option value="table" ${p.type === 'table' ? 'selected' : ''}>表格（指标汇总）</option>
              </select></label>
              <label>范围<select id="dapScope"><option value="global" ${!p.scope || p.scope === 'global' ? 'selected' : ''}>全局</option>${CLUSTERS.map(c => `<option value="${c.id}" ${p.scope === c.id ? 'selected' : ''}>${c.name}</option>`).join('')}</select></label>
            </div>
            <label class="dap-check"><input type="checkbox" id="dapLegend" ${p.legend !== false ? 'checked' : ''}> 显示图例</label>
          </div>
        </div>
        <div class="${secCls(true)}">
          <div class="dap-section-head"><span class="dap-section-arrow">▾</span>指标序列<span class="dap-section-sub">共享时间轴 · 可叠加多个指标、分别挂左/右 Y 轴</span></div>
          <div class="dap-section-body">
            <div class="dap-targets" id="dapTargets">${targets.map(targetRowHTML).join('')}</div>
            <button class="btn sm" id="dapAddTarget">＋ 添加指标</button>
          </div>
        </div>
        <div class="${secCls(advOpen)}">
          <div class="dap-section-head"><span class="dap-section-arrow">▾</span>高级显示<span class="dap-section-sub">阈值 / 事件标注 / 下钻</span></div>
          <div class="dap-section-body">
            <div class="dap-subhead">阈值 · 超限分段着色 + 阈值虚线（升序生效）</div>
            <div class="dap-targets" id="dapThresholds">${steps.map(thresholdRowHTML).join('')}</div>
            <button class="btn sm" id="dapAddThreshold">＋ 添加阈值</button>
            <div class="dap-subhead">事件标注 · 发布/切换/告警叠加在时间轴上</div>
            <label class="dap-check"><input type="checkbox" id="dapAnnEnable" ${ann.enable ? 'checked' : ''}> 启用事件标注</label>
            <div class="dap-ann-types" id="dapAnnTypes" style="${ann.enable ? '' : 'display:none'}">
              ${ANN_TYPES.map(([v, l]) => `<label class="dap-check"><input type="checkbox" value="${v}" ${(ann.types || []).includes(v) ? 'checked' : ''}> ${l}</label>`).join('')}
            </div>
            <div class="dap-subhead">下钻 · 点击图表跳转详情页（携带时间上下文）</div>
            <label>下钻目标<select id="dapDrill">
              <option value="" ${!drill ? 'selected' : ''}>无</option>
              <optgroup label="集群详情">
                ${CLUSTERS.map(c => `<option value="cluster:${c.id}" ${drill && drill.kind === 'cluster' && drill.targetId === c.id ? 'selected' : ''}>${c.name}</option>`).join('')}
              </optgroup>
              <optgroup label="实例详情">
                ${CLUSTERS.flatMap(c => c.instances.map(i => `<option value="instance:${i.id}" ${drill && drill.kind === 'instance' && drill.targetId === i.id ? 'selected' : ''}>${i.name}</option>`)).join('')}
              </optgroup>
            </select></label>
          </div>
        </div>
        <div class="${secCls(styleOpen)}">
          <div class="dap-section-head"><span class="dap-section-arrow">▾</span>图表样式<span class="dap-section-sub">线宽 / 填充 / 堆叠 / 图例 / 坐标轴范围等</span></div>
          <div class="dap-section-body">
            <div class="dap-style-grid">
          <label>线宽<select data-s="lineWidth">${[1,2,3].map(v=>`<option value="${v}" ${pst.lineWidth==v?'selected':''}>${v}px</option>`).join('')}</select></label>
          <label>填充<input type="range" data-s="fill" min="0" max="80" value="${pst.fill}"></label>
          <label>点大小<select data-s="points">${[[0,'无'],[3,'3'],[5,'5'],[8,'8']].map(([v,l])=>`<option value="${v}" ${pst.points==v?'selected':''}>${l}</option>`).join('')}</select></label>
          <label class="dap-check"><input type="checkbox" data-s="connectNulls" ${pst.connectNulls?'checked':''}> 连接空值</label>
          <label>堆叠<select data-s="stack">${[['none','不堆叠'],['normal','常规'],['percent','百分比']].map(([v,l])=>`<option value="${v}" ${pst.stack===v?'selected':''}>${l}</option>`).join('')}</select></label>
          <label>图例<select data-s="legend">${[['top','顶部'],['bottom','底部'],['right','右侧'],['hide','隐藏']].map(([v,l])=>`<option value="${v}" ${pst.legend===v?'selected':''}>${l}</option>`).join('')}</select></label>
          <label>Y轴最小<input type="number" data-s="yMin" value="${pst.yMin!=null?pst.yMin:''}" placeholder="自动"></label>
          <label>Y轴最大<input type="number" data-s="yMax" value="${pst.yMax!=null?pst.yMax:''}" placeholder="自动"></label>
          <label>小数位<select data-s="decimals">${[['null','自动'],['0','0'],['1','1'],['2','2']].map(([v,l])=>`<option value="${v}" ${String(pst.decimals)===v?'selected':''}>${l}</option>`).join('')}</select></label>
          <label>单位覆盖<input type="text" data-s="unit" value="${pst.unit||''}" placeholder="如 %"></label>
            </div>
          </div>
        </div>
      </div>
      <div class="dap-preview">
        <div class="dap-preview-head"><span class="dap-preview-title">实时预览</span><button class="dap-preview-toggle" id="dapPreviewToggle" title="收起/展开预览">收起 ▾</button></div>
        <div class="panel-chart" id="dapPreviewChart"></div>
      </div>
      </div>
      <div class="dap-foot">
        <button class="btn sm" id="dapCancel">取消</button>
        <button class="btn sm primary" id="dapOk">${editing ? '保存' : '添加'}</button>
      </div>
    </div>`;
    document.body.appendChild(overlay);
    const previewCharts = [];
    let previewCollapsed = false;
    /* 分组折叠：仅切换 display，不重建 DOM，collectForm 读值不受影响 */
    overlay.querySelectorAll('.dap-section-head').forEach(h => h.onclick = () => h.parentElement.classList.toggle('closed'));
    const close = () => { if (previewTimer) clearTimeout(previewTimer); previewCharts.forEach(c => { try { c.dispose(); } catch (e) {} }); previewCharts.length = 0; overlay.remove(); };
    overlay.onclick = (e) => { if (e.target === overlay) close(); };
    $('#dapCancel').onclick = close;

    const bindDel = (sel) => $$(sel + ' .dap-t-del').forEach(b => b.onclick = () => b.closest(sel === '#dapThresholds' ? '.dap-threshold' : '.dap-target').remove());

    const addTargetRow = () => {
      const row = document.createElement('div');
      row.innerHTML = targetRowHTML({ metric: 'qps', name: '', color: '', axis: 'left', type: 'line', agg: '', groupBy: '' }, $('#dapTargets').children.length);
      const el = row.firstElementChild;
      $('#dapTargets').appendChild(el);
      el.querySelector('[data-del]').onclick = () => { el.remove(); schedulePreview(); };
      schedulePreview();
    };
    const addThresholdRow = () => {
      const row = document.createElement('div');
      row.innerHTML = thresholdRowHTML({ value: '', color: '#ff9500' });
      const el = row.firstElementChild;
      $('#dapThresholds').appendChild(el);
      el.querySelector('.dap-t-del').onclick = () => { el.remove(); schedulePreview(); };
      schedulePreview();
    };
    $('#dapAddTarget').onclick = addTargetRow;
    $('#dapAddThreshold').onclick = addThresholdRow;
    bindDel('#dapThresholds');
    bindDel('#dapTargets');
    $('#dapAnnEnable').onchange = (e) => { $('#dapAnnTypes').style.display = e.target.checked ? '' : 'none'; schedulePreview(); };

    /* 收集表单为草稿面板对象（保存与实时预览共用） */
    const collectForm = () => {
      const title = $('#dapTitle').value.trim() || '未命名面板';
      const type = $('#dapType').value;
      const scope = $('#dapScope').value;
      const legend = $('#dapLegend').checked;
      const nextTargets = $$('#dapTargets .dap-target').map(row => ({
        metric: row.querySelector('[data-f=metric]').value,
        name: row.querySelector('[data-f=name]').value.trim(),
        color: row.querySelector('[data-f=color]').value,
        axis: row.querySelector('[data-f=axis]').value,
        type: row.querySelector('[data-f=type]').value,
        agg: row.querySelector('[data-f=agg]').value,
        groupBy: row.querySelector('[data-f=groupBy]').value,
      }));
      if (!nextTargets.length) return null;
      const style = Object.assign({}, DEFAULT_PANEL_STYLE);
      $$('.dap-style-grid [data-s]').forEach(inp => {
        const f = inp.dataset.s;
        if (inp.type === 'checkbox') style[f] = inp.checked;
        else if (f === 'yMin' || f === 'yMax') style[f] = inp.value === '' ? null : +inp.value;
        else if (f === 'decimals') style[f] = inp.value === 'null' ? null : +inp.value;
        else if (f === 'lineWidth' || f === 'points' || f === 'fill') style[f] = +inp.value;
        else style[f] = inp.value;
      });
      const thresholds = {
        steps: $$('#dapThresholds .dap-threshold').map(row => ({
          value: parseFloat(row.querySelector('[data-f=tval]').value),
          color: row.querySelector('[data-f=tcolor]').value,
        })).filter(s => !isNaN(s.value)),
      };
      const annotations = {
        enable: $('#dapAnnEnable').checked,
        types: $$('#dapAnnTypes input:checked').map(i => i.value),
      };
      const drillVal = $('#dapDrill').value;
      const drilldown = drillVal
        ? (drillVal.startsWith('cluster:')
          ? { kind: 'cluster', targetId: drillVal.slice('cluster:'.length) }
          : { kind: 'instance', targetId: drillVal.slice('instance:'.length) })
        : null;
      return { id: 'draft', title, type, scope, legend, targets: nextTargets, style, thresholds, annotations, drilldown };
    };

    /* 实时预览：以草稿面板取数渲染，实例局部管理避免污染全局 chartInstances */
    const disposePreviewCharts = () => { previewCharts.forEach(c => { try { c.dispose(); } catch (e) {} }); previewCharts.length = 0; };
    const previewInit = (el, option) => { const c = echarts.init(el); c.setOption(option); previewCharts.push(c); return c; };
    const renderPreview = () => {
      if (!overlay.isConnected) return;
      const el = $('#dapPreviewChart'); if (!el) return;
      if (previewCollapsed) return; // 收起态跳过渲染，展开时主动重绘
      const draft = collectForm();
      if (!draft) { disposePreviewCharts(); el.innerHTML = '<div class="dap-preview-empty">请至少添加一个指标</div>'; return; }
      resolveTargets(draft, ({ xs, targets, annotations }) => {
        if (!overlay.isConnected || !$('#dapPreviewChart')) return;
        disposePreviewCharts();
        el.innerHTML = '';
        if (!targets.length) { el.innerHTML = '<div class="dap-preview-empty">暂无数据</div>'; return; }
        const type = draft.type;
        if (type === 'gauge') { previewInit(el, gaugeOpt(draft.title, targets[0].data[targets[0].data.length - 1], targets[0].unit)); return; }
        if (type === 'stat') { renderStatPanel(draft, targets, el, '-pv', previewInit); return; }
        if (type === 'table') { renderTablePanel(draft, targets, el); return; }
        if (type === 'bar') targets.forEach(t => { t.type = 'bar'; });
        previewInit(el, tsChartOpt(draft.title, xs, targets, {
          legend: draft.legend !== false,
          thresholds: (draft.thresholds && draft.thresholds.steps) || [],
          annotations,
          style: draft.style || {},
        }));
      });
    };
    let previewTimer = null;
    const schedulePreview = () => { clearTimeout(previewTimer); previewTimer = setTimeout(renderPreview, 300); };

    /* 表单变更 → 防抖刷新预览（新增/删除行另行触发） */
    overlay.querySelector('.dap-body').addEventListener('input', schedulePreview);
    overlay.querySelector('.dap-body').addEventListener('change', schedulePreview);
    $('#dapPreviewToggle').onclick = () => {
      previewCollapsed = !previewCollapsed;
      $('#dapLayout').classList.toggle('preview-closed', previewCollapsed);
      $('#dapPreviewToggle').textContent = previewCollapsed ? '展开 ▸' : '收起 ▾';
      if (previewCollapsed) disposePreviewCharts();
      else renderPreview(); // 重新展开时列宽变化，统一重绘保证尺寸正确
    };
    renderPreview();

    $('#dapOk').onclick = () => {
      const draft = collectForm();
      if (!draft) { addTargetRow(); return; }
      const w = (p.w || 6);
      if (editing) {
        Object.assign(p, draft);
      } else {
        dashState.panels.push(Object.assign({ id: 'p' + Date.now(), w, visible: true }, draft));
      }
      savePanels(dashState.panels);
      close(); renderShell();
    };
  };

  const openZoom = (id) => {
    const p = dashState.panels.find(x => x.id === id);
    const overlay = document.createElement('div');
    overlay.className = 'panel-fullscreen';
    overlay.innerHTML = `<div class="pf-bar"><span>${p.title}</span><button class="btn sm" id="pfClose">关闭</button></div><div class="pf-chart" id="pfChart"></div>`;
    document.body.appendChild(overlay);
    const close = () => { chartInstances.forEach(c => c.dispose()); chartInstances.length = 0; overlay.remove(); drawDashBody(); };
    overlay.onclick = (e) => { if (e.target === overlay) close(); };
    $('#pfClose').onclick = close;
    const type = p.type;
    resolveTargets(p, ({ xs, targets, annotations }) => {
      if (!overlay.isConnected || !targets.length) return;
      chartInstances.length = 0;
      if (type === 'gauge') mkChart($('#pfChart'), gaugeOpt(p.title, targets[0].data[targets[0].data.length - 1], targets[0].unit));
      else if (type === 'stat') renderStatPanel(p, targets, $('#pfChart'));
      else if (type === 'table') renderTablePanel(p, targets, $('#pfChart'));
      else {
        if (type === 'bar') targets.forEach(t => { t.type = 'bar'; });
        const chart = mkChart($('#pfChart'), tsChartOpt(p.title, xs, targets, {
          legend: p.legend !== false,
          thresholds: (p.thresholds && p.thresholds.steps) || [],
          annotations,
          style: p.style || {},
        }));
        bindDrilldown(chart, p);
      }
    });
  };

  renderShell();
}

/* ================= 路由 ================= */
const $$ = (s, p) => Array.from((p || document).querySelectorAll(s));
function router() {
  const hash = location.hash || '#/overview';
  if (dashTimer) { clearInterval(dashTimer); dashTimer = null; } // 离开大盘页时停掉自动刷新
  chartInstances.forEach(c => { try { c.dispose(); } catch (e) {} });
  chartInstances.length = 0;
  renderNav(hash);
  const m = hash.match(/^#\/(\w+)(?:\/([^/]+))?(?:\/([^/]+))?/);
  const page = m ? m[1] : 'overview';
  if (page === 'overview') renderOverview();
  else if (page === 'clusters') renderClusters(m && m[2]);
  else if (page === 'cluster') renderClusterDetail(m[2]);
  else if (page === 'instance') renderInstanceDetail(m[2], m[3]);
  else if (page === 'hosts') renderHosts();
  else if (page === 'dashboards') renderDashboards();
  else if (page === 'dashboard') renderDashboard(m && m[2]);
  else renderOverview();
  $('.content').scrollTop = 0;
}
window.addEventListener('hashchange', router);
window.addEventListener('resize', () => chartInstances.forEach(c => c.resize()));
document.addEventListener('click', (e) => {
  const row = e.target.closest('[data-toggle]');
  if (row) {
    const card = row.closest('.cluster-card');
    if (card) card.classList.toggle('open');
  }
});

initChat();
router();
