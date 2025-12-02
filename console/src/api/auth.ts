/**
 * 认证 API 模块
 */
import { apiClient } from './client';
import type { LoginRequest, LoginResponse } from './types';

/**
 * 用户登录
 */
export const login = async (credentials: LoginRequest): Promise<LoginResponse> => {
  return apiClient.post<LoginRequest, LoginResponse>('/v1/auth/login', credentials);
};

/**
 * 认证 API 导出
 */
export const authApi = {
  login,
};
