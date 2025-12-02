/**
 * 认证相关 Hook
 * 封装 useAuthStore 常用操作
 */
import { useAuthStore } from '../stores';

/**
 * 认证 Hook
 */
export const useAuth = () => {
  const {
    token,
    user,
    isLoading,
    isAuthenticated,
    login,
    logout,
    checkAuth,
    setUser,
  } = useAuthStore();

  return {
    /** 当前 token */
    token,
    /** 当前用户 */
    user,
    /** 登录中状态 */
    isLoading,
    /** 是否已认证 */
    isAuthenticated: isAuthenticated(),
    /** 登录方法 */
    login,
    /** 登出方法 */
    logout,
    /** 检查认证状态 */
    checkAuth,
    /** 更新用户信息 */
    setUser,
  };
};
