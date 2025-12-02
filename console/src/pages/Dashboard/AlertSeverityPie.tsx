import React from 'react';
import { Card, Empty, Spin } from 'antd';
import { useNavigate } from 'react-router-dom';
import { PieChart } from '../../components/charts';
import { useDashboardStats } from '../../hooks/useDashboard';

/**
 * 告警严重性饼图组件
 * 展示告警的严重性分布（Critical/High/Medium/Low）
 */
export const AlertSeverityPie: React.FC = () => {
  const navigate = useNavigate();
  const { data: stats, isLoading, error } = useDashboardStats();

  // 点击扇形处理
  const handleSectorClick = (params: { name: string; value: number }) => {
    const severityMap: Record<string, string> = {
      '严重': 'critical',
      '高危': 'high',
      '中危': 'medium',
      '低危': 'low',
    };
    const severity = severityMap[params.name];
    if (severity) {
      navigate(`/alerts?severity=${severity}`);
    }
  };

  if (isLoading) {
    return (
      <Card title="告警严重性分布" style={{ height: '100%' }}>
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
      <Card title="告警严重性分布" style={{ height: '100%' }}>
        <Empty description="加载失败" />
      </Card>
    );
  }

  // 转换数据格式
  const pieData = [
    { name: '严重', value: stats.alerts.critical },
    { name: '高危', value: stats.alerts.high },
    { name: '中危', value: stats.alerts.medium },
    { name: '低危', value: stats.alerts.low },
  ];

  // 颜色配置（对应严重性级别）
  const colors = ['#ff4d4f', '#fa8c16', '#fadb14', '#52c41a'];

  return (
    <Card title="告警严重性分布" style={{ height: '100%' }}>
      <PieChart
        data={pieData}
        colors={colors}
        innerRadius="50%" // 环形图
        onClick={handleSectorClick}
        height={300}
      />
    </Card>
  );
};
