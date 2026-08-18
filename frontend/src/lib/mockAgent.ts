/* ================= Mock Agent：模拟 SSE 流式回复（对齐执行框架 §9 事件协议） =================
 * 后端 agentcluster 就绪后，本模块整体替换为真实 SSE 客户端，页面与卡片渲染零改动。
 */
import type { AgentEvent, CardEnvelope } from './types';
import { ALERT_INSTANCES, SESSIONS, SLOW_SQLS } from './mockData';
import { HOURS, genSeries } from './query';

let cardSeq = 0;
function cid() { return `card_${Date.now().toString(36)}_${(cardSeq++).toString(36)}`; }
function sleep(ms: number) { return new Promise(r => setTimeout(r, ms)); }

type Emit = (ev: AgentEvent) => void;

function metricChartCard(title: string, metric: string, base: number, jitter: number, unit: string, thresholds?: any[]): CardEnvelope {
  const data = genSeries(base, jitter, 14, 20260816);
  return {
    card_id: cid(), card_type: 'metric_chart', protocol_version: '1.0',
    title, status: 'final',
    source: { agent: 'diagnosis_expert', tool_call_id: null },
    context: null,
    payload: {
      chart_type: 'line',
      metrics: [{ name: metric, label: title, unit }],
      data: { points: HOURS.map((h, i) => [h, data[i]]) },
      thresholds,
    },
    fallback_text: `${title}：近 24h 均值 ${Math.round(data.reduce((a, b) => a + b, 0) / data.length)}${unit}`,
  };
}

function sessionsTableCard(): CardEnvelope {
  return {
    card_id: cid(), card_type: 'data_table', protocol_version: '1.0',
    title: '租户会话快照（trade_tenant @ prod-ob-core-01）', status: 'final',
    source: { agent: 'diagnosis_expert', tool_call_id: null },
    payload: {
      columns: [
        { key: 'id', label: '会话 ID', type: 'number' },
        { key: 'user', label: '用户' },
        { key: 'db', label: '库' },
        { key: 'time', label: '时长', type: 'duration' },
        { key: 'state', label: '状态' },
        { key: 'lock', label: '锁信息' },
      ],
      rows: SESSIONS.map(s => ({ id: s.id, user: s.user, db: s.db, time: s.time, state: s.state, lock: s.lock })),
      total: SESSIONS.length,
      row_actions: ['ask'],
    },
    fallback_text: `会话快照：共 ${SESSIONS.length} 个会话，2 个异常（行锁等待 / 长事务）`,
  };
}

function alertsTableCard(): CardEnvelope {
  return {
    card_id: cid(), card_type: 'data_table', protocol_version: '1.0',
    title: '当前告警实例（近 24h）', status: 'final',
    source: { agent: 'router', tool_call_id: null },
    payload: {
      columns: [
        { key: 'name', label: '实例', type: 'string' },
        { key: 'severity', label: '级别', type: 'status' },
        { key: 'title', label: '告警内容' },
        { key: 'time', label: '首次触发', type: 'time' },
        { key: 'count', label: '次数', type: 'number' },
      ],
      rows: ALERT_INSTANCES.map(a => ({ ...a })),
      total: ALERT_INSTANCES.length,
      row_actions: ['ask'],
    },
    fallback_text: `当前 ${ALERT_INSTANCES.length} 个实例处于告警状态，最高 P1`,
  };
}

function taskProgressCard(taskId: string): CardEnvelope {
  return {
    card_id: cid(), card_type: 'task_progress', protocol_version: '1.0',
    title: '深度诊断任务', status: 'streaming',
    source: { agent: 'task_bus' },
    payload: {
      task_id: taskId, task_type: 'builtin_metric_deep_scan', status: 'pending',
      stages: [
        { name: '多指标长时窗扫描', status: 'pending' },
        { name: '异常区间关联分析', status: 'pending' },
        { name: '根因假设生成', status: 'pending' },
      ],
      cancellable: true,
    },
    fallback_text: '深度诊断任务已提交',
  };
}

