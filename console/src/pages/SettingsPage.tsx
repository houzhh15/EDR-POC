/**
 * 系统设置页面骨架
 */
import React from 'react';
import { Typography, Card, Tabs, Descriptions, Switch, Button } from 'antd';
import type { TabsProps } from 'antd';

const { Title, Paragraph } = Typography;

/**
 * 系统设置页面组件
 */
const SettingsPage: React.FC = () => {
  const tabItems: TabsProps['items'] = [
    {
      key: 'general',
      label: '通用设置',
      children: (
        <Card>
          <Descriptions column={1}>
            <Descriptions.Item label="系统名称">EDR Console</Descriptions.Item>
            <Descriptions.Item label="版本">1.0.0</Descriptions.Item>
            <Descriptions.Item label="启用审计日志">
              <Switch defaultChecked disabled />
            </Descriptions.Item>
          </Descriptions>
        </Card>
      ),
    },
    {
      key: 'users',
      label: '用户管理',
      children: (
        <Card>
          <Paragraph type="secondary">
            用户和权限管理功能将在任务 30（用户权限管理页面）中实现。
          </Paragraph>
          <Button type="primary" disabled>
            添加用户
          </Button>
        </Card>
      ),
    },
    {
      key: 'integration',
      label: '集成配置',
      children: (
        <Card>
          <Paragraph type="secondary">
            威胁情报、SIEM、SOAR 等外部系统集成配置将在后续版本中实现。
          </Paragraph>
        </Card>
      ),
    },
  ];

  return (
    <div>
      <Title level={2}>系统设置</Title>
      <Paragraph type="secondary">
        此页面用于配置系统参数、管理用户权限、设置集成等功能。
        功能将在任务 30（用户权限管理页面）及后续任务中实现。
      </Paragraph>

      <Tabs items={tabItems} style={{ marginTop: 24 }} />
    </div>
  );
};

export default SettingsPage;
