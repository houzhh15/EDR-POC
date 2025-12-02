package asset

import (
	"time"

	"github.com/google/uuid"
)

// Pagination 分页参数
type Pagination struct {
	Page      int `json:"page" form:"page"`           // 页码，从1开始
	PageSize  int `json:"page_size" form:"page_size"` // 每页数量，最大100
	Total     int `json:"total"`                      // 总记录数
	TotalPage int `json:"total_pages"`                // 总页数
}

// Normalize 规范化分页参数
func (p *Pagination) Normalize() {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PageSize < 1 {
		p.PageSize = 20
	}
	if p.PageSize > 100 {
		p.PageSize = 100
	}
}

// Offset 计算偏移量
func (p *Pagination) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// CalcTotalPages 计算总页数
func (p *Pagination) CalcTotalPages() {
	if p.Total <= 0 || p.PageSize <= 0 {
		p.TotalPage = 0
		return
	}
	p.TotalPage = (p.Total + p.PageSize - 1) / p.PageSize
}

// PaginationResponse 分页响应（用于 API 响应）
type PaginationResponse struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// ==================== Asset DTOs ====================

// RegisterAssetRequest 资产注册请求
type RegisterAssetRequest struct {
	AgentID      string   `json:"agent_id" binding:"required"`
	TenantID     string   `json:"tenant_id" binding:"required,uuid"`
	Hostname     string   `json:"hostname" binding:"required"`
	OSType       string   `json:"os_type" binding:"required,oneof=windows linux macos"`
	OSVersion    string   `json:"os_version"`
	Architecture string   `json:"architecture"`
	IPAddresses  []string `json:"ip_addresses"`
	MACAddresses []string `json:"mac_addresses"`
	AgentVersion string   `json:"agent_version"`
}

// UpdateAssetRequest 更新资产请求
type UpdateAssetRequest struct {
	Hostname     *string  `json:"hostname,omitempty"`
	OSVersion    *string  `json:"os_version,omitempty"`
	Architecture *string  `json:"architecture,omitempty"`
	IPAddresses  []string `json:"ip_addresses,omitempty"`
	MACAddresses []string `json:"mac_addresses,omitempty"`
	AgentVersion *string  `json:"agent_version,omitempty"`
}

// QueryOptions 资产查询选项
type QueryOptions struct {
	Status    string     `form:"status"`     // 状态过滤: online, offline, unknown
	OSType    string     `form:"os_type"`    // 系统类型: windows, linux, macos
	Hostname  string     `form:"hostname"`   // 主机名模糊搜索
	IP        string     `form:"ip"`         // IP地址过滤
	GroupID   *uuid.UUID `form:"group_id"`   // 分组ID过滤
	SortBy    string     `form:"sort_by"`    // 排序字段: last_seen_at, hostname, created_at
	SortOrder string     `form:"sort_order"` // 排序方向: asc, desc
	Pagination
}

// Validate 验证查询参数
func (q *QueryOptions) Validate() error {
	// 验证状态
	if q.Status != "" {
		status := AssetStatus(q.Status)
		if !status.IsValid() {
			return ErrInvalidRequest.WithMessage("invalid status value")
		}
	}

	// 验证排序字段
	validSortFields := map[string]bool{
		"":              true,
		"last_seen_at":  true,
		"hostname":      true,
		"created_at":    true,
		"first_seen_at": true,
		"os_type":       true,
	}
	if !validSortFields[q.SortBy] {
		return ErrInvalidRequest.WithMessage("invalid sort_by field")
	}

	// 验证排序方向
	if q.SortOrder != "" && q.SortOrder != "asc" && q.SortOrder != "desc" {
		return ErrInvalidRequest.WithMessage("sort_order must be 'asc' or 'desc'")
	}

	q.Pagination.Normalize()
	return nil
}

