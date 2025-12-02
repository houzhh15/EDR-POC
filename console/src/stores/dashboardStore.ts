/**
 * Dashboard Store
 * 使用Zustand管理仪表盘UI状态
 */
import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { TimeRange } from '../types/dashboard';

/**
 * Dashboard状态接口
 */
interface DashboardState {
  /** 时间范围 */
  timeRange: TimeRange;
  /** 自动刷新开关 */
  autoRefresh: boolean;
  /** 刷新间隔（秒） */
  refreshInterval: number;
  /** 最后更新时间 */
  lastUpdated: Date | null;
  /** 选中的攻击链ID */
  selectedChainId: string | null;
}

/**
 * Dashboard Actions接口
 */
interface DashboardActions {
  /**
   * 设置时间范围
   * @param range - 时间范围
   */
  setTimeRange: (range: TimeRange) => void;
  
  /**
   * 切换自动刷新状态
   */
  toggleAutoRefresh: () => void;
  
  /**
   * 设置刷新间隔
   * @param interval - 刷新间隔（秒）
   */
  setRefreshInterval: (interval: number) => void;
  
  /**
   * 设置最后更新时间
   * @param time - 更新时间
   */
  setLastUpdated: (time: Date) => void;
  
  /**
   * 设置选中的攻击链
   * @param chainId - 攻击链ID
   */
  setSelectedChain: (chainId: string | null) => void;
  
  /**
   * 重置所有筛选条件
   */
  resetFilters: () => void;
}

/**
 * Dashboard Store类型
 */
type DashboardStore = DashboardState & DashboardActions;

/**
 * Dashboard Store
 * 持久化timeRange、autoRefresh、refreshInterval到localStorage
 */
export const useDashboardStore = create<DashboardStore>()(
  persist(
    (set) => ({
      // 初始状态
      timeRange: '24h',
      autoRefresh: true,
      refreshInterval: 60,
      lastUpdated: null,
      selectedChainId: null,

      // Actions
      setTimeRange: (range) =>
        set({ timeRange: range }),

      toggleAutoRefresh: () =>
        set((state) => ({ autoRefresh: !state.autoRefresh })),

      setRefreshInterval: (interval) =>
        set({ refreshInterval: interval }),

      setLastUpdated: (time) =>
        set({ lastUpdated: time }),

      setSelectedChain: (chainId) =>
        set({ selectedChainId: chainId }),

      resetFilters: () =>
        set({
          timeRange: '24h',
          selectedChainId: null,
        }),
    }),
    {
      name: 'dashboard-storage',
      // 只持久化部分状态
      partialize: (state) => ({
        timeRange: state.timeRange,
        autoRefresh: state.autoRefresh,
        refreshInterval: state.refreshInterval,
      }),
    }
  )
);
