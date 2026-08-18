/* ================= Mock 数据（后端就绪前的前端数据源） =================
 * 平台主攻两类库型：
 *  - PostgreSQL：经典集群（主备流复制 / Patroni），管理维度 = 实例 + Database + 复制链路
 *  - OceanBase：集群 → Zone → OBServer → Unit 资源池 → 租户 → Database
 */
import type { Cluster, MetricDef } from './types';

export const DB_TYPES = [
  { type: 'pg', name: 'PostgreSQL', icon: '🐘', total: 42, alert: 5 },
  { type: 'oceanbase', name: 'OceanBase', icon: '🌊', total: 36, alert: 4 },
];

export const TOP_ANOMALY = [
  { name: 'trade_tenant @ prod-ob-core-01', cluster: 'prod-ob-core-01', score: 96, issue: '租户 CPU 13.1/14C · 慢SQL激增', inst: 'trade_tenant' },
  { name: 'pg-order-01（主库）', cluster: 'prod-pg-order-01', score: 88, issue: '复制延迟 850ms · WAL 积压 1.2GB', inst: 'in-1e7f3' },
  { name: 'pay_tenant @ prod-ob-core-01', cluster: 'prod-ob-core-01', score: 82, issue: '租户内存水位 91%', inst: 'pay_tenant' },
  { name: 'analytics @ prod-pg-order-01', cluster: 'prod-pg-order-01', score: 74, issue: '连接数 410/500 · 慢查询堆积', inst: 'in-1e7f3' },
  { name: 'pg-report-02（备库）', cluster: 'prod-pg-report-02', score: 68, issue: 'autovacuum 积压 · 磁盘 78%', inst: 'in-7d2c4' },
];

export const SQL_ISSUES = [
  { name: '全表扫描', cnt: 23 },
  { name: '缺失索引', cnt: 18 },
  { name: '隐式类型转换', cnt: 12 },
  { name: '过度排序', cnt: 9 },
  { name: '深分页', cnt: 7 },
  { name: '冗余 JOIN', cnt: 5 },
];

export const SLOW_SQLS = [
  { sql: 'SELECT o.*, u.name FROM trade_order o JOIN user u ON o.uid = u.id WHERE o.status = ?', db: 'trade_tenant/trade_order', time: '12.8s', rows: '4,380,012', count: 342 },
  { sql: 'UPDATE stock_record SET qty = qty - ? WHERE sku_id = ? AND warehouse_id = ?', db: 'trade_tenant/inventory', time: '9.6s', rows: '1,203,550', count: 187 },
  { sql: 'SELECT COUNT(*) FROM access_log WHERE create_time BETWEEN ? AND ? GROUP BY path', db: 'log_tenant/access_log', time: '8.2s', rows: '9,881,204', count: 96 },
  { sql: 'SELECT * FROM payment_bill WHERE bill_no LIKE ? ORDER BY ctime DESC LIMIT ?', db: 'pay_tenant/payment', time: '6.4s', rows: '760,332', count: 64 },
  { sql: 'DELETE FROM session_token WHERE expire_at < ? AND app_id IN (?, ?, ?)', db: 'pg-order-01/auth', time: '5.1s', rows: '2,310,778', count: 41 },
];

