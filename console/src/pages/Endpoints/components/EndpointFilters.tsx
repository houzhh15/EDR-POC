/**
 * 终端筛选组件
 */
import React from 'react';
import { Input, Select, Space, Button } from 'antd';
import { ClearOutlined } from '@ant-design/icons';
import type { EndpointListParams, EndpointStatus, OSType } from '@/types/endpoint';

interface Props {
  values: EndpointListParams;
  onChange: (params: Partial<EndpointListParams>) => void;
}

const statusOptions = [
  { value: '', label: '全部状态' },
  { value: 'online', label: '在线' },
  { value: 'offline', label: '离线' },
  { value: 'unknown', label: '未知' },
];

const osTypeOptions = [
  { value: '', label: '全部系统' },
  { value: 'windows', label: 'Windows' },
  { value: 'linux', label: 'Linux' },
  { value: 'macos', label: 'macOS' },
];

const EndpointFilters: React.FC<Props> = ({ values, onChange }) => {
  const handleClear = () => {
    onChange({ status: undefined, os_type: undefined, hostname: undefined, ip: undefined });
  };

  return (
    <div className="endpoint-filters" style={{ marginBottom: 16 }}>
      <Space wrap>
        <Input.Search
          placeholder="搜索主机名或IP"
          allowClear
          style={{ width: 240 }}
          value={values.hostname}
          onChange={(e) => onChange({ hostname: e.target.value || undefined })}
          onSearch={(val) => onChange({ hostname: val || undefined })}
        />
        <Select
          placeholder="状态"
          style={{ width: 120 }}
          options={statusOptions}
          value={values.status || ''}
          onChange={(val) => onChange({ status: (val || undefined) as EndpointStatus | undefined })}
        />
        <Select
          placeholder="操作系统"
          style={{ width: 120 }}
          options={osTypeOptions}
          value={values.os_type || ''}
          onChange={(val) => onChange({ os_type: (val || undefined) as OSType | undefined })}
        />
        <Button icon={<ClearOutlined />} onClick={handleClear}>清除筛选</Button>
      </Space>
    </div>
  );
};

export default EndpointFilters;
