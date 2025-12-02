import React from 'react';
import { Space, Select, Switch, Button, Typography, InputNumber } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import { useDashboardStore } from '../../stores/dashboardStore';
import { useDashboardRefresh } from '../../hooks/useDashboard';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import 'dayjs/locale/zh-cn';

dayjs.extend(relativeTime);
dayjs.locale('zh-cn');

const { Text } = Typography;

/**
 * 仪表板工具栏组件
 * 功能：时间范围选择、自动刷新开关、刷新间隔设置、手动刷新、上次更新时间显示
 */
export const DashboardToolbar: React.FC = () => {
  const {
    timeRange,
    setTimeRange,
    autoRefresh,
    toggleAutoRefresh,
    refreshInterval,
    setRefreshInterval,
    lastUpdated,
  } = useDashboardStore();

  const { refresh } = useDashboardRefresh();
  const [isRefreshing, setIsRefreshing] = React.useState(false);

  const handleRefresh = async () => {
    setIsRefreshing(true);
    await refresh();
    setTimeout(() => setIsRefreshing(false), 500);
  };

  const timeRangeOptions = [
    { label: '近1小时', value: '1h' },
    { label: '近6小时', value: '6h' },
    { label: '近24小时', value: '24h' },
    { label: '近7天', value: '7d' },
    { label: '近30天', value: '30d' },
  ];

  return (
    <Space
      size="middle"
      style={{
        width: '100%',
        justifyContent: 'space-between',
        padding: '16px',
        background: '#fff',
        borderRadius: '8px',
        marginBottom: '16px',
        flexWrap: 'wrap',
      }}
    >
      {/* 左侧控制区 */}
      <Space size="middle" wrap>
        {/* 时间范围选择器 */}
        <Space>
          <Text>时间范围:</Text>
          <Select
            value={timeRange}
            onChange={setTimeRange}
            options={timeRangeOptions}
            style={{ width: 120 }}
          />
        </Space>

        {/* 自动刷新开关 */}
        <Space>
          <Text>自动刷新:</Text>
          <Switch checked={autoRefresh} onChange={toggleAutoRefresh} />
        </Space>

        {/* 刷新间隔设置 */}
        {autoRefresh && (
          <Space>
            <Text>间隔(秒):</Text>
            <InputNumber
              min={10}
              max={300}
              value={refreshInterval}
              onChange={(value) => value && setRefreshInterval(value)}
              style={{ width: 80 }}
            />
          </Space>
        )}

        {/* 手动刷新按钮 */}
        <Button
          type="primary"
          icon={<ReloadOutlined spin={isRefreshing} />}
          onClick={handleRefresh}
          loading={isRefreshing}
        >
          刷新
        </Button>
      </Space>

      {/* 右侧上次更新时间 */}
      <Space>
        <Text type="secondary">
          上次更新: {lastUpdated ? dayjs(lastUpdated).fromNow() : '未更新'}
        </Text>
      </Space>
    </Space>
  );
};