// AssetResponse 资产响应
type AssetResponse struct {
	ID           uuid.UUID   `json:"id"`
	AgentID      string      `json:"agent_id"`
	TenantID     uuid.UUID   `json:"tenant_id"`
	Hostname     string      `json:"hostname"`
	OSType       string      `json:"os_type"`
	OSVersion    string      `json:"os_version"`
	Architecture string      `json:"architecture"`
	IPAddresses  []string    `json:"ip_addresses"`
	MACAddresses []string    `json:"mac_addresses"`
	AgentVersion string      `json:"agent_version"`
	Status       AssetStatus `json:"status"`
	LastSeenAt   *time.Time  `json:"last_seen_at,omitempty"`
	FirstSeenAt  time.Time   `json:"first_seen_at"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

// FromAsset 从 Asset 实体转换
func (r *AssetResponse) FromAsset(a *Asset) {
	r.ID = a.ID
	r.AgentID = a.AgentID
	r.TenantID = a.TenantID
	r.Hostname = a.Hostname
	r.OSType = a.OSType
	r.OSVersion = a.OSVersion
	r.Architecture = a.Architecture
	r.IPAddresses = []string(a.IPAddresses)
	r.MACAddresses = []string(a.MACAddresses)
	r.AgentVersion = a.AgentVersion
	r.Status = a.Status
	r.LastSeenAt = a.LastSeenAt
	r.FirstSeenAt = a.FirstSeenAt
	r.CreatedAt = a.CreatedAt
	r.UpdatedAt = a.UpdatedAt
}

// AssetListResponse 资产列表响应
type AssetListResponse struct {
	Data       []AssetResponse `json:"data"`
	Pagination Pagination      `json:"pagination"`
}

// ==================== Group DTOs ====================

// CreateGroupRequest 创建分组请求
type CreateGroupRequest struct {
	Name        string     `json:"name" binding:"required,min=1,max=128"`
	Description string     `json:"description,omitempty"`
	Type        GroupType  `json:"type,omitempty"`
	ParentID    *uuid.UUID `json:"parent_id,omitempty"`
}

// UpdateGroupRequest 更新分组请求
type UpdateGroupRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

// GroupResponse 分组响应
type GroupResponse struct {
	ID          uuid.UUID        `json:"id"`
	TenantID    uuid.UUID        `json:"tenant_id"`
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Type        GroupType        `json:"type"`
	ParentID    *uuid.UUID       `json:"parent_id,omitempty"`
	Path        string           `json:"path"`
	Level       int              `json:"level"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
	Children    []*GroupResponse `json:"children,omitempty"`
	AssetCount  int              `json:"asset_count,omitempty"`
}

// FromAssetGroup 从 AssetGroup 实体转换
func (r *GroupResponse) FromAssetGroup(g *AssetGroup) {
	r.ID = g.ID
	r.TenantID = g.TenantID
	r.Name = g.Name
	r.Description = g.Description
	r.Type = g.Type
	r.ParentID = g.ParentID
	r.Path = g.Path
	r.Level = g.Level
	r.CreatedAt = g.CreatedAt
	r.UpdatedAt = g.UpdatedAt

	if len(g.Children) > 0 {
		r.Children = make([]*GroupResponse, len(g.Children))
		for i, child := range g.Children {
			r.Children[i] = &GroupResponse{}
			r.Children[i].FromAssetGroup(child)
		}
	}
}

// GroupTreeResponse 分组树响应
type GroupTreeResponse struct {
	Groups []*GroupResponse `json:"groups"`
}

// AddAssetToGroupRequest 添加资产到分组请求
type AddAssetToGroupRequest struct {
	AssetID uuid.UUID `json:"asset_id" binding:"required"`
}

// ==================== Software DTOs ====================

// SoftwareItem 软件项
type SoftwareItem struct {
	Name        string     `json:"name" binding:"required"`
	Version     string     `json:"version" binding:"required"`
	Publisher   string     `json:"publisher,omitempty"`
	InstallDate *time.Time `json:"install_date,omitempty"`
	InstallPath string     `json:"install_path,omitempty"`
	Size        int64      `json:"size,omitempty"`
}

