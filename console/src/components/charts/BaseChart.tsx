/**
 * BaseChart - ECharts基础封装组件
 * 提供ECharts实例管理、响应式调整、主题支持
 */
import React, { useRef, useEffect, useMemo } from 'react';
import ReactECharts from 'echarts-for-react';
import type { EChartsOption } from 'echarts';
import { Spin } from 'antd';
import { theme } from 'antd';

const { useToken } = theme;

/**
 * BaseChart Props
 */
export interface BaseChartProps {
  /** ECharts配置项 */
  option: EChartsOption;
  /** 图表高度 */
  height?: string | number;
  /** 加载状态 */
  loading?: boolean;
  /** 事件处理器映射 */
  onEvents?: Record<string, (params: unknown) => void>;
  /** 图表样式 */
  style?: React.CSSProperties;
}

/**
 * BaseChart组件
 * @description ECharts基础封装，处理实例管理和响应式
 */
export const BaseChart: React.FC<BaseChartProps> = ({
  option,
  height = '400px',
  loading = false,
  onEvents,
  style,
}) => {
  const chartRef = useRef<ReactECharts>(null);
  const { token } = useToken();

  // 缓存option，避免不必要的重渲染
  const memoizedOption = useMemo(() => {
    // 应用主题色
    return {
      ...option,
      color: option.color || [
        token.colorPrimary,
        token.colorSuccess,
        token.colorWarning,
        token.colorError,
        token.colorInfo,
      ],
    };
  }, [option, token]);

  // 响应式处理
  useEffect(() => {
    const handleResize = () => {
      const chart = chartRef.current?.getEchartsInstance();
      if (chart) {
        chart.resize();
      }
    };

    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, []);

  if (loading) {
    return (
      <div
        style={{
          height,
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'center',
          ...style,
        }}
      >
        <Spin size="large" />
      </div>
    );
  }

  return (
    <ReactECharts
      ref={chartRef}
      option={memoizedOption}
      style={{ height, width: '100%', ...style }}
      onEvents={onEvents}
      notMerge={true}
      lazyUpdate={true}
    />
  );
};
