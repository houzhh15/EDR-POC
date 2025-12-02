/**
 * 终端表格组件
 */
import React from 'react';
import { Table, Space, Dropdown, Button } from 'antd';
import { MoreOutlined, EyeOutlined, StopOutlined, DeleteOutlined } from '@ant-design/icons';
import { Link, useNavigate } from 'react-router-dom';
import type { ColumnsType } from 'antd/es/table';
import { AgentStatusBadge, OSTypeTag, IPAddressCell } from '@/components/common';
import { useEndpointsUIStore } from '@/stores/endpoints';
import type { Endpoint, Pagination } from '@/types/endpoint';

interface Props {
  data: Endpoint[];
  pagination?: Pagination;
  loading: boolean;
  onPageChange: (page: number, pageSize: number) => void;
}

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

const EndpointTable: React.FC<Props> = ({ data, pagination, loading, onPageChange }) => {
  const navigate = useNavigate();
  const { selectedIds, setSelectedIds, openIsolateModal, openDeleteModal } = useEndpointsUIStore();

  const columns: ColumnsType<Endpoint> = [
    {
      title: '终端名称',
      dataIndex: 'hostname',
      key: 'hostname',
      sorter: true,
      render: (hostname, record) => (
        <Link to={`/endpoints/${record.id}`}>{hostname}</Link>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 120,
      sorter: true,
      render: (status, record) => (
        <AgentStatusBadge status={status} lastSeenAt={record.last_seen_at} showText />
      ),
    },
    {
      title: '操作系统',
      key: 'os',
      width: 150,
      sorter: true,
      render: (_, record) => (
        <Space>
          <OSTypeTag osType={record.os_type} />
          <span>{record.os_version}</span>
        </Space>
      ),
    },
    {
      title: 'IP地址',
      dataIndex: 'ip_addresses',
      key: 'ip_addresses',
      width: 160,
      render: (ips) => <IPAddressCell addresses={ips} />,
    },
    {
      title: 'Agent版本',
      dataIndex: 'agent_version',
      key: 'agent_version',
      width: 100,
    },
    {
      title: '最后心跳',
      dataIndex: 'last_seen_at',
      key: 'last_seen_at',
      width: 120,
      sorter: true,
      defaultSortOrder: 'descend',
      render: (time) => formatRelativeTime(time),
    },
    {
      title: '操作',
      key: 'action',
      width: 100,
      fixed: 'right',
      render: (_, record) => (
        <Dropdown
          menu={{
            items: [
              { key: 'view', label: '查看详情', icon: <EyeOutlined /> },
              { key: 'isolate', label: '网络隔离', icon: <StopOutlined />, danger: true },
              { type: 'divider' },
              { key: 'delete', label: '删除', icon: <DeleteOutlined />, danger: true },
            ],
            onClick: ({ key }) => {
              if (key === 'view') navigate(`/endpoints/${record.id}`);
              if (key === 'isolate') openIsolateModal(record.id);
              if (key === 'delete') openDeleteModal(record.id);
            },
          }}
        >
          <Button type="text" icon={<MoreOutlined />} />
        </Dropdown>
      ),
    },
  ];

  return (
    <Table
      rowKey="id"
      columns={columns}
      dataSource={data}
      loading={loading}
      rowSelection={{
        selectedRowKeys: selectedIds,
        onChange: (keys) => setSelectedIds(keys as string[]),
      }}
      pagination={{
        current: pagination?.page,
        pageSize: pagination?.page_size,
        total: pagination?.total,
        showSizeChanger: true,
        pageSizeOptions: ['10', '20', '50', '100'],
        showTotal: (total) => `共 ${total} 条`,
        onChange: onPageChange,
      }}
      scroll={{ x: 1000 }}
    />
  );
};

export default EndpointTable;
