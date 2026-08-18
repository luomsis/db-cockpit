/* ================= ECharts 选项构造（与原型一致） ================= */
import * as echarts from 'echarts';
import type { Annotation, PanelStyle, ResolvedTarget, ThresholdStep } from './types';

export const axisStyle = {
  axisLine: { lineStyle: { color: '#d9dee8' } },
  axisLabel: { color: '#8a97ad', fontSize: 11 },
  splitLine: { lineStyle: { color: '#eef1f8' } },
};
export const TIP = { backgroundColor: '#fff', borderColor: '#e5e9f2', textStyle: { color: '#1d2b45', fontSize: 12 }, extraCssText: 'box-shadow:0 4px 16px rgba(29,43,69,0.12)' };

export const DEFAULT_PANEL_STYLE: PanelStyle = {
  lineWidth: 2, fill: 0, points: 0, connectNulls: true,
  stack: 'none', legend: 'top', yMin: null, yMax: null, decimals: null, unit: '',
};

function axisFmt(unit: string, decimals: number | null) {
  return (val: any) => {
    const num = decimals != null ? Number(val).toFixed(decimals) : val;
    return `${num}${unit || ''}`;
  };
}

export function lineOpt(title: string, xs: string[], series: { name: string; data: number[]; color: string }[], unit?: string) {
  return {
    tooltip: { trigger: 'axis', ...TIP },
    grid: { left: 42, right: 14, top: 30, bottom: 24 },
    title: { text: title, textStyle: { color: '#4e5d78', fontSize: 12, fontWeight: 500 }, left: 4, top: 2 },
    xAxis: { type: 'category', data: xs, boundaryGap: false, ...axisStyle },
    yAxis: { type: 'value', ...axisStyle, axisLabel: { ...axisStyle.axisLabel, formatter: `{value}${unit || ''}` } },
    series: series.map(s => ({
      type: 'line', smooth: true, symbol: 'none' as const, data: s.data, name: s.name,
      lineStyle: { width: 2, color: s.color },
      areaStyle: { color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
        { offset: 0, color: s.color + '44' }, { offset: 1, color: s.color + '05' }]) },
    })),
  };
}