/* ================= PostgreSQL 集群 ================= */
export const CLUSTERS: Cluster[] = [
  {
    id: 'c1', name: 'prod-pg-order-01', type: 'pg', version: 'PostgreSQL 15.6',
    desc: '交易核心集群 · 华东-可用区B', az: '华东-AZ-B', biz: '核心交易', nodes: 3, mode: 'Patroni 流复制（1主2备）',
    cpu: 52, mem: 67, conn: 926, qps: 8400, syncMode: 'Patroni · quorum（ANY 1 (pg2, pg3)）',
    instances: [
      { id: 'in-1e7f3', name: 'pg-order-01', role: '主库 Primary', ip: '10.20.2.11', port: 5432, status: 'ok', cpu: 45, mem: 62, conn: 320, ver: '15.6' },
      { id: 'in-9c3d7', name: 'pg-order-02', role: '备库 Standby', ip: '10.20.2.12', port: 5432, status: 'warn', cpu: 78, mem: 81, conn: 410, ver: '15.6' },
      { id: 'in-6b1a8', name: 'pg-order-03', role: '备库 Standby', ip: '10.20.2.13', port: 5432, status: 'ok', cpu: 33, mem: 58, conn: 196, ver: '15.6' },
    ],
    databases: [
      { name: 'trade_order', owner: 'app_rw', size: '1.8 TB', tables: 326, conn: 320, connLimit: 400, status: 'ok' },
      { name: 'user_center', owner: 'app_rw', size: '680 GB', tables: 214, conn: 190, connLimit: 300, status: 'ok' },
      { name: 'payment', owner: 'app_rw', size: '420 GB', tables: 158, conn: 120, connLimit: 200, status: 'ok' },
      { name: 'analytics', owner: 'report_etl', size: '2.9 TB', tables: 502, conn: 410, connLimit: 500, status: 'warn' },
    ],
    replicas: [
      { instance: 'pg-order-02', role: 'Standby（quorum）', delayMs: 850, walLag: '1.2 GB', status: 'warn' },
      { instance: 'pg-order-03', role: 'Standby（quorum）', delayMs: 120, walLag: '210 MB', status: 'ok' },
    ],
    params: [
      { name: 'shared_buffers', value: '32G', range: '128MB - 128G', desc: '共享缓冲区大小', status: 'ok' },
      { name: 'max_connections', value: '1000', range: '1 - 262143', desc: '最大连接数', status: 'ok' },
      { name: 'work_mem', value: '64MB', range: '64kB - 2G', desc: '排序/哈希操作内存', status: 'pending' },
      { name: 'max_wal_size', value: '16GB', range: '2GB - 64GB', desc: 'WAL 上限（两次检查点间）', status: 'ok' },
      { name: 'autovacuum', value: 'on', range: 'on / off', desc: '自动清理开关', status: 'ok' },
    ],
  },
  {
    id: 'c2', name: 'prod-pg-report-02', type: 'pg', version: 'PostgreSQL 15.6',
    desc: '报表分析集群 · 华东-可用区B', az: '华东-AZ-B', biz: '报表分析', nodes: 2, mode: '流复制（1主1备）',
    cpu: 38, mem: 55, conn: 460, qps: 3100, syncMode: '异步流复制',
    instances: [
      { id: 'in-3f0a1', name: 'pg-report-01', role: '主库 Primary', ip: '10.20.2.21', port: 5432, status: 'ok', cpu: 35, mem: 52, conn: 260, ver: '15.6' },
      { id: 'in-7d2c4', name: 'pg-report-02', role: '备库 Standby', ip: '10.20.2.22', port: 5432, status: 'ok', cpu: 29, mem: 48, conn: 110, ver: '15.6' },
    ],
    databases: [
      { name: 'bi_report', owner: 'report_rw', size: '1.1 TB', tables: 388, conn: 210, connLimit: 300, status: 'ok' },
      { name: 'metrics_cache', owner: 'report_ro', size: '240 GB', tables: 96, conn: 60, connLimit: 150, status: 'ok' },
    ],
    replicas: [
      { instance: 'pg-report-02', role: 'Standby（异步）', delayMs: 240, walLag: '480 MB', status: 'ok' },
    ],
    params: [
      { name: 'shared_buffers', value: '16G', range: '128MB - 64G', desc: '共享缓冲区大小', status: 'ok' },
      { name: 'max_connections', value: '600', range: '1 - 262143', desc: '最大连接数', status: 'ok' },
      { name: 'max_parallel_workers_per_gather', value: '4', range: '0 - 64', desc: '单查询并行度', status: 'pending' },
    ],
  },
  /* ================= OceanBase 集群 ================= */
  {
    id: 'c3', name: 'prod-ob-core-01', type: 'oceanbase', version: 'OceanBase 4.2.1',
    desc: '核心账务集群 · 华东可用区A/B/C', az: '华东-AZ-A/B/C', biz: '核心账务', nodes: 6, mode: '3 Zone × 2 OBServer · Paxos',
    cpu: 68, mem: 72, conn: 1240, qps: 18600,
    zones: ['ZONE1', 'ZONE2', 'ZONE3'],
    instances: [
      { id: 'obc-z1-1', name: 'observer-zone1-01', role: 'OBServer', zone: 'ZONE1', ip: '10.40.1.11', port: 2881, status: 'ok', cpu: 71, mem: 74, conn: 420, ver: '4.2.1' },
      { id: 'obc-z1-2', name: 'observer-zone1-02', role: 'OBServer', zone: 'ZONE1', ip: '10.40.1.12', port: 2881, status: 'ok', cpu: 55, mem: 66, conn: 380, ver: '4.2.1' },
      { id: 'obc-z2-1', name: 'observer-zone2-01', role: 'OBServer', zone: 'ZONE2', ip: '10.40.2.11', port: 2881, status: 'warn', cpu: 82, mem: 85, conn: 300, ver: '4.2.1' },
      { id: 'obc-z2-2', name: 'observer-zone2-02', role: 'OBServer', zone: 'ZONE2', ip: '10.40.2.12', port: 2881, status: 'ok', cpu: 49, mem: 63, conn: 90, ver: '4.2.1' },
      { id: 'obc-z3-1', name: 'observer-zone3-01', role: 'OBServer', zone: 'ZONE3', ip: '10.40.3.11', port: 2881, status: 'ok', cpu: 52, mem: 68, conn: 30, ver: '4.2.1' },
      { id: 'obc-z3-2', name: 'observer-zone3-02', role: 'OBServer', zone: 'ZONE3', ip: '10.40.3.12', port: 2881, status: 'ok', cpu: 47, mem: 61, conn: 20, ver: '4.2.1' },
    ],
    tenants: [
      {
        id: 't-sys', name: 'sys', kind: 'sys', mode: 'mysql', primaryZone: 'RANDOM',
        locality: 'F@ZONE1,F@ZONE2,F@ZONE3', unitNum: 1,
        maxCpu: 6, usedCpu: 1.4, maxMemGb: 48, usedMemGb: 11,
        storageUsed: '40 GB', storageTotal: '100 GB', status: 'ok',
        units: [
          { zone: 'ZONE1', observer: 'observer-zone1-01', maxCpu: 2, usedCpu: 0.5, maxMemGb: 16, usedMemGb: 3.8 },
          { zone: 'ZONE2', observer: 'observer-zone2-01', maxCpu: 2, usedCpu: 0.4, maxMemGb: 16, usedMemGb: 3.6 },
          { zone: 'ZONE3', observer: 'observer-zone3-01', maxCpu: 2, usedCpu: 0.5, maxMemGb: 16, usedMemGb: 3.6 },
        ],
        databases: [
          { name: 'oceanbase', tables: 0, size: '38 GB', conn: 12, status: 'ok' },
        ],
        whitelist: ['%'], connHint: 'mysql -h10.40.1.11 -P2881 -uroot@sys -p***',
      },
      {
        id: 't-trade', name: 'trade_tenant', kind: 'user', mode: 'mysql', primaryZone: 'ZONE1',
        locality: 'F@ZONE1,F@ZONE2,F@ZONE3', unitNum: 1,
        maxCpu: 14, usedCpu: 13.1, maxMemGb: 72, usedMemGb: 51,
        storageUsed: '2.1 TB', storageTotal: '4 TB', status: 'warn',
        units: [
          { zone: 'ZONE1', observer: 'observer-zone1-01', maxCpu: 14, usedCpu: 12.6, maxMemGb: 72, usedMemGb: 49 },
          { zone: 'ZONE2', observer: 'observer-zone2-01', maxCpu: 14, usedCpu: 13.4, maxMemGb: 72, usedMemGb: 52 },
          { zone: 'ZONE3', observer: 'observer-zone3-01', maxCpu: 14, usedCpu: 13.3, maxMemGb: 72, usedMemGb: 52 },
        ],
        databases: [
          { name: 'trade_order', tables: 326, size: '1.2 TB', conn: 180, status: 'ok' },
          { name: 'inventory', tables: 154, size: '640 GB', conn: 96, status: 'ok' },
          { name: 'seckill', tables: 48, size: '88 GB', conn: 210, status: 'warn' },
        ],
        whitelist: ['10.40.10.%', '10.40.11.%'], connHint: 'mysql -h10.40.1.21 -P2881 -utrade_rw@trade_tenant -p***',
      },
      {
        id: 't-pay', name: 'pay_tenant', kind: 'user', mode: 'mysql', primaryZone: 'ZONE2',
        locality: 'F@ZONE1,F@ZONE2,F@ZONE3', unitNum: 1,
        maxCpu: 12, usedCpu: 7.4, maxMemGb: 48, usedMemGb: 43.7,
        storageUsed: '860 GB', storageTotal: '1.5 TB', status: 'warn',
        units: [
          { zone: 'ZONE1', observer: 'observer-zone1-02', maxCpu: 12, usedCpu: 7.1, maxMemGb: 48, usedMemGb: 43.2 },
          { zone: 'ZONE2', observer: 'observer-zone2-02', maxCpu: 12, usedCpu: 7.6, maxMemGb: 48, usedMemGb: 44.1 },
          { zone: 'ZONE3', observer: 'observer-zone3-02', maxCpu: 12, usedCpu: 7.5, maxMemGb: 48, usedMemGb: 43.8 },
        ],
        databases: [
          { name: 'payment', tables: 96, size: '620 GB', conn: 140, status: 'ok' },
          { name: 'billing', tables: 61, size: '240 GB', conn: 52, status: 'ok' },
        ],
        whitelist: ['10.40.20.%'], connHint: 'mysql -h10.40.2.12 -P2881 -upay_rw@pay_tenant -p***',
      },
    ],
    params: [
      { name: 'memory_limit', value: '96G', range: '16G - 物理内存', desc: 'OBServer 内存上限', status: 'ok' },
      { name: 'system_memory', value: '10G', range: '2G - memory_limit', desc: '系统预留内存（500 租户）', status: 'ok' },
      { name: 'datafile_max_size', value: '512G', range: '磁盘容量内', desc: '数据文件上限', status: 'ok' },
      { name: 'enable_syslog_recycle', value: 'true', range: 'true / false', desc: '系统日志自动回收', status: 'pending' },
      { name: 'merger_check_interval', value: '20m', range: '1m - 24h', desc: '合并检查间隔', status: 'ok' },
    ],
  },
  {
    id: 'c4', name: 'prod-ob-log-01', type: 'oceanbase', version: 'OceanBase 4.2.1',
    desc: '日志分析集群 · 华东可用区A/B/C', az: '华东-AZ-A/B/C', biz: '日志分析', nodes: 3, mode: '3 Zone × 1 OBServer · Paxos',
    cpu: 34, mem: 48, conn: 380, qps: 6200,
    zones: ['ZONE1', 'ZONE2', 'ZONE3'],
    instances: [
      { id: 'obl-z1-1', name: 'obs-log-zone1-01', role: 'OBServer', zone: 'ZONE1', ip: '10.41.1.11', port: 2881, status: 'ok', cpu: 36, mem: 49, conn: 160, ver: '4.2.1' },
      { id: 'obl-z2-1', name: 'obs-log-zone2-01', role: 'OBServer', zone: 'ZONE2', ip: '10.41.2.11', port: 2881, status: 'ok', cpu: 31, mem: 45, conn: 120, ver: '4.2.1' },
      { id: 'obl-z3-1', name: 'obs-log-zone3-01', role: 'OBServer', zone: 'ZONE3', ip: '10.41.3.11', port: 2881, status: 'ok', cpu: 29, mem: 44, conn: 100, ver: '4.2.1' },
    ],
    tenants: [
      {
        id: 't-log', name: 'log_tenant', kind: 'user', mode: 'mysql', primaryZone: 'ZONE1',
        locality: 'F@ZONE1,F@ZONE2,F@ZONE3', unitNum: 1,
        maxCpu: 6, usedCpu: 2.1, maxMemGb: 32, usedMemGb: 12,
        storageUsed: '3.2 TB', storageTotal: '8 TB', status: 'ok',
        units: [
          { zone: 'ZONE1', observer: 'obs-log-zone1-01', maxCpu: 6, usedCpu: 2.2, maxMemGb: 32, usedMemGb: 12.4 },
          { zone: 'ZONE2', observer: 'obs-log-zone2-01', maxCpu: 6, usedCpu: 2.0, maxMemGb: 32, usedMemGb: 11.8 },
          { zone: 'ZONE3', observer: 'obs-log-zone3-01', maxCpu: 6, usedCpu: 2.1, maxMemGb: 32, usedMemGb: 11.8 },
        ],
        databases: [
          { name: 'access_log', tables: 12, size: '2.9 TB', conn: 40, status: 'ok' },
          { name: 'audit_log', tables: 8, size: '260 GB', conn: 12, status: 'ok' },
        ],
        whitelist: ['10.41.0.%'], connHint: 'mysql -h10.41.1.11 -P2881 -ulog_rw@log_tenant -p***',
      },
      {
        id: 't-logsys', name: 'sys', kind: 'sys', mode: 'mysql', primaryZone: 'RANDOM',
        locality: 'F@ZONE1,F@ZONE2,F@ZONE3', unitNum: 1,
        maxCpu: 2, usedCpu: 0.4, maxMemGb: 16, usedMemGb: 4,
        storageUsed: '20 GB', storageTotal: '60 GB', status: 'ok',
        units: [
          { zone: 'ZONE1', observer: 'obs-log-zone1-01', maxCpu: 2, usedCpu: 0.4, maxMemGb: 16, usedMemGb: 4.1 },
          { zone: 'ZONE2', observer: 'obs-log-zone2-01', maxCpu: 2, usedCpu: 0.4, maxMemGb: 16, usedMemGb: 4.0 },
          { zone: 'ZONE3', observer: 'obs-log-zone3-01', maxCpu: 2, usedCpu: 0.4, maxMemGb: 16, usedMemGb: 4.0 },
        ],
        databases: [{ name: 'oceanbase', tables: 0, size: '18 GB', conn: 4, status: 'ok' }],
        whitelist: ['%'], connHint: 'mysql -h10.41.1.11 -P2881 -uroot@sys -p***',
      },
    ],
    params: [
      { name: 'memory_limit', value: '48G', range: '16G - 物理内存', desc: 'OBServer 内存上限', status: 'ok' },
      { name: 'datafile_max_size', value: '1T', range: '磁盘容量内', desc: '数据文件上限', status: 'ok' },
    ],
  },
];

