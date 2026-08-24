/* 拓扑原型 mock 数据 —— 结构严格对齐元数据域 v2 表设计（架构文档 §6.1.1 v2）：
 * db_component（kind/group_name/traffic_upstream_id/replication_upstream_id/extensions）
 * db_host（region/az/host_cluster 三级位置）
 * db_cluster（endpoint = 客户端入口） */

export type TopoKind = 'proxy' | 'storage' | 'compute' | 'tenant' | 'access' | 'arbiter';

export interface TopoHost {
  host_ip: string;
  host_name: string;
  region: string;
  az: string;
  host_cluster: string;
}

export interface TopoComponent {
  id: string;
  name: string;
  kind: TopoKind;
  group_name?: string;            // 分组（shard-1 / z1 / config…）
  role?: string;                  // primary/secondary/observer/arbiter…
  version?: string;
  status?: 'ok' | 'warn' | 'err';
  host_ip?: string;               // → db_host（逻辑单元可空）
  port?: number;
  traffic_upstream_id?: string;   // 数据流上游（纵向 · 实线箭头）
  replication_upstream_id?: string; // 复制上游（横向 · 虚线箭头，备→主）
  extensions?: {
    delay_ms?: number;            // 复制延迟（箭头标签）
    sync?: string;                // 同步模式（quorum/异步/Paxos…）
    paxos?: boolean;              // 多主/Paxos：对称虚线渲染
    mode?: string;                // 租户兼容模式
    unit?: string;                // 租户规格
    max_cpu?: number;
    units?: { instance_id: string; zone?: string }[]; // 租户落位（N:M → observer）
  };
}

export interface TopoScenario {
  key: string;
  dbType: string;
  name: string;
  desc: string;
  endpoint: string;               // db_cluster.endpoint → 客户端虚节点
  layerOrder: TopoKind[];         // 纵向层序（client 之后自上而下）
  hosts: TopoHost[];
  components: TopoComponent[];
  hasTenantView?: boolean;        // OB：租户视图
}

/* ---------- ① PG：1 主 2 备流复制（无 proxy，VIP 直连） ---------- */
const pg: TopoScenario = {
  key: 'pg', dbType: 'PostgreSQL', name: 'PG 主从（Patroni）',
  desc: '纵向：存储层直连（VIP 入口）；横向：两备复制自主库（quorum）',
  endpoint: '10.20.2.10:5432',
  layerOrder: ['storage'],
  hosts: [
    { host_ip: '10.20.2.11', host_name: 'pg-order-01', region: '华东', az: 'az-b', host_cluster: 'hc-1' },
    { host_ip: '10.20.2.12', host_name: 'pg-order-02', region: '华东', az: 'az-c', host_cluster: 'hc-2' },
    { host_ip: '10.20.2.13', host_name: 'pg-order-03', region: '华东', az: 'az-c', host_cluster: 'hc-2' },
  ],
  components: [
    { id: 'pg-1', name: 'pg-order-01', kind: 'storage', role: 'primary', version: 'PG 15.6', status: 'ok', host_ip: '10.20.2.11', port: 5432 },
    { id: 'pg-2', name: 'pg-order-02', kind: 'storage', role: 'secondary', version: 'PG 15.6', status: 'warn', host_ip: '10.20.2.12', port: 5432, replication_upstream_id: 'pg-1', extensions: { delay_ms: 850, sync: 'quorum' } },
    { id: 'pg-3', name: 'pg-order-03', kind: 'storage', role: 'secondary', version: 'PG 15.6', status: 'ok', host_ip: '10.20.2.13', port: 5432, replication_upstream_id: 'pg-1', extensions: { delay_ms: 120, sync: 'quorum' } },
  ],
};

