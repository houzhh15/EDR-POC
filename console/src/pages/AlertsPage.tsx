/**
 * 告警管理页面骨架
 */
import React from 'react';
import { Typography, Table, Tag, Space } from 'antd';

const { Title, Paragraph } = Typography;

/**
 * 告警管理页面组件
 */
const AlertsPage: React.FC = () => {
  // 示例列定义
  const columns = [
    {
      title: '告警ID',
      dataIndex: 'id',
      key: 'id',
    },
    {
      title: '严重程度',
      dataIndex: 'severity',
      key: 'severity',
      render: () => <Tag color="red">高危</Tag>,
    },
    {
      title: '告警类型',
      dataIndex: 'type',
      key: 'type',
    },
    {
      title: '受影响终端',
      dataIndex: 'endpoint',
      key: 'endpoint',
    },
    {
      title: '发生时间',
      dataIndex: 'time',
      key: 'time',
    },
    {
      title: '操作',
      key: 'action',
      render: () => (
        <Space size="middle">
          <a>查看</a>
          <a>处置</a>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <Title level={2}>告警中心</Title>
      <Paragraph type="secondary">
        此页面用于管理和处置安全告警，支持告警查询、详情查看、处置操作等功能。
        功能将在任务 26（告警管理页面）中实现。
      </Paragraph>

      <Table
        columns={columns}
        dataSource={[]}
        locale={{ emptyText: '暂无告警数据' }}
        style={{ marginTop: 24 }}
      />
    </div>
  );
};

export default AlertsPage;
