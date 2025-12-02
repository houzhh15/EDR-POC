import React, { useState } from 'react';
import { Card, List, Badge, Progress, Tag, Drawer, Empty, Spin, Typography } from 'antd';
import { useTopThreats } from '../../hooks/useDashboard';
import type { ThreatType, Severity } from '../../types/dashboard';

const { Text } = Typography;

export interface TopNListProps {
  /** 威胁类型 */
  type: ThreatType;
  /** 显示数量 */
  limit?: number;
  /** 列表标题 */
  title?: string;
}

/**
 * Top N威胁列表组件
 * 展示进程/IP/域名的Top威胁排行
 */
export const TopNList: React.FC<TopNListProps> = ({
  type,
  limit = 10,
  title,
}) => {
  const { data, isLoading, error } = useTopThreats(type, limit);
  const [selectedItem, setSelectedItem] = useState<any>(null);
  const [drawerVisible, setDrawerVisible] = useState(false);

  // 默认标题
  const defaultTitles: Record<ThreatType, string> = {
    process: 'Top 进程',
    ip: 'Top IP地址',
    domain: 'Top 域名',
  };

  // 排名徽章颜色
  const getRankBadgeColor = (rank: number) => {
    if (rank === 1) return '#ffd700'; // 金色
    if (rank === 2) return '#c0c0c0'; // 银色
    if (rank === 3) return '#cd7f32'; // 铜色
    return '#d9d9d9';
  };

  // 风险等级Tag颜色
  const getSeverityColor = (severity: Severity) => {
    const colorMap: Record<Severity, string> = {
      critical: 'red',
      high: 'orange',
      medium: 'gold',
      low: 'blue',
    };
    return colorMap[severity];
  };

  // 风险等级中文
  const getSeverityText = (severity: Severity) => {
    const textMap: Record<Severity, string> = {
      critical: '严重',
      high: '高危',
      medium: '中危',
      low: '低危',
    };
    return textMap[severity];
  };

  // 点击列表项
  const handleItemClick = (item: any) => {
    setSelectedItem(item);
    setDrawerVisible(true);
  };

  if (isLoading) {
    return (
      <Card title={title || defaultTitles[type]} style={{ height: '100%' }}>
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

  if (error || !data) {
    return (
      <Card title={title || defaultTitles[type]} style={{ height: '100%' }}>
        <Empty description="加载失败" />
      </Card>
    );
  }

  // 计算最大值用于Progress比例
  const maxCount = data[0]?.count || 1;

  return (
    <>
      <Card 
        title={title || defaultTitles[type]} 
        style={{ height: '100%', maxHeight: '600px' }}
        bodyStyle={{ maxHeight: '520px', overflow: 'auto' }}
      >
        <List
          dataSource={data}
          renderItem={(item) => (
            <List.Item
              key={item.rank}
              style={{ cursor: 'pointer', padding: '12px 0' }}
              onClick={() => handleItemClick(item)}
            >
              <List.Item.Meta
                avatar={
                  <Badge
                    count={item.rank}
                    style={{
                      backgroundColor: getRankBadgeColor(item.rank),
                      color: item.rank <= 3 ? '#000' : '#fff',
                      fontWeight: 'bold',
                      fontSize: '12px',
                      minWidth: '24px',
                      height: '24px',
                      lineHeight: '24px',
                    }}
                  />
                }
                title={
                  <div
                    style={{
                      display: 'flex',
                      justifyContent: 'space-between',
                      alignItems: 'center',
                    }}
                  >
                    <Text strong ellipsis style={{ maxWidth: '60%' }}>
                      {item.name}
                    </Text>
                    <Tag color={getSeverityColor(item.risk_level)}>
                      {getSeverityText(item.risk_level)}
                    </Tag>
                  </div>
                }
                description={
                  <div style={{ marginTop: '8px' }}>
                    <div
                      style={{
                        display: 'flex',
                        justifyContent: 'space-between',
                        marginBottom: '4px',
                      }}
                    >
                      <Text type="secondary" style={{ fontSize: '12px' }}>
                        命中次数
                      </Text>
                      <Text style={{ fontSize: '12px' }}>{item.count}</Text>
                    </div>
                    <Progress
                      percent={(item.count / maxCount) * 100}
                      showInfo={false}
                      strokeColor={
                        item.risk_level === 'critical' || item.risk_level === 'high'
                          ? '#ff4d4f'
                          : '#1890ff'
                      }
                      size="small"
                    />
                  </div>
                }
              />
            </List.Item>
          )}
        />
      </Card>

      {/* 详情抽屉 */}
      <Drawer
        title="威胁详情"
        placement="right"
        onClose={() => setDrawerVisible(false)}
        open={drawerVisible}
        width={480}
      >
        {selectedItem && (
          <div>
            <div style={{ marginBottom: '16px' }}>
              <Text type="secondary">名称</Text>
              <div>
                <Text strong style={{ fontSize: '16px' }}>
                  {selectedItem.name}
                </Text>
              </div>
            </div>

            <div style={{ marginBottom: '16px' }}>
              <Text type="secondary">排名</Text>
              <div>
                <Badge
                  count={selectedItem.rank}
                  style={{
                    backgroundColor: getRankBadgeColor(selectedItem.rank),
                    color: selectedItem.rank <= 3 ? '#000' : '#fff',
                  }}
                />
              </div>
            </div>

            <div style={{ marginBottom: '16px' }}>
              <Text type="secondary">命中次数</Text>
              <div>
                <Text strong>{selectedItem.count}</Text>
              </div>
            </div>

            <div style={{ marginBottom: '16px' }}>
              <Text type="secondary">风险等级</Text>
              <div>
                <Tag color={getSeverityColor(selectedItem.risk_level)}>
                  {getSeverityText(selectedItem.risk_level)}
                </Tag>
              </div>
            </div>

            {selectedItem.metadata && (
              <div>
                <Text type="secondary">其他信息</Text>
                <div style={{ marginTop: '8px' }}>
                  <pre
                    style={{
                      background: '#f5f5f5',
                      padding: '12px',
                      borderRadius: '4px',
                      fontSize: '12px',
                    }}
                  >
                    {JSON.stringify(selectedItem.metadata, null, 2)}
                  </pre>
                </div>
              </div>
            )}
          </div>
        )}
      </Drawer>
    </>
  );
};