/* ---------- ② MySQL + Proxy：存储成员级流量指向 proxy ---------- */
const mysql: TopoScenario = {
  key: 'mysql', dbType: 'MySQL', name: 'MySQL + 代理层',
  desc: '纵向：proxy→存储（成员级流量精化）；横向：两备复制',
  endpoint: '10.30.1.100:3306',
  layerOrder: ['proxy', 'storage'],
  hosts: [
    { host_ip: '10.30.1.11', host_name: 'haproxy-1', region: '华东', az: 'az-a', host_cluster: 'hc-1' },
    { host_ip: '10.30.1.12', host_name: 'haproxy-2', region: '华东', az: 'az-b', host_cluster: 'hc-2' },
    { host_ip: '10.30.2.21', host_name: 'mysql-1', region: '华东', az: 'az-a', host_cluster: 'hc-1' },
    { host_ip: '10.30.2.22', host_name: 'mysql-2', region: '华东', az: 'az-b', host_cluster: 'hc-2' },
    { host_ip: '10.30.2.23', host_name: 'mysql-3', region: '华东', az: 'az-c', host_cluster: 'hc-3' },
  ],
  components: [
    { id: 'pxy-1', name: 'haproxy-1', kind: 'proxy', role: 'active', version: '2.8', status: 'ok', host_ip: '10.30.1.11', port: 3306 },
    { id: 'pxy-2', name: 'haproxy-2', kind: 'proxy', role: 'backup', version: '2.8', status: 'ok', host_ip: '10.30.1.12', port: 3306 },
    { id: 'my-1', name: 'mysql-1', kind: 'storage', role: 'primary', version: '8.0.36', status: 'ok', host_ip: '10.30.2.21', port: 3306, traffic_upstream_id: 'pxy-1' },
    { id: 'my-2', name: 'mysql-2', kind: 'storage', role: 'secondary', version: '8.0.36', status: 'ok', host_ip: '10.30.2.22', port: 3306, traffic_upstream_id: 'pxy-2', replication_upstream_id: 'my-1', extensions: { delay_ms: 60, sync: '半同步' } },
    { id: 'my-3', name: 'mysql-3', kind: 'storage', role: 'secondary', version: '8.0.36', status: 'ok', host_ip: '10.30.2.23', port: 3306, traffic_upstream_id: 'pxy-2', replication_upstream_id: 'my-1', extensions: { delay_ms: 210, sync: '异步' } },
  ],
};

/* ---------- ③ MongoDB 分片：mongos / shard-N / config 分组 ---------- */
const mongo: TopoScenario = {
  key: 'mongo', dbType: 'MongoDB', name: 'MongoDB 分片集群',
  desc: '分组（group_name）表达分片与 config；shard 成员流量指向 mongos；组内主备复制',
  endpoint: '10.40.1.100:27017',
  layerOrder: ['proxy', 'storage'],
  hosts: [
    { host_ip: '10.40.1.11', host_name: 'mongos-1', region: '华东', az: 'az-a', host_cluster: 'hc-1' },
    { host_ip: '10.40.1.12', host_name: 'mongos-2', region: '华东', az: 'az-b', host_cluster: 'hc-2' },
    { host_ip: '10.40.2.21', host_name: 'sh1-n1', region: '华东', az: 'az-a', host_cluster: 'hc-1' },
    { host_ip: '10.40.2.22', host_name: 'sh1-n2', region: '华东', az: 'az-b', host_cluster: 'hc-2' },
    { host_ip: '10.40.2.31', host_name: 'sh2-n1', region: '华东', az: 'az-b', host_cluster: 'hc-2' },
    { host_ip: '10.40.2.32', host_name: 'sh2-n2', region: '华东', az: 'az-c', host_cluster: 'hc-3' },
    { host_ip: '10.40.3.41', host_name: 'cfg-1', region: '华东', az: 'az-a', host_cluster: 'hc-1' },
    { host_ip: '10.40.3.42', host_name: 'cfg-2', region: '华东', az: 'az-b', host_cluster: 'hc-2' },
  ],
  components: [
    { id: 'mgs-1', name: 'mongos-1', kind: 'proxy', role: 'router', version: '7.0', status: 'ok', host_ip: '10.40.1.11', port: 27017 },
    { id: 'mgs-2', name: 'mongos-2', kind: 'proxy', role: 'router', version: '7.0', status: 'ok', host_ip: '10.40.1.12', port: 27017 },
    { id: 'sh1-1', name: 'shard1-n1', kind: 'storage', group_name: 'shard-1', role: 'primary', version: '7.0', status: 'ok', host_ip: '10.40.2.21', port: 27018, traffic_upstream_id: 'mgs-1' },
    { id: 'sh1-2', name: 'shard1-n2', kind: 'storage', group_name: 'shard-1', role: 'secondary', version: '7.0', status: 'ok', host_ip: '10.40.2.22', port: 27018, traffic_upstream_id: 'mgs-1', replication_upstream_id: 'sh1-1', extensions: { delay_ms: 40, sync: '同步' } },
    { id: 'sh2-1', name: 'shard2-n1', kind: 'storage', group_name: 'shard-2', role: 'primary', version: '7.0', status: 'ok', host_ip: '10.40.2.31', port: 27018, traffic_upstream_id: 'mgs-2' },
    { id: 'sh2-2', name: 'shard2-n2', kind: 'storage', group_name: 'shard-2', role: 'secondary', version: '7.0', status: 'warn', host_ip: '10.40.2.32', port: 27018, traffic_upstream_id: 'mgs-2', replication_upstream_id: 'sh2-1', extensions: { delay_ms: 1300, sync: '异步' } },
    { id: 'cfg-1', name: 'config-1', kind: 'arbiter', group_name: 'config', role: 'primary', version: '7.0', status: 'ok', host_ip: '10.40.3.41', port: 27019 },
    { id: 'cfg-2', name: 'config-2', kind: 'arbiter', group_name: 'config', role: 'secondary', version: '7.0', status: 'ok', host_ip: '10.40.3.42', port: 27019, replication_upstream_id: 'cfg-1', extensions: { delay_ms: 15, sync: '同步' } },
  ],
};