export const INSTANCE_USERS = [
  { user: 'trade_rw@trade_tenant', host: '10.40.10.%', priv: 'SELECT, INSERT, UPDATE', lastLogin: '2026-08-16 15:42', status: 'ok' },
  { user: 'trade_ro@trade_tenant', host: '10.40.10.%', priv: 'SELECT', lastLogin: '2026-08-16 15:39', status: 'ok' },
  { user: 'root@sys', host: '%', priv: 'ALL PRIVILEGES', lastLogin: '2026-08-15 22:10', status: 'ok' },
  { user: 'app_rw', host: '10.20.%.%', priv: 'SELECT, INSERT, UPDATE', lastLogin: '2026-08-16 14:20', status: 'ok' },
  { user: 'report_etl', host: '10.21.0.%', priv: 'SELECT, COPY', lastLogin: '2026-08-14 03:00', status: 'warn' },
  { user: 'tmp_debug', host: '10.99.1.5', priv: 'SELECT', lastLogin: '2026-07-28 11:20', status: 'err' },
];

export const SESSIONS = [
  { id: 88231, user: 'trade_rw@trade_tenant', host: '10.40.10.21:40218', db: 'trade_order', cmd: 'Query', time: '28s', state: 'Sending data', lock: '行锁等待', status: 'err' },
  { id: 88190, user: 'trade_rw@trade_tenant', host: '10.40.10.22:40871', db: 'trade_order', cmd: 'Query', time: '12s', state: 'update', lock: '—', status: 'warn' },
  { id: 88102, user: 'report_etl', host: '10.21.0.8:38452', db: 'analytics', cmd: 'Query', time: '96s', state: 'Copying to tmp table', lock: '—', status: 'warn' },
  { id: 87955, user: 'app_ro', host: '10.20.5.10:41002', db: 'user_center', cmd: 'Sleep', time: '300s', state: '—', lock: '—', status: 'ok' },
  { id: 87901, user: 'pay_rw@pay_tenant', host: '10.40.20.9:40233', db: 'payment', cmd: 'Query', time: '3s', state: 'executing', lock: '—', status: 'ok' },
];

