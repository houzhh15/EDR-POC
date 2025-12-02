import React from 'react';
import { Card, Empty, Spin } from 'antd';
import { LineChart } from '../../components/charts';
import { useDashboardStore } from '../../stores/dashboardStore';
import { useAlertTrend } from '../../hooks/useDashboard';
import dayjs from 'dayjs';

/**
 * 告警趋势折线图组件
 * 展示不同严重性级别的告警随时间变化趋势
 */
export const AlertTrendChart: React.FC = () => {
  const { timeRange } = useDashboardStore();
  const { data, isLoading, error } = useAlertTrend(timeRange);

  if (isLoading) {
    return (
      <Card title="告警趋势" style={{ height: '100%' }}>
        <div
          style={{
            display: 'flex',
            justifyContent: 'center',
            alignItems: 'center',
            height: 400,
          }}
        >
          <Spin />
        </div>
      </Card>
    );
  }

  if (error || !data) {
    return (
      <Card title="告警趋势" style={{ height: '100%' }}>
        <Empty description="加载失败" />
      </Card>
    );
  }

  // 转换数据格式：格式化时间戳并展开为多行（ECharts多系列需要）
  const formattedData: any[] = [];
  data.forEach((point) => {
    const timestamp =
      timeRange === '24h'
        ? dayjs(point.timestamp).format('HH:mm')
        : dayjs(point.timestamp).format('MM-DD');
    
    formattedData.push(
      { timestamp, severity: '严重', count: point.critical },
      { timestamp, severity: '高危', count: point.high },
      { timestamp, severity: '中危', count: point.medium },
      { timestamp, severity: '低危', count: point.low }
    );
  });

  return (
    <Card
      title="告警趋势"
      style={{ height: '100%' }}
      extra={<span style={{ fontSize: '12px', color: '#8c8c8c' }}>时间范围: {timeRange}</span>}
    >
      <LineChart
        data={formattedData}
        xField="timestamp"
        yField="count"
        seriesField="severity"
        smooth
        areaStyle
        height={400}
      />
    </Card>
  );
};
