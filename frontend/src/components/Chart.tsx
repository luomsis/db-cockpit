import { useEffect, useRef } from 'react';
import * as echarts from 'echarts';

/* ECharts 受控封装：init/dispose/resize 生命周期管理 */
export function Chart({ option, className, style, onReady }: {
  option: any;
  className?: string;
  style?: React.CSSProperties;
  onReady?: (chart: echarts.ECharts) => void;
}) {
  const ref = useRef<HTMLDivElement>(null);
  const chartRef = useRef<echarts.ECharts | null>(null);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    const c = echarts.init(el);
    chartRef.current = c;
    onReady?.(c);
    const ro = new ResizeObserver(() => c.resize());
    ro.observe(el);
    return () => { ro.disconnect(); c.dispose(); chartRef.current = null; };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    chartRef.current?.setOption(option, true);
  }, [option]);

  return <div ref={ref} className={className} style={style} />;
}