/* ---------- ④ Redis Cluster：3 分组主备 ---------- */
const redis: TopoScenario = {
  key: 'redis', dbType: 'Redis', name: 'Redis Cluster',
  desc: '去中心化：无 proxy 层，客户端直连（traffic 置空）；分组=槽分片',
  endpoint: '10.50.1.100:6379',
  layerOrder: ['storage'],
  hosts: [
    { host_ip: '10.50.1.11', host_name: 'redis-11', region: '华北', az: 'az-a', host_cluster: 'hc-1' },
    { host_ip: '10.50.1.12', host_name: 'redis-12', region: '华北', az: 'az-b', host_cluster: 'hc-2' },
    { host_ip: '10.50.1.21', host_name: 'redis-21', region: '华北', az: 'az-b', host_cluster: 'hc-2' },
    { host_ip: '10.50.1.22', host_name: 'redis-22', region: '华北', az: 'az-c', host_cluster: 'hc-3' },
    { host_ip: '10.50.1.31', host_name: 'redis-31', region: '华北', az: 'az-c', host_cluster: 'hc-3' },
    { host_ip: '10.50.1.32', host_name: 'redis-32', region: '华北', az: 'az-a', host_cluster: 'hc-1' },
  ],
  components: [
    { id: 'rd-11', name: 'redis-11', kind: 'storage', group_name: 'shard-1 (0-5460)', role: 'master', version: '7.2', status: 'ok', host_ip: '10.50.1.11', port: 6379 },
    { id: 'rd-12', name: 'redis-12', kind: 'storage', group_name: 'shard-1 (0-5460)', role: 'replica', version: '7.2', status: 'ok', host_ip: '10.50.1.12', port: 6379, replication_upstream_id: 'rd-11', extensions: { delay_ms: 8, sync: '异步' } },
    { id: 'rd-21', name: 'redis-21', kind: 'storage', group_name: 'shard-2 (5461-10922)', role: 'master', version: '7.2', status: 'ok', host_ip: '10.50.1.21', port: 6379 },
    { id: 'rd-22', name: 'redis-22', kind: 'storage', group_name: 'shard-2 (5461-10922)', role: 'replica', version: '7.2', status: 'ok', host_ip: '10.50.1.22', port: 6379, replication_upstream_id: 'rd-21', extensions: { delay_ms: 12, sync: '异步' } },
    { id: 'rd-31', name: 'redis-31', kind: 'storage', group_name: 'shard-3 (10923-16383)', role: 'master', version: '7.2', status: 'warn', host_ip: '10.50.1.31', port: 6379 },
    { id: 'rd-32', name: 'redis-32', kind: 'storage', group_name: 'shard-3 (10923-16383)', role: 'replica', version: '7.2', status: 'ok', host_ip: '10.50.1.32', port: 6379, replication_upstream_id: 'rd-31', extensions: { delay_ms: 95, sync: '异步' } },
  ],
};