/* Grafana 风格时序图：多指标共享时间轴 + 双 Y 轴 + 序列级样式覆盖 + 面板级样式 */
export function tsChartOpt(title: string, xs: string[], targets: ResolvedTarget[], cfg: {
  legend?: boolean; thresholds?: ThresholdStep[]; annotations?: Annotation[]; style?: Partial<PanelStyle>;
}) {
  const st = { ...DEFAULT_PANEL_STYLE, ...(cfg.style || {}) };
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
    const s: any = {
      name: t.name, data: t.data, yAxisIndex: t.axis === 'right' ? 1 : 0,
      type: 'line', smooth: true, symbol: 'none', connectNulls: st.connectNulls,
      lineStyle: { width: lw, color: t.color }, itemStyle: { color: t.color },
    };
    if (t.dashed) s.lineStyle = { ...s.lineStyle, type: 'dashed' };
    if (t.type === 'bar') {
      s.type = 'bar'; s.barWidth = 10; s.smooth = undefined; s.lineStyle = undefined;
      s.itemStyle = { color: t.color, borderRadius: [3, 3, 0, 0] };
    } else if (t.type === 'points') {
      s.symbol = 'circle'; s.symbolSize = st.points > 0 ? st.points : 5;
    }
    if (st.fill > 0 && t.type !== 'bar') {
      const op = Math.round((st.fill / 100) * 255).toString(16).padStart(2, '0');
      s.areaStyle = { color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
        { offset: 0, color: t.color + op }, { offset: 1, color: t.color + '05' }]) };
    } else if (t.type === 'area') {
      s.areaStyle = { color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
        { offset: 0, color: t.color + '44' }, { offset: 1, color: t.color + '05' }]) };
    }
    if (st.points > 0 && t.type !== 'bar' && t.type !== 'points') {
      s.symbol = 'circle'; s.symbolSize = st.points;
    }
    if (isStack) s.stack = isPercent ? 'percent' : 'total';
    return s;
  });

  const yAxisBase: any = {
    type: 'value', ...axisStyle,
    axisLabel: { ...axisStyle.axisLabel, formatter: axisFmt(leftUnit, st.decimals) },
  };
  if (st.yMin != null && st.yMin !== (null as any)) yAxisBase.min = +st.yMin;
  if (st.yMax != null && st.yMax !== (null as any)) yAxisBase.max = +st.yMax;
  if (isPercent) yAxisBase.max = 100;
  const yAxes: any[] = [yAxisBase];
  if (hasRight) {
    const yr: any = {
      type: 'value', position: 'right', ...axisStyle, splitLine: { show: false },
      axisLabel: { ...axisStyle.axisLabel, formatter: axisFmt(rightUnit, st.decimals) },
    };
    if (isPercent) yr.max = 100;
    yAxes.push(yr);
  }

  const legPos = st.legend || 'top';
  const showLegend = cfg.legend !== false && legPos !== 'hide';
  const legCfg: any = { show: showLegend, textStyle: { color: '#4e5d78', fontSize: 11 }, itemWidth: 12, itemHeight: 8 };
  const grid: any = { left: 52, right: hasRight ? 52 : 14, top: 12, bottom: 24 };
  if (showLegend) {
    if (legPos === 'bottom') { legCfg.bottom = 0; grid.bottom = 36; }
    else if (legPos === 'right') { legCfg.right = 0; legCfg.top = 'center'; legCfg.orient = 'vertical'; grid.right = 100; }
    else { legCfg.top = 0; grid.top = 30; }
  }

  const option: any = {
    tooltip: { trigger: 'axis', ...TIP },
    legend: legCfg,
    grid,
    title: { text: title, textStyle: { color: '#4e5d78', fontSize: 12, fontWeight: 500 }, left: 4, top: 2 },
    xAxis: { type: 'category', data: xs, boundaryGap: false, ...axisStyle },
    yAxis: yAxes,
    series,
  };

  /* 阈值：Grafana steps 语义（升序分段着色）+ 阈值虚线
   * echarts 5.5 已知缺陷：visualMap 仅 2 个分段（单阈值）或数据完全未跨过
   * 第一个阈值时，折线渐变 stops 退化为空会抛 "reading 'coord'" ——
   * 这两种情况跳过 visualMap（阈值虚线仍绘制，视觉语义不受影响） */
  const mkData: any[] = [];
  const steps = (cfg.thresholds || []).slice().sort((a, b) => a.value - b.value);
  const maxVal = targets.length ? Math.max(...targets.flatMap(t => t.data)) : -Infinity;
  if (steps.length >= 2 && maxVal >= steps[0].value) {
    const pieces: any[] = [{ lt: steps[0].value, color: '#8a97ad' }];
    steps.forEach((s, i) => pieces.push({
      gte: s.value, lt: i + 1 < steps.length ? steps[i + 1].value : Infinity, color: s.color,
    }));
    option.visualMap = { show: false, dimension: 1, pieces };
  }
  steps.forEach(s => mkData.push({ yAxis: s.value, lineStyle: { color: s.color, width: 1, type: 'dashed' } }));

  /* 事件标注 */
  const annColors: Record<string, string> = { release: '#006aff', switch: '#ff9500', alert: '#f53f3f' };
  (cfg.annotations || []).forEach(a => {
    mkData.push({
      xAxis: a.time, title: a.title,
      lineStyle: { color: annColors[a.type] || '#8a97ad', width: 1.5 },
      label: { show: true, formatter: (p: any) => p.data.title, position: 'insideEndTop', color: annColors[a.type] || '#8a97ad', fontSize: 10 },
    });
  });
  if (mkData.length && series.length) series[0].markLine = { silent: true, symbol: 'none', data: mkData };
  return option;
}

export function gaugeOpt(title: string, val: number, unit?: string) {
  return {
    series: [{
      type: 'gauge', startAngle: 210, endAngle: -30, min: 0, max: 100,
      radius: '92%', center: ['50%', '58%'],
      progress: { show: true, width: 10, itemStyle: { color: new echarts.graphic.LinearGradient(0, 0, 1, 0, [{ offset: 0, color: '#006aff' }, { offset: 1, color: '#f53f3f' }]) } },
      axisLine: { lineStyle: { width: 10, color: [[1, '#eef1f8']] } },
      axisTick: { show: false }, splitLine: { show: false }, axisLabel: { show: false }, pointer: { show: false },
      detail: { valueAnimation: true, formatter: `{value}${unit || ''}`, color: '#1d2b45', fontSize: 20, fontWeight: 700, offsetCenter: [0, '12%'] },
      title: { show: true, offsetCenter: [0, '46%'], color: '#8a97ad', fontSize: 11 },
      data: [{ value: val, name: title }],
    }],
  };
}

export function lockGaugeOpt(value: number) {
  return {
    series: [{
      type: 'gauge', startAngle: 210, endAngle: -30, min: 0, max: 100,
      radius: '98%', center: ['50%', '58%'],
      progress: { show: true, width: 12, itemStyle: { color: new echarts.graphic.LinearGradient(0, 0, 1, 0, [{ offset: 0, color: '#006aff' }, { offset: 1, color: '#f53f3f' }]) } },
      axisLine: { lineStyle: { width: 12, color: [[1, '#eef1f8']] } },
      axisTick: { show: false }, splitLine: { show: false },
      axisLabel: { show: false }, pointer: { show: false },
      detail: { valueAnimation: true, formatter: '{value}%', color: '#ff9500', fontSize: 22, fontWeight: 700, offsetCenter: [0, '12%'] },
      title: { show: true, offsetCenter: [0, '48%'], color: '#8a97ad', fontSize: 11 },
      data: [{ value, name: '锁等待率' }],
    }],
  };
}
