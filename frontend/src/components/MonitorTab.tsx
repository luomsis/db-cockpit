import { useMemo } from 'react';
import { Chart } from './Chart';
import { HOURS, genSeries } from '../lib/query';
import { lineOpt } from '../lib/chartOptions';

/* 通用性能监控 2×2 图表 */
export function MonitorTab() {
  const opts = useMemo(() => ([
    lineOpt('CPU 使用率（%）', HOURS, [{ name: 'CPU', data: genSeries(55, 14, 14), color: '#006aff' }], '%'),
    lineOpt('内存使用率（%）', HOURS, [{ name: '内存', data: genSeries(68, 8), color: '#7a5af8' }], '%'),
    lineOpt('QPS / TPS', HOURS, [
      { name: 'QPS', data: genSeries(18000, 5000, 14), color: '#00a3e0' },
      { name: 'TPS', data: genSeries(4200, 1200, 14), color: '#00b365' }]),
    lineOpt('活跃会话数', HOURS, [{ name: '会话', data: genSeries(120, 40, 15), color: '#ff9500' }]),
  ]), []);
  return (
    <div className="mon-grid">
      {opts.map((o, i) => <div className="card" key={i}><Chart option={o} className="chart-box lg" /></div>)}
    </div>
  );
}
