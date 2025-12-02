/**
 * 终端管理 API 封装
 */
import { apiClient } from './client';
import type {
  Endpoint,
  EndpointListParams,
  EndpointListResponse,
  EndpointSoftware,
  EndpointChange,
  IsolateEndpointRequest,
  UpdateEndpointRequest,
  Pagination,
} from '@/types/endpoint';

export const endpointsApi = {
  // 获取终端列表
  list: (params: EndpointListParams): Promise<EndpointListResponse> =>
    apiClient.get('/v1/assets', { params }),

  // 获取终端详情
  getById: (id: string): Promise<{ data: Endpoint }> =>
    apiClient.get(`/v1/assets/${id}`),

  // 更新终端信息
  update: (id: string, data: UpdateEndpointRequest): Promise<{ data: Endpoint }> =>
    apiClient.put(`/v1/assets/${id}`, data),

  // 删除终端
  delete: (id: string): Promise<{ message: string }> =>
    apiClient.delete(`/v1/assets/${id}`),

  // 获取软件清单
  getSoftware: (id: string, params?: { page?: number; page_size?: number; name?: string }): Promise<{
    data: EndpointSoftware[];
    pagination: Pagination;
  }> => apiClient.get(`/v1/assets/${id}/software`, { params }),

  // 获取变更记录
  getChanges: (id: string): Promise<{ data: EndpointChange[] }> =>
    apiClient.get(`/v1/assets/${id}/changes`),

  // 网络隔离
  isolate: (id: string, data: IsolateEndpointRequest): Promise<{ message: string }> =>
    apiClient.post(`/v1/assets/${id}/actions/isolate`, data),

  // 解除隔离
  unisolate: (id: string): Promise<{ message: string }> =>
    apiClient.post(`/v1/assets/${id}/actions/unisolate`),

  // 触发扫描
  scan: (id: string): Promise<{ message: string }> =>
    apiClient.post(`/v1/assets/${id}/actions/scan`),
};
