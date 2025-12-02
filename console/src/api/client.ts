/**
 * Axios HTTP 客户端配置
 */
import axios, { AxiosError, InternalAxiosRequestConfig } from 'axios';
import { message } from 'antd';
import { getToken, removeToken } from '../utils/storage';

/**
 * 创建 Axios 实例
 */
export const apiClient = axios.create({
  baseURL: '/api',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
});

/**
 * 请求拦截器
 * 从 localStorage 获取 token 并设置 Authorization Header
 */
apiClient.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    const token = getToken();
    if (token && config.headers) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

/**
 * 响应拦截器
 * 统一处理错误响应
 */
apiClient.interceptors.response.use(
  (response) => {
    // 成功时返回 response.data
    return response.data;
  },
  (error: AxiosError) => {
    if (error.response) {
      const status = error.response.status;

      switch (status) {
        case 401:
          // 认证失败，清除 token 并跳转登录页
          removeToken();
          window.location.href = '/login';
          break;
        case 403:
          message.error('权限不足');
          break;
        case 500:
        case 502:
        case 503:
        case 504:
          message.error('服务器错误，请稍后重试');
          break;
        default:
          // 其他错误，显示后端返回的错误信息
          {
            const data = error.response.data as { message?: string };
            message.error(data?.message || '请求失败');
          }
      }
    } else if (error.request) {
      // 网络错误
      message.error('网络连接失败，请检查网络');
    } else {
      message.error('请求错误');
    }

    return Promise.reject(error);
  }
);
