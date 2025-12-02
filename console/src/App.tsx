/**
 * 应用入口组件
 */
import { ConfigProvider } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import AppRouter from './router';

/**
 * 主题配置
 */
const themeConfig = {
  token: {
    colorPrimary: '#1890ff',
    borderRadius: 4,
  },
};

/**
 * 应用组件
 */
function App() {
  return (
    <ConfigProvider locale={zhCN} theme={themeConfig}>
      <AppRouter />
    </ConfigProvider>
  );
}

export default App;