function diagnosisReportCard(taskId: string): CardEnvelope {
  return {
    card_id: cid(), card_type: 'diagnosis_report', protocol_version: '1.0',
    title: '租户 trade_tenant @ prod-ob-core-01 性能诊断报告', status: 'final',
    source: { agent: 'diagnosis_expert', tool_call_id: null },
    context: { instance_id: 'trade_tenant', cluster_id: 'prod-ob-core-01', time_range: { start: '2026-08-16T13:00:00+08:00', end: '2026-08-16T14:30:00+08:00' } },
    payload: {
      summary: 'CPU 飙升由全表扫描型慢 SQL 放大引发，锁等待为次生影响',
      severity: 'critical',
      root_causes: [
        { hypothesis: 'trade_order.status 无索引导致全表扫描，QPS 高峰放大 CPU', confidence: 0.92 },
        { hypothesis: '长事务 TRX-998231 持锁未提交引发会话堆积', confidence: 0.74 },
      ],
      findings: [
        { category: 'metric_anomaly', detail: 'CPU 使用率 14:00 后从 55% 升至 98%，与 QPS 激增同步' },
        { category: 'slow_sql', detail: 'TOP1 慢 SQL 扫描 438 万行，日均执行 342 次' },
        { category: 'lock', detail: '会话 88231 持有 stock_record 行锁 1m28s，阻塞 3 个会话' },
        { category: 'session', detail: '活跃会话 512（阈值 300），堆积持续 40 分钟' },
      ],
      suggestions: [
        '添加联合索引 idx_trade_order_status_uid (status, uid)，预计扫描行数下降 92%',
        '联系 app_rw 业务方确认事务边界，避免长事务持锁',
        '如影响持续扩大，可在会话管理中 Kill 会话 88231 止血',
      ],
      provider: 'builtin',
    },
    fallback_text: '诊断结论：全表扫描慢 SQL 引发 CPU 飙升，锁等待为次生影响。建议添加联合索引。',
  };
}

/* ---------- 场景脚本 ---------- */

async function scenarioDiagnosis(emit: Emit, signal: { cancelled: boolean }) {
  const guard = () => signal.cancelled;
  const say = async (text: string) => {
    emit({ type: 'token', text_delta: '' });
    for (let i = 0; i < text.length && !guard(); i += 3) {
      emit({ type: 'token', text_delta: text.slice(i, i + 3) });
      await sleep(24);
    }
  };
  const thought = async (tool: string, ms: number) => {
    if (guard()) return;
    emit({ type: 'thought', step: 0, tool_name: tool, status: 'running' });
    await sleep(ms);
    if (!guard()) emit({ type: 'thought', step: 0, tool_name: tool, status: 'success' });
  };

  await say('好的，我将对 prod-ob-core-01 的 trade_tenant 租户发起诊断，先采集指标与告警证据：\n\n');
  await thought('builtin_get_metrics', 800);
  if (guard()) return;
  emit({ type: 'card', mode: 'create', card: metricChartCard('CPU 使用率（近 24h）', 'cpu', 55, 14, '%', [{ value: 85, label: '告警阈值', severity: 'warning' }]) });
  await thought('builtin_list_alerts', 600);
  if (guard()) return;
  await say('指标显示 CPU 在 14:00 后异常飙升，继续采集会话快照与慢 SQL 证据：\n\n');
  await thought('builtin_session_snapshot', 900);
  if (guard()) return;
  emit({ type: 'card', mode: 'create', card: sessionsTableCard() });
  await sleep(300);

  const taskId = 'task_' + Date.now().toString(36);
  emit({ type: 'card', mode: 'create', card: taskProgressCard(taskId) });
  await say('\n\n初步证据已确认异常，已提交异步深度诊断任务（多指标长时窗扫描），完成后我将汇总交叉验证结论。');
  await sleep(400);

  const stages = ['多指标长时窗扫描', '异常区间关联分析', '根因假设生成'];
  const progressSeq = [[15, 0], [45, 0], [70, 1], [95, 2]] as [number, number][];
  for (const [p, si] of progressSeq) {
    if (guard()) return;
    emit({ type: 'progress', task_id: taskId, progress: p, stage: `${stages[si]}…` });
    await sleep(900);
  }
  if (guard()) return;
  emit({ type: 'progress', task_id: taskId, progress: 100, stage: '完成' });
  await sleep(500);
  if (guard()) return;
  emit({ type: 'card', mode: 'update', card: diagnosisReportCard(taskId) });
  await sleep(200);
  await say('\n\n诊断报告已生成：根因是 trade_order.status 缺索引引发全表扫描（置信度 92%），锁等待为次生影响。可点击卡片查看完整证据链，或直接追问。');
  emit({ type: 'done' });
}

