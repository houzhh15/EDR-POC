/**
 * 策略管理页面骨架
 */
import React from 'react';
import { Typography, Table, Tag, Space, Button } from 'antd';
import { PlusOutlined } from '@ant-design/icons';

const { Title, Paragraph } = Typography;

/**
 * 策略管理页面组件
 */
const PoliciesPage: React.FC = () => {
  // 示例列定义
  const columns = [
    {
      title: '策略名称',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      render: () => <Tag color="blue">检测规则</Tag>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: () => <Tag color="green">已启用</Tag>,
    },
    {
      title: '应用范围',
      dataIndex: 'scope',
      key: 'scope',
    },
    {
      title: '更新时间',
      dataIndex: 'updatedAt',
      key: 'updatedAt',
    },
    {
      title: '操作',
      key: 'action',
      render: () => (
        <Space size="middle">
          <a>编辑</a>
          <a>禁用</a>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div>
          <Title level={2}>策略管理</Title>
          <Paragraph type="secondary">
            此页面用于管理检测策略和响应规则，支持策略配置、规则编辑、下发管理等功能。
            功能将在任务 29（策略管理页面）中实现。
          </Paragraph>
        </div>
        <Button type="primary" icon={<PlusOutlined />} disabled>
          创建策略
        </Button>
      </div>

      <Table
        columns={columns}
        dataSource={[]}
        locale={{ emptyText: '暂无策略数据' }}
        style={{ marginTop: 24 }}
      />
    </div>
  );
};

export default PoliciesPage;
