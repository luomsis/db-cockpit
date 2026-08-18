import { useEffect, useMemo, useState } from 'react';
import { Chart } from '../components/Chart';
import { Stat, Pill } from '../components/bits';
import { DB_TYPES, TOP_ANOMALY, SQL_ISSUES, SLOW_SQLS } from '../lib/mockData';
import { apiGet, withFallback } from '../lib/api';
import { lineOpt, lockGaugeOpt, TIP, axisStyle } from '../lib/chartOptions';
import * as echarts from 'echarts';
import { useBreadcrumb } from '../App';

interface DbTypeRow { type: string; name: string; icon: string; total: number; alert: number }
interface AnomalyRow { name: string; cluster: string; score: number; issue: string; inst: string }
interface IssueRow { name: string; cnt: number }
interface SlowSqlRow { sql: string; db: string; time: string; rows: string; count: number }
interface LockSummary { lockWaitRate: number; lockWaitSessions: number; deadlockToday: number; longestWait: string; mdlBlocked: number; hotTable: string }

/* 概览数据：apiserver 优先，失败回退本地 mock（初始即 mock 值，无闪烁） */
const DEFAULT_LOCK: LockSummary = {
  lockWaitRate: 38, lockWaitSessions: 7, deadlockToday: 2,
  longestWait: '1m 28s · TRX-998231', mdlBlocked: 1, hotTable: 'stock_record',
};

