// Package asset 提供资产管理相关功能
package asset

import (
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AssetStatus 资产状态常量
type AssetStatus string

const (
	AssetStatusUnknown AssetStatus = "unknown"
	AssetStatusOnline  AssetStatus = "online"
	AssetStatusOffline AssetStatus = "offline"
)

// String 返回状态字符串
func (s AssetStatus) String() string {
	return string(s)
}

// IsValid 验证状态值是否有效
func (s AssetStatus) IsValid() bool {
	switch s {
	case AssetStatusUnknown, AssetStatusOnline, AssetStatusOffline:
		return true
	default:
		return false
	}
}

// GroupType 分组类型常量
type GroupType string

const (
	GroupTypeDepartment GroupType = "department"
	GroupTypeLocation   GroupType = "location"
	GroupTypeCustom     GroupType = "custom"
)

// StringSlice 字符串切片类型，用于 PostgreSQL TEXT[] 存储
type StringSlice []string

// Value 实现 driver.Valuer 接口 (PostgreSQL TEXT[] 格式)
func (s StringSlice) Value() (driver.Value, error) {
	if s == nil || len(s) == 0 {
		return "{}", nil
	}
	// PostgreSQL 数组格式: {"val1","val2"}
	result := "{"
	for i, v := range s {
		if i > 0 {
			result += ","
		}
		// 转义双引号
		escaped := fmt.Sprintf("\"%s\"", v)
		result += escaped
	}
	result += "}"
	return result, nil
}

// Scan 实现 sql.Scanner 接口 (PostgreSQL TEXT[] 格式)
func (s *StringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = []string{}
		return nil
	}

	var str string
	switch v := value.(type) {
	case []byte:
		str = string(v)
	case string:
		str = v
	default:
		return fmt.Errorf("failed to scan StringSlice: expected []byte or string, got %T", value)
	}

	// 处理 PostgreSQL TEXT[] 格式: {val1,val2} 或 {"val1","val2"}
	if str == "{}" || str == "" {
		*s = []string{}
		return nil
	}

	// 移除首尾的 { }
	str = str[1 : len(str)-1]
	if str == "" {
		*s = []string{}
		return nil
	}

	// 简单解析（不处理复杂转义）
	var result []string
	var current string
	inQuote := false
	for i := 0; i < len(str); i++ {
		c := str[i]
		switch c {
		case '"':
			inQuote = !inQuote
		case ',':
			if !inQuote {
				result = append(result, current)
				current = ""
			} else {
				current += string(c)
			}
		default:
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}

	*s = result
	return nil
}

// Asset 资产实体
type Asset struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	AgentID      string         `gorm:"column:agent_id;type:varchar(64);not null" json:"agent_id"`
	TenantID     uuid.UUID      `gorm:"column:tenant_id;type:uuid;not null;index:idx_assets_tenant_status" json:"tenant_id"`
	Hostname     string         `gorm:"column:hostname;type:varchar(255);not null;index:idx_assets_hostname" json:"hostname"`
	OSType       string         `gorm:"column:os_type;type:varchar(32);not null" json:"os_type"`
	OSVersion    string         `gorm:"column:os_version;type:varchar(128)" json:"os_version"`
	Architecture string         `gorm:"column:architecture;type:varchar(16)" json:"architecture"`
	IPAddresses  StringSlice    `gorm:"column:ip_addresses;type:jsonb" json:"ip_addresses"`
	MACAddresses StringSlice    `gorm:"column:mac_addresses;type:jsonb" json:"mac_addresses"`
	AgentVersion string         `gorm:"column:agent_version;type:varchar(32)" json:"agent_version"`
	Status       AssetStatus    `gorm:"column:status;type:varchar(16);not null;index:idx_assets_tenant_status" json:"status"`
	LastSeenAt   *time.Time     `gorm:"column:last_seen_at;index:idx_assets_last_seen" json:"last_seen_at,omitempty"`
	FirstSeenAt  time.Time      `gorm:"column:first_seen_at;not null" json:"first_seen_at"`
	CreatedAt    time.Time      `gorm:"column:created_at;not null" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"column:updated_at;not null" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"column:deleted_at;index" json:"deleted_at,omitempty"`
}

// BeforeCreate GORM hook - 创建前生成 UUID 和时间戳
func (a *Asset) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	now := time.Now()
	if a.FirstSeenAt.IsZero() {
		a.FirstSeenAt = now
	}
	if a.CreatedAt.IsZero() {
		a.CreatedAt = now
	}
	if a.UpdatedAt.IsZero() {
		a.UpdatedAt = now
	}
	if a.Status == "" {
		a.Status = AssetStatusUnknown
	}
	return nil
}

