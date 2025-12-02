/**
 * Agent 状态徽章组件
 */
import React from 'react';
import { Badge, Tooltip } from 'antd';
import type { EndpointStatus } from '@/types/endpoint';

interface Props {
  status: EndpointStatus;
  lastSeenAt?: string | null;
  showText?: boolean;
}

const statusConfig: Record<EndpointStatus, { color: string; text: string }> = {
  online: { color: '#52c41a', text: '在线' },
  offline: { color: '#ff4d4f', text: '离线' },
  unknown: { color: '#8c8c8c', text: '未知' },
};

/**
 * 格式化相对时间
 */
const formatRelativeTime = (dateStr: string | null | undefined): string => {
  if (!dateStr) return '未知';
  const date = new Date(dateStr);
  const now = new Date();
  const diff = now.getTime() - date.getTime();
  const minutes = Math.floor(diff / 60000);
  const hours = Math.floor(diff / 3600000);
  const days = Math.floor(diff / 86400000);

  if (minutes < 1) return '刚刚';
  if (minutes < 60) return `${minutes} 分钟前`;
  if (hours < 24) return `${hours} 小时前`;
  return `${days} 天前`;
};

/**
 * 获取时间差（小时）
 */
const getTimeDiffInHours = (dateStr: string | null | undefined): number => {
  if (!dateStr) return 0;
  const date = new Date(dateStr);
  const now = new Date();
  return (now.getTime() - date.getTime()) / 3600000;
};

const AgentStatusBadge: React.FC<Props> = ({ status, lastSeenAt, showText = false }) => {
  const config = statusConfig[status] || statusConfig.unknown;

  let displayText = config.text;
  let tooltipText = config.text;

  if (lastSeenAt) {
    const relativeTime = formatRelativeTime(lastSeenAt);
    if (status === 'offline') {
      const hours = getTimeDiffInHours(lastSeenAt);
      if (hours >= 1) {
        displayText = `离线 (已失联 ${Math.floor(hours)} 小时)`;
      }
    }
    tooltipText = `最后心跳: ${relativeTime}`;
  }

  const badge = <Badge color={config.color} text={showText ? displayText : undefined} />;

  return lastSeenAt ? (
    <Tooltip title={tooltipText}>{badge}</Tooltip>
  ) : (
    badge
  );
};

export default AgentStatusBadge;
