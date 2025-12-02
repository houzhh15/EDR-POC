/**
 * 应用路由配置
 */
import { Routes, Route } from 'react-router-dom';
import ProtectedRoute from './components/ProtectedRoute';
import { MainLayout } from './components/Layout';
import {
  LoginPage,
  DashboardPage,
  AlertsPage,
  EndpointsPage,
  EndpointDetailPage,
  EventsPage,
  PoliciesPage,
  SettingsPage,
  NotFoundPage,
} from './pages';

/**
 * 应用路由器组件
 * 注意: BrowserRouter 已在 main.tsx 中配置，此处只需 Routes
 */
const AppRouter: React.FC = () => {
  return (
    <Routes>
      {/* 公开路由 - 登录页 */}
      <Route path="/login" element={<LoginPage />} />

      {/* 受保护路由 */}
      <Route element={<ProtectedRoute />}>
        {/* 主布局 */}
        <Route element={<MainLayout />}>
          {/* 首页 - 仪表盘 */}
          <Route index element={<DashboardPage />} />
          {/* 告警中心 */}
          <Route path="alerts" element={<AlertsPage />} />
          {/* 终端管理 */}
          <Route path="endpoints">
            <Route index element={<EndpointsPage />} />
            <Route path=":id" element={<EndpointDetailPage />} />
          </Route>
          {/* 事件查询 */}
          <Route path="events" element={<EventsPage />} />
          {/* 策略管理 */}
          <Route path="policies" element={<PoliciesPage />} />
          {/* 系统设置 */}
          <Route path="settings" element={<SettingsPage />} />
        </Route>
      </Route>

      {/* 404 页面 */}
      <Route path="*" element={<NotFoundPage />} />
    </Routes>
  );
};

export default AppRouter;
