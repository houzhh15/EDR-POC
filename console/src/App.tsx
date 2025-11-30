import { Layout, Menu, theme } from 'antd';
import {
  DashboardOutlined,
  AlertOutlined,
  DesktopOutlined,
  FileSearchOutlined,
  SettingOutlined,
} from '@ant-design/icons';

const { Header, Content, Sider } = Layout;

const menuItems = [
  {
    key: 'dashboard',
    icon: <DashboardOutlined />,
    label: '仪表盘',
  },
  {
    key: 'alerts',
    icon: <AlertOutlined />,
    label: '告警中心',
  },
  {
    key: 'assets',
    icon: <DesktopOutlined />,
    label: '资产管理',
  },
  {
    key: 'events',
    icon: <FileSearchOutlined />,
    label: '事件查询',
  },
  {
    key: 'settings',
    icon: <SettingOutlined />,
    label: '系统设置',
  },
];

function App() {
  const {
    token: { colorBgContainer },
  } = theme.useToken();

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Header
        style={{
          display: 'flex',
          alignItems: 'center',
          background: '#001529',
          padding: '0 24px',
        }}
      >
        <div
          style={{
            color: '#fff',
            fontSize: '18px',
            fontWeight: 'bold',
          }}
        >
          EDR Console
        </div>
      </Header>
      <Layout>
        <Sider width={200} style={{ background: colorBgContainer }}>
          <Menu
            mode="inline"
            defaultSelectedKeys={['dashboard']}
            style={{ height: '100%', borderRight: 0 }}
            items={menuItems}
          />
        </Sider>
        <Layout style={{ padding: '24px' }}>
          <Content
            style={{
              padding: 24,
              margin: 0,
              minHeight: 280,
              background: colorBgContainer,
              borderRadius: 8,
            }}
          >
            <h1>欢迎使用 EDR Console</h1>
            <p>终端检测与响应平台管理控制台</p>
          </Content>
        </Layout>
      </Layout>
    </Layout>
  );
}

export default App;
