/**
 * 操作系统类型标签组件
 */
import React from 'react';
import { Tag } from 'antd';
import { WindowsOutlined, AppleOutlined } from '@ant-design/icons';
import type { OSType } from '@/types/endpoint';

interface Props {
  osType: OSType;
}

const osConfig: Record<OSType, { color: string; icon: React.ReactNode; label: string }> = {
  windows: { color: 'blue', icon: <WindowsOutlined />, label: 'Windows' },
  linux: { color: 'orange', icon: null, label: 'Linux' },
  macos: { color: 'purple', icon: <AppleOutlined />, label: 'macOS' },
};

const OSTypeTag: React.FC<Props> = ({ osType }) => {
  const config = osConfig[osType] || { color: 'default', icon: null, label: osType };

  return (
    <Tag color={config.color} icon={config.icon}>
      {config.label}
    </Tag>
  );
};

export default OSTypeTag;
