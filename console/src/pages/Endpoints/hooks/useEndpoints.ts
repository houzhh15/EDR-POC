/**
 * 终端管理 React Query Hooks
 */
import { useQuery } from '@tanstack/react-query';
import { endpointsApi } from '@/api/endpoints';
import type { EndpointListParams } from '@/types/endpoint';

// 查询键工厂
export const endpointKeys = {
  all: ['endpoints'] as const,
  lists: () => [...endpointKeys.all, 'list'] as const,
  list: (params: EndpointListParams) => [...endpointKeys.lists(), params] as const,
  details: () => [...endpointKeys.all, 'detail'] as const,
  detail: (id: string) => [...endpointKeys.details(), id] as const,
  software: (id: string) => [...endpointKeys.detail(id), 'software'] as const,
  changes: (id: string) => [...endpointKeys.detail(id), 'changes'] as const,
};

// 终端列表 Hook
export function useEndpoints(params: EndpointListParams) {
  return useQuery({
    queryKey: endpointKeys.list(params),
    queryFn: () => endpointsApi.list(params),
    staleTime: 30 * 1000, // 30秒内不重新请求
    placeholderData: (prev) => prev, // 保持上一次数据避免闪烁
  });
}

// 终端详情 Hook
export function useEndpointDetail(id: string) {
  return useQuery({
    queryKey: endpointKeys.detail(id),
    queryFn: () => endpointsApi.getById(id),
    enabled: !!id,
  });
}

// 软件清单 Hook
export function useEndpointSoftware(id: string, params?: { page?: number; name?: string }) {
  return useQuery({
    queryKey: [...endpointKeys.software(id), params],
    queryFn: () => endpointsApi.getSoftware(id, params),
    enabled: !!id,
  });
}

// 变更记录 Hook
export function useEndpointChanges(id: string) {
  return useQuery({
    queryKey: endpointKeys.changes(id),
    queryFn: () => endpointsApi.getChanges(id),
    enabled: !!id,
  });
}
