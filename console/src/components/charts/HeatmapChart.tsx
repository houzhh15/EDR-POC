/**
 * HeatmapChart - 热力图组件
 * 基于BaseChart封装的热力图组件
 */
import React, { useMemo } from 'react';
import type { EChartsOption } from 'echarts';
import { BaseChart } from './BaseChart';

/**
 * HeatmapChart Props
 */
export interface HeatmapChartProps {
  /** 数据（二维数组，每项为[x索引, y索引, 值]） */
  data: [number, number, number][];
  /** X轴类目 */
  xCategories: string[];
  /** Y轴类目 */
  yCategories: string[];
  /** 颜色范围 */
  colorRange?: string[];
  /** 高度 */
  height?: string | number;
  /** 加载状态 */
  loading?: boolean;
  /** 点击回调 */
  onClick?: (params: { x: string; y: string; value: number }) => void;
}

/**
 * HeatmapChart组件
 * @description 热力图封装，用于MITRE ATT&CK矩阵等场景
 */
export const HeatmapChart: React.FC<HeatmapChartProps> = ({
  data,
  xCategories,
  yCategories,
  colorRange = ['#ebedf0', '#c6e48b', '#7bc96f', '#239a3b', '#196127'],
  height = '600px',
  loading = false,
  onClick,
}) => {
  const option: EChartsOption = useMemo(() => {
    // 计算最大值用于visualMap
    const maxValue = Math.max(...data.map((item) => item[2]));

    return {
      tooltip: {
        position: 'top',
        formatter: (params: any) => {
          const xIndex = params.data[0];
          const yIndex = params.data[1];
          const value = params.data[2];
          return `${xCategories[xIndex]}<br/>${yCategories[yIndex]}: ${value}`;
        },
      },
      grid: {
        height: '70%',
        top: '10%',
        left: '15%',
      },
      xAxis: {
        type: 'category',
        data: xCategories,
        splitArea: {
          show: true,
        },
        axisLabel: {
          interval: 0,
          rotate: 45,
        },
      },
      yAxis: {
        type: 'category',
        data: yCategories,
        splitArea: {
          show: true,
        },
      },
      visualMap: {
        min: 0,
        max: maxValue,
        calculable: true,
        orient: 'horizontal',
        left: 'center',
        bottom: '5%',
        inRange: {
          color: colorRange,
        },
      },
      series: [
        {
          name: 'Heatmap',
          type: 'heatmap',
          data: data,
          label: {
            show: false,
          },
          emphasis: {
            itemStyle: {
              shadowBlur: 10,
              shadowColor: 'rgba(0, 0, 0, 0.5)',
            },
          },
        },
      ],
    };
  }, [data, xCategories, yCategories, colorRange]);

  const onEvents = useMemo(() => {
    return onClick
      ? {
          click: (params: any) => {
            const xIndex = params.data[0];
            const yIndex = params.data[1];
            const value = params.data[2];
            onClick({
              x: xCategories[xIndex],
              y: yCategories[yIndex],
              value,
            });
          },
        }
      : undefined;
  }, [onClick, xCategories, yCategories]);

  return (
    <BaseChart
      option={option}
      height={height}
      loading={loading}
      onEvents={onEvents}
    />
  );
};
