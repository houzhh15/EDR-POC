/**
 * 认证状态管理 Store
 */
import { create } from 'zustand';
import { devtools } from 'zustand/middleware';
import type { User } from '../types';
import { authApi } from '../api';
import { getToken, setToken, removeToken } from '../utils/storage';

/**
 * 认证状态接口
 */
interface AuthState {
  /** JWT Token */
  token: string | null;
  /** 当前用户信息 */
  user: User | null;
  /** 登录中状态 */
  isLoading: boolean;
}

/**
 * 认证操作接口
 */
interface AuthActions {
  /** 是否已认证（计算属性） */
  isAuthenticated: () => boolean;
  /** 用户登录 */
  login: (username: string, password: string) => Promise<void>;
  /** 用户登出 */
  logout: () => void;
  /** 检查认证状态，从 localStorage 恢复 */
  checkAuth: () => boolean;
  /** 更新用户信息 */
  setUser: (user: User | null) => void;
}

type AuthStore = AuthState & AuthActions;

/**
 * 初始状态
 */
const initialState: AuthState = {
  token: null,
  user: null,
  isLoading: false,
};

/**
 * 认证 Store
 */
export const useAuthStore = create<AuthStore>()(
  devtools(
    (set, get) => ({
      ...initialState,

      /**
       * 计算属性：是否已认证
       */
      isAuthenticated: () => {
        return get().token !== null;
      },

      /**
       * 用户登录
       */
      login: async (username: string, password: string) => {
        set({ isLoading: true });
        try {
          const response = await authApi.login({ username, password });
          // 存储 token 到 localStorage
          setToken(response.token);
          // 更新状态
          set({
            token: response.token,
            user: response.user,
            isLoading: false,
          });
        } catch (error) {
          set({ isLoading: false });
          throw error;
        }
      },

      /**
       * 用户登出
       */
      logout: () => {
        // 清除 localStorage
        removeToken();
        // 清除状态
        set({
          token: null,
          user: null,
        });
      },

      /**
       * 检查认证状态，从 localStorage 恢复 token
       */
      checkAuth: () => {
        const token = getToken();
        if (token) {
          set({ token });
          return true;
        }
        return false;
      },

      /**
       * 更新用户信息
       */
      setUser: (user: User | null) => {
        set({ user });
      },
    }),
    { name: 'auth' }
  )
);
