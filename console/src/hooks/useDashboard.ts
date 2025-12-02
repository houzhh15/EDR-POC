/**
 * Dashboard React Query Hooks
 * 封装仪表盘数据获取逻辑
 */
import { useQuery, useQueryClient, UseQueryResult } from '@tanstack/react-query';
import { dashboardApi } from '../api/dashboard';
import { useDashboardStore } from '../stores/dashboardStore';
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
 * @returns UseQueryResult<DashboardStats>
 */
export function useDashboardStats(): UseQueryResult<DashboardStats> {
  const { autoRefresh, refreshInterval } = useDashboardStore();

  return useQuery({
    queryKey: ['dashboard', 'stats'],
    queryFn: dashboardApi.getDashboardStats,
    staleTime: 30000,
    refetchInterval: autoRefresh ? refreshInterval * 1000 : false,
    enabled: true,
    retry: 3,
    refetchOnWindowFocus: true,
  });
}

/**
 * 获取告警趋势数据
 * @param range - 时间范围
 * @returns UseQueryResult<AlertTrendPoint[]>
 */
export function useAlertTrend(
  range: TimeRange
): UseQueryResult<AlertTrendPoint[]> {
  const { autoRefresh, refreshInterval } = useDashboardStore();

  return useQuery({
    queryKey: ['dashboard', 'alert-trend', range],
    queryFn: () => dashboardApi.getAlertTrend(range),
    staleTime: 60000,
    refetchInterval: autoRefresh ? refreshInterval * 1000 : false,
    enabled: true,
    retry: 3,
    refetchOnWindowFocus: true,
  });
}

/**
 * 获取Top威胁数据
 * @param type - 威胁类型
 * @param limit - 返回数量，默认10
 * @returns UseQueryResult<TopNItem[]>
 */
export function useTopThreats(
  type: ThreatType,
  limit: number = 10
): UseQueryResult<TopNItem[]> {
  const { autoRefresh, refreshInterval } = useDashboardStore();

  return useQuery({
    queryKey: ['dashboard', 'top-threats', type, limit],
    queryFn: () => dashboardApi.getTopThreats(type, limit),
    staleTime: 60000,
    refetchInterval: autoRefresh ? refreshInterval * 1000 : false,
    enabled: true,
    retry: 3,
    refetchOnWindowFocus: true,
  });
}

/**
 * 获取MITRE ATT&CK覆盖度数据
 * @returns UseQueryResult<MitreCell[]>
 */
export function useMitreCoverage(): UseQueryResult<MitreCell[]> {
  const { autoRefresh, refreshInterval } = useDashboardStore();

  return useQuery({
    queryKey: ['dashboard', 'mitre-coverage'],
    queryFn: dashboardApi.getMitreCoverage,
    staleTime: 300000,
    refetchInterval: autoRefresh ? Math.max(refreshInterval * 5, 300) * 1000 : false,
    enabled: true,
    retry: 3,
    refetchOnWindowFocus: true,
  });
}

/**
 * 获取攻击链数据
 * @param limit - 返回链数，默认5
 * @returns UseQueryResult<AttackChain[]>
 */
export function useAttackChains(
  limit: number = 5
): UseQueryResult<AttackChain[]> {
  const { autoRefresh, refreshInterval } = useDashboardStore();

  return useQuery({
    queryKey: ['dashboard', 'attack-chains', limit],
    queryFn: () => dashboardApi.getAttackChains(limit),
    staleTime: 60000,
    refetchInterval: autoRefresh ? refreshInterval * 1000 : false,
    enabled: true,
    retry: 3,
    refetchOnWindowFocus: true,
  });
}

/**
 * Dashboard刷新控制Hook
 * @returns 刷新函数
 */
export function useDashboardRefresh() {
  const queryClient = useQueryClient();
  const { setLastUpdated } = useDashboardStore();

  /**
   * 手动刷新所有仪表盘数据
   */
  const refresh = () => {
    queryClient.invalidateQueries({ queryKey: ['dashboard'] });
    setLastUpdated(new Date());
  };

  return { refresh };
}
