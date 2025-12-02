/**
 * IP 地址单元格组件
 */
import React from 'react';
import { Tooltip, Typography } from 'antd';

interface Props {
  addresses: string[];
  maxDisplay?: number;
}

const IPAddressCell: React.FC<Props> = ({ addresses = [], maxDisplay = 1 }) => {
  if (!addresses || addresses.length === 0) {
    return <Typography.Text type="secondary">-</Typography.Text>;
  }

  const displayIPs = addresses.slice(0, maxDisplay);
  const remainingCount = addresses.length - maxDisplay;

  if (remainingCount <= 0) {
    return <span>{displayIPs.join(', ')}</span>;
  }

  return (
    <Tooltip title={addresses.join('\n')} placement="topLeft">
      <span>
        {displayIPs.join(', ')}
        <Typography.Text type="secondary"> +{remainingCount}</Typography.Text>
      </span>
    </Tooltip>
  );
};

export default IPAddressCell;
