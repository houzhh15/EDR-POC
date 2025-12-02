/**
 * Dashboard API 封装
 * 提供仪表盘数据获取接口
 * 
 * 🔧 开发模式：当前使用Mock数据（后端API未实现）
 * 🚀 生产模式：修改导入切换到真实API
 */

// Mock模式导入（开发环境）
import * as mockApi from './__mocks__/dashboard';

// 真实API模式导入（生产环境 - 当前已注释）
// import { apiClient } from './client';

import type {
  DashboardStats,
  TimeRange,
  AlertTrendPoint,
  ThreatType,
  TopNItem,
  MitreCell,
  AttackChain,
} from '../types/dashboard';

/**
 * 获取仪表盘统计数据
 * @returns Promise<DashboardStats> 统计数据
 */
async function getDashboardStats(): Promise<DashboardStats> {
  return mockApi.getDashboardStats();
}

/**
 * 获取告警趋势数据
 * @param range 时间范围
 * @returns Promise<AlertTrendPoint[]> 趋势数据数组
 */
async function getAlertTrend(range: TimeRange): Promise<AlertTrendPoint[]> {
  return mockApi.getAlertTrend(range);
}

/**
 * 获取Top威胁数据
 * @param type 威胁类型
 * @param limit 返回数量（默认10）
 * @returns Promise<TopNItem[]> Top N数据数组
 */
async function getTopThreats(
  type: ThreatType,
  limit: number = 10
): Promise<TopNItem[]> {
  return mockApi.getTopThreats(type, limit);
}

/**
 * 获取MITRE ATT&CK覆盖度数据
 * @returns Promise<MitreCell[]> MITRE矩阵数据数组
 */
async function getMitreCoverage(): Promise<MitreCell[]> {
  return mockApi.getMitreCoverage();
}

/**
 * 获取攻击链数据
 * @param limit 返回数量（默认5）
 * @returns Promise<AttackChain[]> 攻击链数组
 */
async function getAttackChains(limit: number = 5): Promise<AttackChain[]> {
  return mockApi.getAttackChains(limit);
}

/**
 * Dashboard API 导出对象
 */
export const dashboardApi = {
  getDashboardStats,
  getAlertTrend,
  getTopThreats,
  getMitreCoverage,
  getAttackChains,
};
