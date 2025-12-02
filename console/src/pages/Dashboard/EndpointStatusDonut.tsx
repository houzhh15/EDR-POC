import React from 'react';
import { Card, Empty, Spin, Typography } from 'antd';
import { useNavigate } from 'react-router-dom';
import { PieChart } from '../../components/charts';
import { useDashboardStats } from '../../hooks/useDashboard';

const { Text } = Typography;

/**
 * 终端状态环形图组件
 * 展示终端的在线/离线/风险状态分布
 */
export const EndpointStatusDonut: React.FC = () => {
  const navigate = useNavigate();
  const { data: stats, isLoading, error } = useDashboardStats();

  // 点击扇形处理
  const handleSectorClick = (params: { name: string; value: number }) => {
    const statusMap: Record<string, string> = {
      '在线': 'online',
      '离线': 'offline',
      '风险': 'at_risk',
    };
    const status = statusMap[params.name];
    if (status) {
      navigate(`/endpoints?status=${status}`);
    }
  };

  if (isLoading) {
    return (
      <Card title="终端状态分布" style={{ height: '100%' }}>
        <div
          style={{
            display: 'flex',
            justifyContent: 'center',
            alignItems: 'center',
            height: 300,
          }}
        >
          <Spin />
        </div>
      </Card>
    );
  }

  if (error || !stats) {
    return (
      <Card title="终端状态分布" style={{ height: '100%' }}>
        <Empty description="加载失败" />
      </Card>
    );
  }

  // 转换数据格式
  const donutData = [
    { name: '在线', value: stats.endpoints.online },
    { name: '离线', value: stats.endpoints.offline },
    { name: '风险', value: stats.endpoints.at_risk },
  ];

  // 颜色配置
  const colors = ['#52c41a', '#bfbfbf', '#ff7a45'];

  // 计算在线率
  const onlineRate =
    stats.endpoints.total > 0
      ? ((stats.endpoints.online / stats.endpoints.total) * 100).toFixed(1)
      : '0.0';

  return (
    <Card
      title="终端状态分布"
      style={{ height: '100%' }}
      extra={
        <Text type="secondary">
          总数: {stats.endpoints.total} | 在线率: {onlineRate}%
        </Text>
      }
    >
      <PieChart
        data={donutData}
        colors={colors}
        innerRadius="60%" // 更大的环形图内径，方便中心显示文字
        onClick={handleSectorClick}
        height={300}
      />
    </Card>
  );
};
