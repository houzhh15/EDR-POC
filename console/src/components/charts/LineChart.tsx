/**
 * LineChart - 折线图组件
 * 基于BaseChart封装的折线图组件
 */
import React, { useMemo } from 'react';
import type { EChartsOption } from 'echarts';
import { BaseChart } from './BaseChart';

/**
 * LineChart Props
 */
export interface LineChartProps {
  /** 数据 */
  data: Record<string, any>[];
  /** X轴字段名 */
  xField: string;
  /** Y轴字段名（单字段或多字段） */
  yField: string | string[];
  /** 系列字段（用于分组） */
  seriesField?: string;
  /** 是否平滑曲线 */
  smooth?: boolean;
  /** 高度 */
  height?: string | number;
  /** 加载状态 */
  loading?: boolean;
  /** 是否显示区域填充 */
  areaStyle?: boolean;
}

/**
 * LineChart组件
 * @description 折线图封装，支持多系列、缩放、区域填充
 */
export const LineChart: React.FC<LineChartProps> = ({
  data,
  xField,
  yField,
  smooth = true,
  height = '400px',
  loading = false,
  areaStyle = false,
}) => {
  const option: EChartsOption = useMemo(() => {
    // 提取X轴数据
    const xAxisData = data.map((item) => item[xField]);

    // 处理Y轴字段（支持单个或多个）
    const yFields = Array.isArray(yField) ? yField : [yField];

    // 构建series
    const series = yFields.map((field) => ({
      name: field,
      type: 'line' as const,
      smooth,
      data: data.map((item) => item[field]),
      areaStyle: areaStyle ? {} : undefined,
    }));

    return {
      tooltip: {
        trigger: 'axis',
        axisPointer: {
          type: 'cross',
        },
      },
      legend: {
        data: yFields,
        top: 10,
      },
      grid: {
        left: '3%',
        right: '4%',
        bottom: '15%',
        top: '15%',
        containLabel: true,
      },
      xAxis: {
        type: 'category',
        boundaryGap: false,
        data: xAxisData,
      },
      yAxis: {
        type: 'value',
      },
      dataZoom: [
        {
          type: 'inside',
          start: 0,
          end: 100,
        },
        {
          start: 0,
          end: 100,
        },
      ],
      series,
    };
  }, [data, xField, yField, smooth, areaStyle]);

  return <BaseChart option={option} height={height} loading={loading} />;
};
