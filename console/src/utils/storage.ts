/**
 * 本地存储工具模块
 */

/** Token 存储键名 */
export const TOKEN_KEY = 'edr_token';

/**
 * 获取存储的 token
 */
export const getToken = (): string | null => {
  return localStorage.getItem(TOKEN_KEY);
};

/**
 * 存储 token
 */
export const setToken = (token: string): void => {
  localStorage.setItem(TOKEN_KEY, token);
};

/**
 * 移除 token
 */
export const removeToken = (): void => {
  localStorage.removeItem(TOKEN_KEY);
};

/**
 * 泛型获取任意值
 */
export const getItem = <T>(key: string): T | null => {
  const value = localStorage.getItem(key);
  if (value === null) {
    return null;
  }
  try {
    return JSON.parse(value) as T;
  } catch {
    return null;
  }
};

/**
 * 存储任意值
 */
export const setItem = <T>(key: string, value: T): void => {
  localStorage.setItem(key, JSON.stringify(value));
};

/**
 * 移除任意值
 */
export const removeItem = (key: string): void => {
  localStorage.removeItem(key);
};