export const TRANSACTIONS = [
  { id: 'TRX-998231', session: 88231, user: 'trade_rw@trade_tenant', dur: '1m 28s', undo: '14.2 MB', lockRows: '38,201', waiting: '是', sql: 'UPDATE stock_record SET qty = qty - ? …', status: 'err' },
  { id: 'TRX-998190', session: 88190, user: 'trade_rw@trade_tenant', dur: '42s', undo: '6.8 MB', lockRows: '12,044', waiting: '否', sql: 'INSERT INTO trade_order (…) VALUES (…)', status: 'warn' },
  { id: 'TRX-998102', session: 88102, user: 'report_etl', dur: '96s', undo: '0.2 MB', lockRows: '0', waiting: '否', sql: 'SELECT COUNT(*) FROM access_log …', status: 'ok' },
];

export const REPORTS = [
  { ico: '📊', title: '性能周报 · 第 33 周', desc: 'PG/OB 集群负载、TOP SQL、容量趋势', date: '2026-08-17 08:00', size: '2.4 MB' },
  { ico: '🔍', title: '慢 SQL 专项治理报告', desc: '本周新增 23 条慢 SQL，已优化 15 条', date: '2026-08-16 18:00', size: '1.1 MB' },
  { ico: '📈', title: 'OB 租户容量预测（8月）', desc: 'trade_tenant 存储 47 天后触达 85% 水位', date: '2026-08-15 08:00', size: '3.2 MB' },
  { ico: '🛡️', title: '巡检报告 · 每日', desc: '参数基线、备份校验、账号审计', date: '2026-08-17 06:00', size: '860 KB' },
];