// UpdateSoftwareInventoryRequest 更新软件清单请求
type UpdateSoftwareInventoryRequest struct {
	Software []SoftwareItem `json:"software" binding:"required,dive"`
}

// SoftwareResponse 软件响应
type SoftwareResponse struct {
	ID          uuid.UUID  `json:"id"`
	AssetID     uuid.UUID  `json:"asset_id"`
	Name        string     `json:"name"`
	Version     string     `json:"version"`
	Publisher   string     `json:"publisher,omitempty"`
	InstallDate *time.Time `json:"install_date,omitempty"`
	InstallPath string     `json:"install_path,omitempty"`
	Size        int64      `json:"size"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// FromSoftwareInventory 从 SoftwareInventory 实体转换
func (r *SoftwareResponse) FromSoftwareInventory(s *SoftwareInventory) {
	r.ID = s.ID
	r.AssetID = s.AssetID
	r.Name = s.Name
	r.Version = s.Version
	r.Publisher = s.Publisher
	r.InstallDate = s.InstallDate
	r.InstallPath = s.InstallPath
	r.Size = s.Size
	r.UpdatedAt = s.UpdatedAt
}

// SoftwareListResponse 软件列表响应
type SoftwareListResponse struct {
	Data       []SoftwareResponse `json:"data"`
	Pagination Pagination         `json:"pagination"`
}

// SoftwareQueryOptions 软件查询选项
type SoftwareQueryOptions struct {
	Name      string `form:"name"`      // 软件名模糊搜索
	Publisher string `form:"publisher"` // 发布者过滤
	Pagination
}

// ==================== ChangeLog DTOs ====================

// ChangeLogResponse 变更日志响应
type ChangeLogResponse struct {
	ID        uuid.UUID `json:"id"`
	AssetID   uuid.UUID `json:"asset_id"`
	FieldName string    `json:"field_name"`
	OldValue  string    `json:"old_value"`
	NewValue  string    `json:"new_value"`
	ChangedBy string    `json:"changed_by,omitempty"`
	ChangedAt time.Time `json:"changed_at"`
}

// FromAssetChangeLog 从 AssetChangeLog 实体转换
func (r *ChangeLogResponse) FromAssetChangeLog(c *AssetChangeLog) {
	r.ID = c.ID
	r.AssetID = c.AssetID
	r.FieldName = c.FieldName
	r.OldValue = c.OldValue
	r.NewValue = c.NewValue
	r.ChangedBy = c.ChangedBy
	r.ChangedAt = c.ChangedAt
}

// ChangeLogListResponse 变更日志列表响应
type ChangeLogListResponse struct {
	Data       []ChangeLogResponse `json:"data"`
	Pagination Pagination          `json:"pagination"`
}

// ChangeLogQueryOptions 变更日志查询选项
type ChangeLogQueryOptions struct {
	FieldName string     `form:"field_name"` // 字段名过滤
	StartTime *time.Time `form:"start_time"` // 开始时间
	EndTime   *time.Time `form:"end_time"`   // 结束时间
	Pagination
}

// ==================== API Response Wrappers ====================

// APIResponse 通用 API 响应
type APIResponse struct {
	Data  interface{} `json:"data,omitempty"`
	Error *ErrorInfo  `json:"error,omitempty"`
}

// ErrorInfo 错误信息
type ErrorInfo struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// NewSuccessResponse 创建成功响应
func NewSuccessResponse(data interface{}) *APIResponse {
	return &APIResponse{Data: data}
}

// NewErrorResponse 创建错误响应
func NewErrorResponse(err *AssetError) *APIResponse {
	return &APIResponse{
		Error: &ErrorInfo{
			Code:    err.Code,
			Message: err.Message,
		},
	}
}
