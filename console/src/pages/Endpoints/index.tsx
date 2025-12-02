/**
 * 终端列表页面
 */
import React from 'react';
import { Typography, Button, Space } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import { useSearchParams } from 'react-router-dom';
import { EndpointFilters, EndpointTable, IsolateModal, DeleteConfirmModal } from './components';
import { useEndpoints } from './hooks/useEndpoints';
import { useEndpointsUIStore } from '@/stores/endpoints';
import type { EndpointListParams, EndpointStatus, OSType } from '@/types/endpoint';

const { Title } = Typography;

const EndpointsPage: React.FC = () => {
  const [searchParams, setSearchParams] = useSearchParams();

  // 从URL解析查询参数
  const params: EndpointListParams = {
    page: Number(searchParams.get('page')) || 1,
    page_size: Number(searchParams.get('page_size')) || 20,
    status: (searchParams.get('status') as EndpointStatus) || undefined,
    os_type: (searchParams.get('os_type') as OSType) || undefined,
    hostname: searchParams.get('hostname') || undefined,
    sort_by: (searchParams.get('sort_by') as EndpointListParams['sort_by']) || 'last_seen_at',
    sort_order: (searchParams.get('sort_order') as EndpointListParams['sort_order']) || 'desc',
  };

  const { data, isLoading, refetch } = useEndpoints(params);
  const { selectedIds, clearSelection } = useEndpointsUIStore();

  // 更新URL参数
  const handleFiltersChange = (newParams: Partial<EndpointListParams>) => {
    const merged = { ...params, ...newParams, page: 1 };
    const newSearchParams = new URLSearchParams();
    Object.entries(merged).forEach(([key, value]) => {
      if (value !== undefined && value !== null && value !== '') {
        newSearchParams.set(key, String(value));
      }
    });
    setSearchParams(newSearchParams);
  };

  // 分页变化
  const handlePageChange = (page: number, pageSize: number) => {
    const newSearchParams = new URLSearchParams(searchParams);
    newSearchParams.set('page', String(page));
    newSearchParams.set('page_size', String(pageSize));
    setSearchParams(newSearchParams);
  };

  return (
    <div className="endpoints-page">
      <div className="page-header" style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <Title level={2} style={{ margin: 0 }}>终端管理</Title>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => refetch()}>刷新</Button>
        </Space>
      </div>

      <EndpointFilters values={params} onChange={handleFiltersChange} />

      {selectedIds.length > 0 && (
        <div className="selection-bar" style={{ marginBottom: 16, padding: '8px 16px', background: '#e6f7ff', borderRadius: 4 }}>
          已选择 {selectedIds.length} 项
          <Button size="small" style={{ marginLeft: 8 }} onClick={clearSelection}>取消选择</Button>
        </div>
      )}

      <EndpointTable
        data={data?.data || []}
        pagination={data?.pagination}
        loading={isLoading}
        onPageChange={handlePageChange}
      />

      <IsolateModal />
      <DeleteConfirmModal />
    </div>
  );
};

export default EndpointsPage;