export interface HostRow {
  ip: string; zone: string; spec: string; os: string;
  cpu: number; mem: number; disk: number; diskTotal: number;
  insts: string[]; cluster: string; cid: string; status: string;
}

export const HOSTS: HostRow[] = [
  { ip: '10.20.2.11', zone: '华东-AZ-B', spec: '16C / 64G', os: 'CentOS 7.9', cpu: 45, mem: 62, disk: 58, diskTotal: 1800, insts: ['pg-order-01'], cluster: 'prod-pg-order-01', cid: 'c1', status: 'ok' },
  { ip: '10.20.2.12', zone: '华东-AZ-B', spec: '16C / 64G', os: 'CentOS 7.9', cpu: 78, mem: 81, disk: 72, diskTotal: 1800, insts: ['pg-order-02'], cluster: 'prod-pg-order-01', cid: 'c1', status: 'warn' },
  { ip: '10.20.2.13', zone: '华东-AZ-B', spec: '16C / 64G', os: 'CentOS 7.9', cpu: 33, mem: 58, disk: 55, diskTotal: 1800, insts: ['pg-order-03'], cluster: 'prod-pg-order-01', cid: 'c1', status: 'ok' },
  { ip: '10.40.1.11', zone: '华东-AZ-A', spec: '16C / 96G', os: 'CentOS 7.9', cpu: 71, mem: 74, disk: 66, diskTotal: 2000, insts: ['observer-zone1-01'], cluster: 'prod-ob-core-01', cid: 'c3', status: 'ok' },
  { ip: '10.40.2.11', zone: '华东-AZ-B', spec: '16C / 96G', os: 'CentOS 7.9', cpu: 82, mem: 85, disk: 69, diskTotal: 2000, insts: ['observer-zone2-01'], cluster: 'prod-ob-core-01', cid: 'c3', status: 'warn' },
  { ip: '10.40.1.12', zone: '华东-AZ-A', spec: '16C / 96G', os: 'CentOS 7.9', cpu: 55, mem: 66, disk: 61, diskTotal: 2000, insts: ['observer-zone1-02'], cluster: 'prod-ob-core-01', cid: 'c3', status: 'ok' },
  { ip: '10.41.1.11', zone: '华东-AZ-A', spec: '8C / 48G', os: 'Ubuntu 22.04', cpu: 36, mem: 49, disk: 84, diskTotal: 1200, insts: ['obs-log-zone1-01'], cluster: 'prod-ob-log-01', cid: 'c4', status: 'ok' },
  { ip: '10.20.2.21', zone: '华东-AZ-B', spec: '16C / 64G', os: 'CentOS 7.9', cpu: 35, mem: 52, disk: 52, diskTotal: 1800, insts: ['pg-report-01'], cluster: 'prod-pg-report-02', cid: 'c2', status: 'ok' },
];

