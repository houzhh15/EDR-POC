/**
 * 终端详情页面
 */
import React from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Typography, Descriptions, Tabs, Button, Space, Spin, Card, Dropdown } from 'antd';
import { ArrowLeftOutlined, StopOutlined, MoreOutlined, SyncOutlined, DeleteOutlined, ReloadOutlined } from '@ant-design/icons';
import { useEndpointDetail } from './hooks/useEndpoints';
import { AgentStatusBadge, OSTypeTag } from '@/components/common';
import { useEndpointsUIStore } from '@/stores/endpoints';
import { IsolateModal, DeleteConfirmModal } from './components';
import type { MenuProps } from 'antd';

const { Title } = Typography;

/**
 * 格式化日期时间
 */
const formatDateTime = (dateStr: string | null | undefined): string => {
  if (!dateStr) return '-';
  const date = new Date(dateStr);
  return date.toLocaleString('zh-CN');
};

/**
 * 格式化相对时间
 */
const formatRelativeTime = (dateStr: string | null | undefined): string => {
  if (!dateStr) return '-';
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
 * 软件清单占位组件
 */
const SoftwareInventory: React.FC<{ endpointId: string }> = ({ endpointId }) => (
  <div>软件清单 - 终端ID: {endpointId}</div>
);

/**
 * 告警历史占位组件
 */
const AlertHistory: React.FC<{ endpointId: string }> = ({ endpointId }) => (
  <div>告警历史 - 终端ID: {endpointId}</div>
);

/**
 * 事件时间线占位组件
 */
const EventTimeline: React.FC<{ endpointId: string }> = ({ endpointId }) => (
  <div>事件时间线 - 终端ID: {endpointId}</div>
);

/**
 * 变更记录占位组件
 */
const ChangeHistory: React.FC<{ endpointId: string }> = ({ endpointId }) => (
  <div>变更记录 - 终端ID: {endpointId}</div>
);

const EndpointDetailPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { data, isLoading, error } = useEndpointDetail(id!);
  const { openIsolateModal, openDeleteModal } = useEndpointsUIStore();

  // 更多操作下拉菜单
  const moreMenuItems: MenuProps['items'] = [
    {
      key: 'scan',
      label: '触发扫描',
      icon: <SyncOutlined />,
    },
    {
      key: 'refresh',
      label: '刷新信息',
      icon: <ReloadOutlined />,
    },
    {
      type: 'divider',
    },
    {
      key: 'delete',
      label: '删除终端',
      icon: <DeleteOutlined />,
      danger: true,
    },
  ];

  const handleMoreMenuClick: MenuProps['onClick'] = ({ key }) => {
    switch (key) {
      case 'scan':
        // TODO: 实现触发扫描功能
        console.log('触发扫描:', id);
        break;
      case 'refresh':
        // TODO: 实现刷新信息功能
        window.location.reload();
        break;
      case 'delete':
        openDeleteModal(id!);
        break;
    }
  };

  if (isLoading) {
    return (
      <div style={{ textAlign: 'center', padding: 100 }}>
        <Spin size="large" />
      </div>
    );
  }

  if (error || !data) {
    return <div>加载失败</div>;
  }

  const endpoint = data.data;

  const tabItems = [
    { key: 'software', label: '软件清单', children: <SoftwareInventory endpointId={id!} /> },
    { key: 'alerts', label: '告警历史', children: <AlertHistory endpointId={id!} /> },
    { key: 'timeline', label: '事件时间线', children: <EventTimeline endpointId={id!} /> },
    { key: 'changes', label: '变更记录', children: <ChangeHistory endpointId={id!} /> },
  ];

  return (
    <div className="endpoint-detail-page">
      <div className="page-header" style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <Space>
          <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/endpoints')}>返回列表</Button>
          <Title level={2} style={{ margin: 0 }}>{endpoint.hostname}</Title>
          <AgentStatusBadge status={endpoint.status} lastSeenAt={endpoint.last_seen_at} showText />
        </Space>
        <Space>
          <Button icon={<StopOutlined />} danger onClick={() => openIsolateModal(endpoint.id)}>网络隔离</Button>
          <Dropdown menu={{ items: moreMenuItems, onClick: handleMoreMenuClick }} trigger={['click']}>
            <Button icon={<MoreOutlined />}>更多操作</Button>
          </Dropdown>
        </Space>
      </div>

      <Card title="系统信息" style={{ marginBottom: 16 }}>
        <Descriptions column={3}>
          <Descriptions.Item label="主机名">{endpoint.hostname}</Descriptions.Item>
          <Descriptions.Item label="操作系统">
            <Space>
              <OSTypeTag osType={endpoint.os_type} />
              {endpoint.os_version}
            </Space>
          </Descriptions.Item>
          <Descriptions.Item label="系统架构">{endpoint.architecture}</Descriptions.Item>
          <Descriptions.Item label="IP地址">{endpoint.ip_addresses?.join(', ') || '-'}</Descriptions.Item>
          <Descriptions.Item label="MAC地址">{endpoint.mac_addresses?.join(', ') || '-'}</Descriptions.Item>
          <Descriptions.Item label="Agent版本">{endpoint.agent_version}</Descriptions.Item>
          <Descriptions.Item label="Agent ID">{endpoint.agent_id}</Descriptions.Item>
          <Descriptions.Item label="首次发现">{formatDateTime(endpoint.first_seen_at)}</Descriptions.Item>
          <Descriptions.Item label="最后心跳">{formatRelativeTime(endpoint.last_seen_at)}</Descriptions.Item>
        </Descriptions>
      </Card>

      <Card>
        <Tabs items={tabItems} />
      </Card>

      <IsolateModal />
      <DeleteConfirmModal onSuccess={() => navigate('/endpoints')} />
    </div>
  );
};

export default EndpointDetailPage;