// TableName 指定表名
func (Asset) TableName() string {
	return "assets"
}

// AssetGroup 资产分组
type AssetGroup struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	TenantID    uuid.UUID  `gorm:"column:tenant_id;type:uuid;not null" json:"tenant_id"`
	Name        string     `gorm:"column:name;type:varchar(128);not null" json:"name"`
	Description string     `gorm:"column:description;type:text" json:"description,omitempty"`
	Type        GroupType  `gorm:"column:type;type:varchar(32);not null" json:"type"`
	ParentID    *uuid.UUID `gorm:"column:parent_id;type:uuid" json:"parent_id,omitempty"`
	Path        string     `gorm:"column:path;type:varchar(512);not null;index:idx_groups_path" json:"path"`
	Level       int        `gorm:"column:level;not null" json:"level"`
	CreatedAt   time.Time  `gorm:"column:created_at;not null" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"column:updated_at;not null" json:"updated_at"`

	// 关联关系（不存储到数据库）
	Children []*AssetGroup `gorm:"-" json:"children,omitempty"`
}

// BeforeCreate GORM hook
func (g *AssetGroup) BeforeCreate(tx *gorm.DB) error {
	if g.ID == uuid.Nil {
		g.ID = uuid.New()
	}
	now := time.Now()
	if g.CreatedAt.IsZero() {
		g.CreatedAt = now
	}
	if g.UpdatedAt.IsZero() {
		g.UpdatedAt = now
	}
	if g.Type == "" {
		g.Type = GroupTypeCustom
	}
	return nil
}

// TableName 指定表名
func (AssetGroup) TableName() string {
	return "asset_groups"
}

// AssetGroupMember 资产分组关联（复合主键）
type AssetGroupMember struct {
	AssetID  uuid.UUID `gorm:"column:asset_id;type:uuid;primaryKey" json:"asset_id"`
	GroupID  uuid.UUID `gorm:"column:group_id;type:uuid;primaryKey" json:"group_id"`
	JoinedAt time.Time `gorm:"column:joined_at;not null" json:"joined_at"`
}

// BeforeCreate GORM hook
func (m *AssetGroupMember) BeforeCreate(tx *gorm.DB) error {
	if m.JoinedAt.IsZero() {
		m.JoinedAt = time.Now()
	}
	return nil
}

// TableName 指定表名
func (AssetGroupMember) TableName() string {
	return "asset_group_members"
}

// SoftwareInventory 软件清单
type SoftwareInventory struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	AssetID     uuid.UUID  `gorm:"column:asset_id;type:uuid;not null;index:idx_software_asset" json:"asset_id"`
	Name        string     `gorm:"column:name;type:varchar(255);not null;index:idx_software_name" json:"name"`
	Version     string     `gorm:"column:version;type:varchar(64);not null" json:"version"`
	Publisher   string     `gorm:"column:publisher;type:varchar(255)" json:"publisher,omitempty"`
	InstallDate *time.Time `gorm:"column:install_date" json:"install_date,omitempty"`
	InstallPath string     `gorm:"column:install_path;type:varchar(512)" json:"install_path,omitempty"`
	Size        int64      `gorm:"column:size" json:"size"`
	UpdatedAt   time.Time  `gorm:"column:updated_at;not null" json:"updated_at"`
}

// BeforeCreate GORM hook
func (s *SoftwareInventory) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	if s.UpdatedAt.IsZero() {
		s.UpdatedAt = time.Now()
	}
	return nil
}

// TableName 指定表名
func (SoftwareInventory) TableName() string {
	return "software_inventory"
}

// AssetChangeLog 资产变更日志
type AssetChangeLog struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	AssetID   uuid.UUID `gorm:"column:asset_id;type:uuid;not null;index:idx_change_logs_asset" json:"asset_id"`
	FieldName string    `gorm:"column:field_name;type:varchar(64);not null" json:"field_name"`
	OldValue  string    `gorm:"column:old_value;type:text" json:"old_value"`
	NewValue  string    `gorm:"column:new_value;type:text" json:"new_value"`
	ChangedBy string    `gorm:"column:changed_by;type:varchar(64)" json:"changed_by,omitempty"`
	ChangedAt time.Time `gorm:"column:changed_at;not null;index:idx_change_logs_time" json:"changed_at"`
}

// BeforeCreate GORM hook
func (c *AssetChangeLog) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	if c.ChangedAt.IsZero() {
		c.ChangedAt = time.Now()
	}
	return nil
}

// TableName 指定表名
func (AssetChangeLog) TableName() string {
	return "asset_change_logs"
}