/* ---------- ⑤ OceanBase：obproxy + 租户（units 落位） + observer 三 Zone Paxos ---------- */
const ob: TopoScenario = {
  key: 'ob', dbType: 'OceanBase', name: 'OB 集群（含租户视图）',
  desc: '纵向：obproxy→租户→observer（units 落位）；横向：observer 间 Paxos（多主，复制字段置空）',
  endpoint: '10.60.1.100:2883',
  layerOrder: ['proxy', 'tenant', 'storage'],
  hasTenantView: true,
  hosts: [
    { host_ip: '10.60.1.11', host_name: 'obproxy-1', region: '华东', az: 'az-a', host_cluster: 'hc-1' },
    { host_ip: '10.60.1.12', host_name: 'obproxy-2', region: '华东', az: 'az-b', host_cluster: 'hc-2' },
    { host_ip: '10.60.2.11', host_name: 'obs-z1-1', region: '华东', az: 'az-a', host_cluster: 'hc-1' },
    { host_ip: '10.60.2.12', host_name: 'obs-z1-2', region: '华东', az: 'az-a', host_cluster: 'hc-1' },
    { host_ip: '10.60.3.11', host_name: 'obs-z2-1', region: '华东', az: 'az-b', host_cluster: 'hc-2' },
    { host_ip: '10.60.3.12', host_name: 'obs-z2-2', region: '华东', az: 'az-b', host_cluster: 'hc-2' },
    { host_ip: '10.60.4.11', host_name: 'obs-z3-1', region: '华东', az: 'az-c', host_cluster: 'hc-3' },
    { host_ip: '10.60.4.12', host_name: 'obs-z3-2', region: '华东', az: 'az-c', host_cluster: 'hc-3' },
  ],
  components: [
    { id: 'obp-1', name: 'obproxy-1', kind: 'proxy', role: 'active', version: '4.2', status: 'ok', host_ip: '10.60.1.11', port: 2883 },
    { id: 'obp-2', name: 'obproxy-2', kind: 'proxy', role: 'active', version: '4.2', status: 'ok', host_ip: '10.60.1.12', port: 2883 },
    { id: 'tn-trade', name: 'trade_tenant', kind: 'tenant', role: 'user', status: 'warn', port: 2881, traffic_upstream_id: 'obp-1',
      extensions: { mode: 'MySQL', unit: '14C / 72G × 3', max_cpu: 14, units: [{ instance_id: 'obs-z1-1', zone: 'z1' }, { instance_id: 'obs-z2-1', zone: 'z2' }, { instance_id: 'obs-z3-1', zone: 'z3' }] } },
    { id: 'tn-pay', name: 'pay_tenant', kind: 'tenant', role: 'user', status: 'ok', port: 2881, traffic_upstream_id: 'obp-2',
      extensions: { mode: 'MySQL', unit: '12C / 48G × 3', max_cpu: 12, units: [{ instance_id: 'obs-z1-2', zone: 'z1' }, { instance_id: 'obs-z2-2', zone: 'z2' }, { instance_id: 'obs-z3-2', zone: 'z3' }] } },
    { id: 'obs-z1-1', name: 'observer-z1-1', kind: 'storage', group_name: 'ZONE1', role: 'observer', version: 'OB 4.2.1', status: 'ok', host_ip: '10.60.2.11', port: 2881, extensions: { paxos: true } },
    { id: 'obs-z1-2', name: 'observer-z1-2', kind: 'storage', group_name: 'ZONE1', role: 'observer', version: 'OB 4.2.1', status: 'ok', host_ip: '10.60.2.12', port: 2881, extensions: { paxos: true } },
    { id: 'obs-z2-1', name: 'observer-z2-1', kind: 'storage', group_name: 'ZONE2', role: 'observer', version: 'OB 4.2.1', status: 'warn', host_ip: '10.60.3.11', port: 2881, extensions: { paxos: true } },
    { id: 'obs-z2-2', name: 'observer-z2-2', kind: 'storage', group_name: 'ZONE2', role: 'observer', version: 'OB 4.2.1', status: 'ok', host_ip: '10.60.3.12', port: 2881, extensions: { paxos: true } },
    { id: 'obs-z3-1', name: 'observer-z3-1', kind: 'storage', group_name: 'ZONE3', role: 'observer', version: 'OB 4.2.1', status: 'ok', host_ip: '10.60.4.11', port: 2881, extensions: { paxos: true } },
    { id: 'obs-z3-2', name: 'observer-z3-2', kind: 'storage', group_name: 'ZONE3', role: 'observer', version: 'OB 4.2.1', status: 'ok', host_ip: '10.60.4.12', port: 2881, extensions: { paxos: true } },
  ],
};

export const TOPO_SCENARIOS: TopoScenario[] = [pg, mysql, mongo, redis, ob];
