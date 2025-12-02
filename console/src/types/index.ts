/**
 * 用户角色类型
 */
export type UserRole = 'admin' | 'analyst' | 'operator' | 'viewer';

/**
 * 用户信息接口
 */
export interface User {
  id: string;
  username: string;
  displayName: string;
  email?: string;
  role: UserRole;
  avatar?: string;
  createdAt?: string;
  lastLoginAt?: string;
}

/**
 * 通用 API 响应接口
 */
export interface ApiResponse<T> {
  code: number;
  message: string;
  data: T;
}

/**
 * API 错误接口
 */
export interface ApiError {
  code: number;
  message: string;
  details?: Record<string, string>;
}

/**
 * 分页响应接口
 */
export interface PaginatedResponse<T> {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
}

/**
 * 登录请求接口
 */
export interface LoginRequest {
  username: string;
  password: string;
}

/**
 * 登录响应接口
 */
export interface LoginResponse {
  token: string;
  user: User;
  expiresIn: number;
}

// 导出终端类型
export * from './endpoint';
