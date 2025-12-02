/**
 * PieChart - 饼图/环形图组件
 * 基于BaseChart封装的饼图组件
 */
import React, { useMemo } from 'react';
import type { EChartsOption } from 'echarts';
import { BaseChart } from './BaseChart';

/**
 * 饼图数据项
 */
export interface PieDataItem {
  /** 名称 */
  name: string;
  /** 数值 */
  value: number;
}

/**
 * PieChart Props
 */
export interface PieChartProps {
  /** 数据 */
  data: PieDataItem[];
  /** 自定义颜色 */
  colors?: string[];
  /** 是否显示图例 */
  showLegend?: boolean;
  /** 内径（环形图），如'50%' */
  innerRadius?: string;
  /** 高度 */
  height?: string | number;
  /** 加载状态 */
  loading?: boolean;
  /** 点击回调 */
  onClick?: (params: { name: string; value: number }) => void;
}

/**
 * PieChart组件
 * @description 饼图/环形图封装，支持点击交互
 */
export const PieChart: React.FC<PieChartProps> = ({
  data,
  colors,
  showLegend = true,
  innerRadius = '0%',
  height = '400px',
  loading = false,
  onClick,
}) => {
  const option: EChartsOption = useMemo(() => {
    return {
      tooltip: {
        trigger: 'item',
        formatter: '{b}: {c} ({d}%)',
      },
      legend: showLegend
        ? {
            orient: 'vertical',
            right: '10%',
            top: 'center',
          }
        : undefined,
      series: [
        {
          type: 'pie',
          radius: [innerRadius, '70%'],
          center: ['40%', '50%'],
          data: data,
          emphasis: {
            itemStyle: {
              shadowBlur: 10,
              shadowOffsetX: 0,
              shadowColor: 'rgba(0, 0, 0, 0.5)',
            },
          },
          label: {
            formatter: '{b}\n{d}%',
          },
          itemStyle: colors
            ? {
                color: (params) => colors[params.dataIndex % colors.length],
              }
            : undefined,
        },
      ],
    };
  }, [data, colors, showLegend, innerRadius]);

  const onEvents = useMemo(() => {
    return onClick
      ? {
          click: (params: any) => {
            onClick({ name: params.name, value: params.value });
          },
        }
      : undefined;
  }, [onClick]);

  return (
    <BaseChart
      option={option}
      height={height}
      loading={loading}
      onEvents={onEvents}
    />
  );
};
