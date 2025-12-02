import React, { useEffect } from 'react';
import { Card, Row, Col, List, Tag, Empty, Spin, Typography } from 'antd';
import { ClockCircleOutlined, DesktopOutlined } from '@ant-design/icons';
import { useAttackChains } from '../../hooks/useDashboard';
import { useDashboardStore } from '../../stores/dashboardStore';
import { G6Graph } from '../../components/charts';
import type { Severity } from '../../types/dashboard';
import dayjs from 'dayjs';

const { Text } = Typography;

/**
 * 攻击链面板组件
 * 左侧显示攻击链列表，右侧显示选中链的关系图
 */
export const AttackChainPanel: React.FC = () => {
  const { data: chains, isLoading, error } = useAttackChains(5);
  const { selectedChainId, setSelectedChain } = useDashboardStore();

  // 默认选中第一条链
  useEffect(() => {
    if (chains && chains.length > 0 && !selectedChainId) {
      setSelectedChain(chains[0].id);
    }
  }, [chains, selectedChainId, setSelectedChain]);

  // 严重性颜色
  const getSeverityColor = (severity: Severity) => {
    const colorMap: Record<Severity, string> = {
      critical: 'red',
      high: 'orange',
      medium: 'gold',
      low: 'blue',
    };
    return colorMap[severity];
  };

  // 严重性文本
  const getSeverityText = (severity: Severity) => {
    const textMap: Record<Severity, string> = {
      critical: '严重',
      high: '高危',
      medium: '中危',
      low: '低危',
    };
    return textMap[severity];
  };

  if (isLoading) {
    return (
      <Card title="攻击链分析" style={{ height: '100%' }}>
        <div
          style={{
            display: 'flex',
            justifyContent: 'center',
            alignItems: 'center',
            height: 500,
          }}
        >
          <Spin />
        </div>
      </Card>
    );
  }

  if (error || !chains || chains.length === 0) {
    return (
      <Card title="攻击链分析" style={{ height: '100%' }}>
        <Empty description="暂无攻击链数据" />
      </Card>
    );
  }

  // 获取选中的链
  const selectedChain = chains.find((chain) => chain.id === selectedChainId) || chains[0];

  return (
    <Card title="攻击链分析" style={{ height: '100%' }}>
      <Row gutter={16} style={{ minHeight: 500 }}>
        {/* 左侧：攻击链列表 */}
        <Col xs={24} lg={8}>
          <List
            dataSource={chains}
            renderItem={(chain) => (
              <List.Item
                key={chain.id}
                onClick={() => setSelectedChain(chain.id)}
                style={{
                  cursor: 'pointer',
                  padding: '12px',
                  background: chain.id === selectedChainId ? '#e6f7ff' : 'transparent',
                  borderRadius: '4px',
                  marginBottom: '8px',
                  border: chain.id === selectedChainId ? '1px solid #1890ff' : '1px solid transparent',
                  transition: 'all 0.3s',
                }}
              >
                <List.Item.Meta
                  title={
                    <div
                      style={{
                        display: 'flex',
                        justifyContent: 'space-between',
                        alignItems: 'center',
                      }}
                    >
                      <Text strong ellipsis style={{ maxWidth: '60%' }}>
                        {chain.id}
                      </Text>
                      <Tag color={getSeverityColor(chain.severity)}>
                        {getSeverityText(chain.severity)}
                      </Tag>
                    </div>
                  }
                  description={
                    <div>
                      <div
                        style={{
                          display: 'flex',
                          alignItems: 'center',
                          gap: '4px',
                          marginBottom: '4px',
                        }}
                      >
                        <ClockCircleOutlined style={{ fontSize: '12px' }} />
                        <Text type="secondary" style={{ fontSize: '12px' }}>
                          {dayjs(chain.created_at).format('YYYY-MM-DD HH:mm')}
                        </Text>
                      </div>
                      <div style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
                        <DesktopOutlined style={{ fontSize: '12px' }} />
                        <Text type="secondary" style={{ fontSize: '12px' }}>
                          {chain.nodes.length} 个节点，{chain.edges.length} 条边
                        </Text>
                      </div>
                    </div>
                  }
                />
              </List.Item>
            )}
          />
        </Col>

        {/* 右侧：攻击链关系图 */}
        <Col xs={24} lg={16}>
          <div
            style={{
              border: '1px solid #f0f0f0',
              borderRadius: '4px',
              padding: '16px',
              background: '#fafafa',
              minHeight: 480,
            }}
          >
            <div style={{ marginBottom: '12px' }}>
              <Text strong>攻击链: {selectedChain.id}</Text>
              <Tag
                color={getSeverityColor(selectedChain.severity)}
                style={{ marginLeft: '8px' }}
              >
                {getSeverityText(selectedChain.severity)}
              </Tag>
            </div>
            <G6Graph
              nodes={selectedChain.nodes}
              edges={selectedChain.edges}
              height={420}
            />
          </div>
        </Col>
      </Row>
    </Card>
  );
};