async function scenarioAlerts(emit: Emit, signal: { cancelled: boolean }) {
  const guard = () => signal.cancelled;
  emit({ type: 'thought', step: 0, tool_name: 'builtin_list_alerts', status: 'running' });
  await sleep(700);
  if (guard()) return;
  emit({ type: 'thought', step: 0, tool_name: 'builtin_list_alerts', status: 'success' });
  emit({ type: 'card', mode: 'create', card: alertsTableCard() });
  const text = `当前共有 ${ALERT_INSTANCES.length} 个实例处于告警状态，其中 P1 级别 1 个：trade_tenant @ prod-ob-core-01（租户 CPU 13.1/14C，疑似全表扫描导致）。\n\n建议优先处理 P1，可直接对我说「诊断 trade_tenant」。`;
  for (let i = 0; i < text.length && !guard(); i += 3) {
    emit({ type: 'token', text_delta: text.slice(i, i + 3) });
    await sleep(24);
  }
  emit({ type: 'done' });
}

async function scenarioDataqa(emit: Emit, signal: { cancelled: boolean }) {
  const guard = () => signal.cancelled;
  emit({ type: 'thought', step: 0, tool_name: 'builtin_get_metrics', status: 'running' });
  await sleep(700);
  if (guard()) return;
  emit({ type: 'thought', step: 0, tool_name: 'builtin_get_metrics', status: 'success' });
  emit({ type: 'card', mode: 'create', card: metricChartCard('全局 QPS 趋势（近 24h）', 'qps', 18500, 4200, '') });
  const text = '近 24h 全局 QPS 均值约 18.5k，14:00 起出现约 35% 的流量高峰并伴随慢 SQL 增多。需要看集群维度对比或切时间窗，可以直接告诉我。';
  for (let i = 0; i < text.length && !guard(); i += 3) {
    emit({ type: 'token', text_delta: text.slice(i, i + 3) });
    await sleep(24);
  }
  emit({ type: 'done' });
}

async function scenarioGreeting(emit: Emit, signal: { cancelled: boolean }) {
  const text = '你好！我是 DB Cockpit 智能运维助手 🤖\n\n我可以帮你：\n• 诊断实例性能异常（指标 → 会话 → 慢 SQL → 深度扫描全链路）\n• 查询平台运维数据（QPS / 告警 / 慢 SQL 统计）\n• 分析锁等待与长事务根因\n\n试着问我：「当前有哪些告警实例？」或「诊断 trade_tenant」';
  for (let i = 0; i < text.length && !signal.cancelled; i += 3) {
    emit({ type: 'token', text_delta: text.slice(i, i + 3) });
    await sleep(20);
  }
  emit({ type: 'done' });
}

async function scenarioFallback(emit: Emit, signal: { cancelled: boolean }) {
  const text = `已收到问题。当前为前端演示模式（数据为 mock），我支持的演示场景：\n\n1. 「当前有哪些告警实例」— 告警问数\n2. 「诊断 trade_tenant」— 完整诊断链路（含异步深度扫描）\n3. 「QPS 趋势」— 指标问数\n\n正式版将接入 agent 集群（路由 / 诊断 / 问数专家），通过工具注册表调用真实数据。`;
  for (let i = 0; i < text.length && !signal.cancelled; i += 3) {
    emit({ type: 'token', text_delta: text.slice(i, i + 3) });
    await sleep(20);
  }
  emit({ type: 'done' });
}

function pickScenario(text: string) {
  if (/你好|hi|hello|帮助|你能/.test(text)) return scenarioGreeting;
  if (/告警/.test(text) && !/诊断/.test(text)) return scenarioAlerts;
  if (/诊断|变慢|排查|根因|为什么.*(慢|卡)|分析.*实例/.test(text)) return scenarioDiagnosis;
  if (/QPS|qps|指标|趋势|多少|统计|查询/.test(text)) return scenarioDataqa;
  return scenarioFallback;
}

export interface MockTurnHandle { cancel: () => void }

export function runMockTurn(text: string, emit: Emit): MockTurnHandle {
  const signal = { cancelled: false };
  pickScenario(text)(ev => { if (!signal.cancelled) emit(ev); }, signal).catch(() => {
    if (!signal.cancelled) emit({ type: 'error', code: 'mock_error', message: '模拟执行中断' });
  });
  return { cancel: () => { signal.cancelled = true; emit({ type: 'done' }); } };
}

export const QUICK_QUESTIONS = ['当前有哪些告警实例？', '诊断 prod-ob-core-01 的 trade_tenant', 'QPS 趋势怎么样？'];
export { SLOW_SQLS };
