import React from 'react';
import { Card, Statistic, Skeleton, Space } from 'antd';
import { ArrowUpOutlined, ArrowDownOutlined } from '@ant-design/icons';
import type { TrendDirection } from '../../types/dashboard';

export interface StatCardProps {
  /** 卡片标题 */
  title: string;
  /** 统计数值 */
  value: number;
  /** 图标 */
  icon: React.ReactNode;
  /** 图标颜色 */
  color: string;
  /** 趋势信息（可选） */
  trend?: {
    direction: TrendDirection;
    value: number;
  };
  /** 加载状态 */
  loading?: boolean;
  /** 点击回调 */
  onClick?: () => void;
}

/**
 * 统计卡片组件
 * 用于展示关键指标，支持趋势显示和交互
 */
export const StatCard: React.FC<StatCardProps> = ({
  title,
  value,
  icon,
  color,
  trend,
  loading = false,
  onClick,
}) => {
  if (loading) {
    return (
      <Card>
        <Skeleton active paragraph={{ rows: 2 }} />
      </Card>
    );
  }

  const getTrendColor = (direction: TrendDirection) => {
    switch (direction) {
      case 'up':
        return '#ff4d4f'; // 上升通常表示告警增加（负面）
      case 'down':
        return '#52c41a'; // 下降表示告警减少（正面）
      case 'flat':
      default:
        return '#8c8c8c'; // 平稳
    }
  };

  const getTrendIcon = (direction: TrendDirection) => {
    switch (direction) {
      case 'up':
        return <ArrowUpOutlined />;
      case 'down':
        return <ArrowDownOutlined />;
      case 'flat':
      default:
        return null;
    }
  };

  return (
    <Card
      hoverable={!!onClick}
      onClick={onClick}
      style={{
        height: '100%',
        cursor: onClick ? 'pointer' : 'default',
        transition: 'all 0.3s',
      }}
      styles={{
        body: {
          display: 'flex',
          flexDirection: 'column',
          justifyContent: 'space-between',
          height: '100%',
        },
      }}
      onMouseEnter={(e) => {
        if (onClick) {
          e.currentTarget.style.transform = 'translateY(-2px)';
          e.currentTarget.style.boxShadow =
            '0 4px 12px rgba(0, 0, 0, 0.15)';
        }
      }}
      onMouseLeave={(e) => {
        if (onClick) {
          e.currentTarget.style.transform = 'translateY(0)';
          e.currentTarget.style.boxShadow = 'none';
        }
      }}
    >
      {/* 上部：图标和数值 */}
      <Space
        size="large"
        style={{
          width: '100%',
          justifyContent: 'space-between',
          marginBottom: '16px',
        }}
      >
        {/* 左侧图标 */}
        <div
          style={{
            fontSize: 32,
            color,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}
        >
          {icon}
        </div>

        {/* 右侧统计数值 */}
        <Statistic title={title} value={value} />
      </Space>

      {/* 下部：趋势信息 */}
      {trend && (
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: '8px',
            color: getTrendColor(trend.direction),
            fontSize: '14px',
          }}
        >
          {getTrendIcon(trend.direction)}
          <span>
            {trend.direction === 'up' ? '+' : trend.direction === 'down' ? '-' : ''}
            {trend.value}% vs 上周期
          </span>
        </div>
      )}
    </Card>
  );
};