export default function Overview() {
  useBreadcrumb([{ label: '首页' }, { label: '运维概览' }]);
  const [dbTypes, setDbTypes] = useState<DbTypeRow[]>(DB_TYPES);
  const [topAnomaly, setTopAnomaly] = useState<AnomalyRow[]>(TOP_ANOMALY);
  const [sqlIssues, setSqlIssues] = useState<IssueRow[]>(SQL_ISSUES);
  const [slowSqls, setSlowSqls] = useState<SlowSqlRow[]>(SLOW_SQLS);
  const [lock, setLock] = useState<LockSummary>(DEFAULT_LOCK);

  useEffect(() => {
    let alive = true;
    withFallback(
      apiGet<{ dbTypes: DbTypeRow[]; topAnomaly: AnomalyRow[]; sqlIssues: IssueRow[]; slowSqls: SlowSqlRow[]; lock: LockSummary }>('/api/overview'),
      () => null,
    ).then(d => {
      if (!alive || !d) return;
      if (d.dbTypes?.length) setDbTypes(d.dbTypes);
      if (d.topAnomaly?.length) setTopAnomaly(d.topAnomaly);
      if (d.sqlIssues?.length) setSqlIssues(d.sqlIssues);
      if (d.slowSqls?.length) setSlowSqls(d.slowSqls);
      if (d.lock) setLock(d.lock);
    });
    return () => { alive = false; };
  }, []);

  const totalInst = dbTypes.reduce((a, b) => a + b.total, 0);
  const totalAlert = dbTypes.reduce((a, b) => a + b.alert, 0);

  const dbTypeOpt = useMemo(() => ({
    tooltip: { trigger: 'axis', ...TIP },
    legend: { data: ['实例数', '告警数'], textStyle: { color: '#4e5d78', fontSize: 11 }, top: 0, right: 0, itemWidth: 12, itemHeight: 8 },
    grid: { left: 36, right: 10, top: 30, bottom: 24 },
    xAxis: { type: 'category', data: dbTypes.map(d => d.name), ...axisStyle },
    yAxis: { type: 'value', ...axisStyle },
    series: [
      { name: '实例数', type: 'bar', barWidth: 14, itemStyle: { borderRadius: [4, 4, 0, 0], color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [{ offset: 0, color: '#338bff' }, { offset: 1, color: '#006aff' }]) }, data: dbTypes.map(d => d.total) },
      { name: '告警数', type: 'bar', barWidth: 14, itemStyle: { borderRadius: [4, 4, 0, 0], color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [{ offset: 0, color: '#ffb056' }, { offset: 1, color: '#ff9500' }]) }, data: dbTypes.map(d => d.alert) },
    ],
  }), [dbTypes]);

  const sqlIssueOpt = useMemo(() => ({
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' }, ...TIP },
    grid: { left: 92, right: 30, top: 8, bottom: 8 },
    xAxis: { type: 'value', ...axisStyle, splitLine: { lineStyle: { color: '#eef1f8' } } },
    yAxis: { type: 'category', data: sqlIssues.map(s => s.name).reverse(), ...axisStyle, axisLabel: { color: '#4e5d78', fontSize: 11.5 } },
    series: [{
      type: 'bar', barWidth: 12, data: sqlIssues.map(s => s.cnt).reverse(),
      label: { show: true, position: 'right', color: '#006aff', fontSize: 11 },
      itemStyle: { borderRadius: [0, 4, 4, 0], color: new echarts.graphic.LinearGradient(1, 0, 0, 0, [{ offset: 0, color: '#00a3e0' }, { offset: 1, color: '#006aff' }]) },
    }],
  }), [sqlIssues]);

  return (
    <>
      <div className="page-title">运维概览 <span className="pill info"><i></i>实时 · 每 15s 刷新</span></div>
      <div className="page-desc">数据库智能驾驶仓 · 全局健康视图（数据截至 {new Date().toLocaleTimeString('zh-CN', { hour12: false })}）</div>
      <div className="ov-grid">
        <div className="card c1">
          <div className="card-head">
            <div className="card-title"><span className="t-ico"></span>数据库类型及告警概览</div>
            <a className="card-link" href="#/clusters">查看全部 ›</a>
          </div>
          <div className="stat-row">
            <Stat num={dbTypes.length} lbl="数据库类型" cls="ok" />
            <Stat num={totalAlert} lbl="告警实例数" cls="warn" />
            <Stat num={totalInst} lbl="总实例数" />
          </div>
          <Chart option={dbTypeOpt} className="chart-box" />
        </div>

        <div className="card c2">
          <div className="card-head">
            <div className="card-title"><span className="t-ico"></span>性能异常实例 TOP5</div>
            <span className="card-sub">健康分越低越危险</span>
          </div>
          <div className="rank-list">
            {topAnomaly.map((t, i) => (
              <div className="rank-item" key={t.name} onClick={() => { location.hash = '#/clusters'; }}>
                <span className={`rank-no ${i < 2 ? 'hot' : ''}`}>{i + 1}</span>
                <div className="rank-main">
                  <div className="rank-name">{t.name}</div>
                  <div className="rank-meta">{t.cluster} · {t.issue}</div>
                </div>
                <span className={`rank-score ${t.score >= 90 ? 'score-danger' : t.score >= 75 ? 'score-warn' : 'score-ok'}`}>{t.score}</span>
              </div>
            ))}
          </div>
        </div>

        <div className="card c3">
          <div className="card-head">
            <div className="card-title"><span className="t-ico"></span>SQL 性能问题概览</div>
            <span className="card-sub">近 24 小时</span>
          </div>
          <Chart option={sqlIssueOpt} className="chart-box lg" style={{ height: 252 }} />
        </div>

        <div className="card c4">
          <div className="card-head">
            <div className="card-title"><span className="t-ico"></span>锁分析</div>
            <span className="card-sub">全局锁态势</span>
          </div>
          <div className="lock-flex">
            <div className="lock-gauge"><Chart option={lockGaugeOpt(lock.lockWaitRate)} style={{ height: 170 }} /></div>
            <div className="lock-right">
              <div className="lock-kv"><span className="k">当前锁等待会话</span><span className="v" style={{ color: 'var(--red)' }}>{lock.lockWaitSessions}</span></div>
              <div className="lock-kv"><span className="k">今日死锁次数</span><span className="v" style={{ color: 'var(--amber)' }}>{lock.deadlockToday}</span></div>
              <div className="lock-kv"><span className="k">最长锁等待</span><span className="v">{lock.longestWait}</span></div>
              <div className="lock-kv"><span className="k">元数据锁（MDL）阻塞</span><span className="v">{lock.mdlBlocked}</span></div>
              <div className="lock-kv"><span className="k">热点争用表</span><span className="v">{lock.hotTable}</span></div>
            </div>
          </div>
        </div>

        <div className="card c5">
          <div className="card-head">
            <div className="card-title"><span className="t-ico"></span>慢 SQL TOP5</div>
            <a className="card-link" href="#/chat">诊断 ›</a>
          </div>
          <div className="sql-list">
            {slowSqls.map(s => (
              <div className="sql-item" key={s.sql}>
                <div className="sql-text">{s.sql}</div>
                <div className="sql-meta">
                  <span>库 <b>{s.db}</b></span>
                  <span>耗时 <b style={{ color: 'var(--amber)' }}>{s.time}</b></span>
                  <span>扫描 <b>{s.rows}</b> 行</span>
                  <span>次数 <b>{s.count}</b></span>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </>
  );
}
