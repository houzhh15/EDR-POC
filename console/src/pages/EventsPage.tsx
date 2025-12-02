/**
 * 事件查询页面骨架
 */
import React from 'react';
import { Typography, Table, Input, DatePicker, Space, Card } from 'antd';
import { SearchOutlined } from '@ant-design/icons';

const { Title, Paragraph } = Typography;
const { RangePicker } = DatePicker;

/**
 * 事件查询页面组件
 */
const EventsPage: React.FC = () => {
  // 示例列定义
  const columns = [
    {
      title: '时间',
      dataIndex: 'timestamp',
      key: 'timestamp',
    },
    {
      title: '事件类型',
      dataIndex: 'type',
      key: 'type',
    },
    {
      title: '终端',
      dataIndex: 'endpoint',
      key: 'endpoint',
    },
    {
      title: '进程',
      dataIndex: 'process',
      key: 'process',
    },
    {
      title: '详情',
      dataIndex: 'details',
      key: 'details',
      ellipsis: true,
    },
  ];

  return (
    <div>
      <Title level={2}>事件查询</Title>
      <Paragraph type="secondary">
        此页面用于搜索和分析安全事件，支持全文检索、高级过滤、威胁狩猎等功能。
        功能将在任务 28（事件检索与威胁狩猎）中实现。
      </Paragraph>

      <Card style={{ marginTop: 24 }}>
        <Space size="middle" wrap>
          <Input
            placeholder="搜索事件..."
            prefix={<SearchOutlined />}
            style={{ width: 300 }}
            disabled
          />
          <RangePicker disabled />
        </Space>
      </Card>

      <Table
        columns={columns}
        dataSource={[]}
        locale={{ emptyText: '暂无事件数据' }}
        style={{ marginTop: 16 }}
      />
    </div>
  );
};

export default EventsPage;
