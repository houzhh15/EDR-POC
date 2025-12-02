/**
 * 路由守卫组件
 * 未认证用户重定向到登录页
 */
import { useState, useEffect } from 'react';
import { Navigate, Outlet, useLocation } from 'react-router-dom';
import { Spin } from 'antd';
import { useAuthStore } from '../stores';

/**
 * 受保护路由组件
 */
const ProtectedRoute: React.FC = () => {
  const location = useLocation();
  const { isAuthenticated, checkAuth } = useAuthStore();
  const [checking, setChecking] = useState(true);

  useEffect(() => {
    // 如果未认证，尝试从 localStorage 恢复
    if (!isAuthenticated()) {
      checkAuth();
    }
    setChecking(false);
  }, [isAuthenticated, checkAuth]);

  // 检查中显示加载状态
  if (checking) {
    return (
      <div style={{ 
        display: 'flex', 
        justifyContent: 'center', 
        alignItems: 'center', 
        height: '100vh' 
      }}>
        <Spin size="large" />
      </div>
    );
  }

  // 未认证重定向到登录页
  if (!isAuthenticated()) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }

  // 已认证渲染子路由
  return <Outlet />;
};

export default ProtectedRoute;