export const ALERT_INSTANCES = [
  { name: 'trade_tenant @ prod-ob-core-01', severity: 'P1', title: '租户 CPU 13.1/14C（阈值 90%）', time: '14:32', count: 6 },
  { name: 'pg-order-01（主库）', severity: 'P2', title: '备库复制延迟 850ms（阈值 300ms）', time: '14:18', count: 3 },
  { name: 'pay_tenant @ prod-ob-core-01', severity: 'P2', title: '租户内存水位 91%', time: '13:55', count: 2 },
  { name: 'analytics @ prod-pg-order-01', severity: 'P3', title: '连接数 410/500 · 慢查询堆积', time: '12:40', count: 1 },
  { name: 'observer-zone2-01', severity: 'P3', title: 'OBServer CPU 82% · 建议检查 Unit 均衡', time: '11:02', count: 1 },
];

/* 预置指标库（集群级）——MockProvider 数据源 */
export const METRIC_LIB: MetricDef[] = [
  { id: 'qps', name: 'QPS', unit: '', base: 210, jitter: 60 },
  { id: 'cpu', name: 'CPU 使用率', unit: '%', base: 55, jitter: 14 },
  { id: 'mem', name: '内存使用率', unit: '%', base: 68, jitter: 10 },
  { id: 'sessions', name: '活跃会话数', unit: '', base: 120, jitter: 40 },
  { id: 'slow_sql', name: '慢 SQL 趋势', unit: '', base: 18, jitter: 8 },
  { id: 'lock_wait', name: '锁等待会话', unit: '', base: 6, jitter: 4 },
  { id: 'disk', name: '磁盘使用率', unit: '%', base: 66, jitter: 3 },
  { id: 'repl_delay', name: '复制/同步延迟', unit: 's', base: 2, jitter: 3 },
];

export const LINE_COLORS = ['#006aff', '#00b365', '#ff9500', '#7a5af8', '#00a3e0', '#f53f3f'];

export const STATUS_MAP: Record<string, [string, string]> = {
  ok: ['ok', '正常'], warn: ['warn', '警告'], err: ['err', '异常'], info: ['info', '提示'],
};
export const TYPE_ICON: Record<string, string> = { pg: '🐘', oceanbase: '🌊' };
export const EXTRA_TYPE_NAME: Record<string, string> = { oceanbase: 'OceanBase' };
