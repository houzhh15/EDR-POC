/**
 * 终端管理类型定义
 * 与后端 cloud/internal/asset/models.go 和 dto.go 保持一致
 */

// 终端状态枚举
export type EndpointStatus = 'online' | 'offline' | 'unknown';

// 操作系统类型
export type OSType = 'windows' | 'linux' | 'macos';

// 终端实体（与后端 Asset 对应）
export interface Endpoint {
  id: string;
  agent_id: string;
  tenant_id: string;
  hostname: string;
  os_type: OSType;
  os_version: string;
  architecture: string;
  ip_addresses: string[];
  mac_addresses: string[];
  agent_version: string;
  status: EndpointStatus;
  last_seen_at: string | null;
  first_seen_at: string;
  created_at: string;
  updated_at: string;
}

// 列表查询参数
export interface EndpointListParams {
  page?: number;
  page_size?: number;
  status?: EndpointStatus;
  os_type?: OSType;
  hostname?: string;
  ip?: string;
  group_id?: string;
  sort_by?: 'last_seen_at' | 'hostname' | 'created_at' | 'first_seen_at' | 'os_type';
  sort_order?: 'asc' | 'desc';
}

// 分页信息
export interface Pagination {
  page: number;
  page_size: number;
  total: number;
  total_pages: number;
}

// 列表响应
export interface EndpointListResponse {
  data: Endpoint[];
  pagination: Pagination;
}

// 软件清单
export interface EndpointSoftware {
  id: string;
  name: string;
  version: string;
  publisher: string;
  install_date: string;
  install_path: string;
}

// 变更记录
export interface EndpointChange {
  id: string;
  change_type: string;
  field_name: string;
  old_value: string;
  new_value: string;
  changed_at: string;
  changed_by: string;
}

// 隔离请求
export interface IsolateEndpointRequest {
  reason: string;
  keep_management: boolean;
}

// 更新请求
export interface UpdateEndpointRequest {
  hostname?: string;
  os_version?: string;
  architecture?: string;
  ip_addresses?: string[];
  mac_addresses?: string[];
  agent_version?: string;
}
