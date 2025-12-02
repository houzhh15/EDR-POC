/**
 * 安全态势仪表盘页面
 * 整合所有子组件，展示企业整体安全状态、威胁趋势和关键安全指标
 */
import React, { useEffect } from 'react';
import { Typography, Row, Col, Spin } from 'antd';
import {
  AlertOutlined,
  DesktopOutlined,
  SafetyCertificateOutlined,
  FileSearchOutlined,
} from '@ant-design/icons';
import {
  DashboardToolbar,
  StatCard,
  AlertSeverityPie,
  EndpointStatusDonut,
  AlertTrendChart,
  TopNList,
  MitreHeatmap,
  AttackChainPanel,
  MetricsCard,
} from './Dashboard';
import { useDashboardStats, useDashboardRefresh } from '../hooks/useDashboard';
import { useDashboardStore } from '../stores/dashboardStore';

const { Title } = Typography;

/**
 * 仪表盘页面组件
 * @description 
 * - 7行响应式布局（工具栏、统计卡片、趋势图、分布图、Top N、热力图、攻击链）
 * - 自动刷新机制（可配置间隔10-300秒）
 * - React Query数据缓存管理
 * - Zustand UI状态持久化
 */
const DashboardPage: React.FC = () => {
  const { data: stats, isLoading } = useDashboardStats();
  const { refresh } = useDashboardRefresh();
  const { autoRefresh, refreshInterval, setLastUpdated } = useDashboardStore();

  // 自动刷新逻辑
  useEffect(() => {
    if (!autoRefresh) return;

    const interval = setInterval(() => {
      refresh();
      setLastUpdated(new Date());
    }, refreshInterval * 1000);

    return () => clearInterval(interval);
  }, [autoRefresh, refreshInterval, refresh, setLastUpdated]);

  // 主要数据加载中，显示全屏加载
  if (isLoading && !stats) {
    return (
      <div
        style={{
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'center',
          height: '100vh',
        }}
      >
        <Spin size="large" tip="加载仪表盘数据..." />
      </div>
    );
  }

  return (
    <div style={{ padding: '24px' }}>
      {/* 页面标题 */}
      <Title level={2} style={{ marginBottom: '24px' }}>
        安全态势仪表盘
      </Title>

      {/* 第一行：工具栏 */}
      <DashboardToolbar />

      {/* 第二行：4个统计卡片 */}
      <Row gutter={[16, 16]} style={{ marginBottom: '16px' }}>
        <Col xs={24} sm={12} lg={6}>
          <StatCard
            title="待处理告警"
            value={stats?.alerts.total || 0}
            icon={<AlertOutlined />}
            color="#ff4d4f"
            trend={stats?.alerts.trend}
            loading={isLoading}
          />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <StatCard
            title="在线终端"
            value={stats?.endpoints.online || 0}
            icon={<DesktopOutlined />}
            color="#52c41a"
            loading={isLoading}
          />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <StatCard
            title="检测规则"
            value={stats?.rules.enabled || 0}
            icon={<SafetyCertificateOutlined />}
            color="#1890ff"
            loading={isLoading}
          />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <StatCard
            title="今日事件"
            value={stats?.events_today || 0}
            icon={<FileSearchOutlined />}
            color="#fa8c16"
            loading={isLoading}
          />
        </Col>
      </Row>

      {/* 第三行：告警趋势图 */}
      <Row gutter={[16, 16]} style={{ marginBottom: '16px' }}>
        <Col xs={24}>
          <AlertTrendChart />
        </Col>
      </Row>

      {/* 第四行：告警严重性饼图 + 终端状态环形图 */}
      <Row gutter={[16, 16]} style={{ marginBottom: '16px' }}>
        <Col xs={24} lg={12}>
          <AlertSeverityPie />
        </Col>
        <Col xs={24} lg={12}>
          <EndpointStatusDonut />
        </Col>
      </Row>

      {/* 第五行：Top N列表 + 指标卡片 */}
      <Row gutter={[16, 16]} style={{ marginBottom: '16px' }}>
        <Col xs={24} sm={12} lg={6}>
          <TopNList type="process" title="Top 进程" limit={5} />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <TopNList type="ip" title="Top IP地址" limit={5} />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <TopNList type="domain" title="Top 域名" limit={5} />
        </Col>
        <Col xs={24} lg={6}>
          <MetricsCard />
        </Col>
      </Row>

      {/* 第六行：MITRE ATT&CK热力图 */}
      <Row gutter={[16, 16]} style={{ marginBottom: '16px' }}>
        <Col xs={24}>
          <MitreHeatmap />
        </Col>
      </Row>

      {/* 第七行：攻击链可视化 */}
      <Row gutter={[16, 16]}>
        <Col xs={24}>
          <AttackChainPanel />
        </Col>
      </Row>
    </div>
  );
};

export default DashboardPage;
