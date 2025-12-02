import React from 'react';
import { ClockCircleOutlined, ThunderboltOutlined } from '@ant-design/icons';
import { StatCard } from './StatCard';
import { useDashboardStats } from '../../hooks/useDashboard';

/**
 * 指标卡片组件
 * 展示MTTD（平均检测时间）和MTTR（平均响应时间）
 */
export const MetricsCard: React.FC = () => {
  const { data: stats, isLoading } = useDashboardStats();

  // 格式化时间（分钟转为小时或分钟）
  const formatTime = (minutes: number) => {
    if (minutes >= 60) {
      const hours = (minutes / 60).toFixed(1);
      return { value: parseFloat(hours), unit: '小时' };
    }
    return { value: Math.round(minutes), unit: '分钟' };
  };

  // MTTD数据
  const mttd = stats ? formatTime(stats.mttd_minutes) : { value: 0, unit: '分钟' };
  // MTTR数据
  const mttr = stats ? formatTime(stats.mttr_minutes) : { value: 0, unit: '分钟' };

  // 简化版趋势（实际应该从后端获取对比数据）
  // 这里演示随机趋势，实际应该替换为真实数据
  const getMockTrend = () => {
    const directions: ('up' | 'down' | 'flat')[] = ['up', 'down', 'flat'];
    const randomDirection = directions[Math.floor(Math.random() * directions.length)];
    const randomValue = Math.floor(Math.random() * 20) + 1;
    return { direction: randomDirection, value: randomValue };
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '16px', height: '600px' }}>
      {/* MTTD卡片 */}
      <div style={{ flex: 1 }}>
        <StatCard
          title={`平均检测时间 (${mttd.unit})`}
          value={mttd.value}
          icon={<ClockCircleOutlined />}
          color="#1890ff"
          trend={stats ? getMockTrend() : undefined}
          loading={isLoading}
        />
      </div>

      {/* MTTR卡片 */}
      <div style={{ flex: 1 }}>
        <StatCard
          title={`平均响应时间 (${mttr.unit})`}
          value={mttr.value}
          icon={<ThunderboltOutlined />}
          color="#52c41a"
          trend={stats ? getMockTrend() : undefined}
          loading={isLoading}
        />
      </div>
    </div>
  );
};
