package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserRole 用户角色
type UserRole string

const (
	UserRoleAdmin    UserRole = "admin"
	UserRoleOperator UserRole = "operator"
	UserRoleViewer   UserRole = "viewer"
)

// IsValid 验证角色值
func (r UserRole) IsValid() bool {
	switch r {
	case UserRoleAdmin, UserRoleOperator, UserRoleViewer:
		return true
	}
	return false
}

// String 返回角色字符串
func (r UserRole) String() string {
	return string(r)
}

// UserStatus 用户状态
type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusInactive UserStatus = "inactive"
	UserStatusLocked   UserStatus = "locked"
)

// IsValid 验证状态值
func (s UserStatus) IsValid() bool {
	switch s {
	case UserStatusActive, UserStatusInactive, UserStatusLocked:
		return true
	}
	return false
}

// String 返回状态字符串
func (s UserStatus) String() string {
	return string(s)
}

// User 用户模型
type User struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID     uuid.UUID  `gorm:"type:uuid;not null;index" json:"tenant_id"`
	Username     string     `gorm:"type:varchar(100);not null" json:"username"`
	Email        string     `gorm:"type:varchar(255);not null" json:"email"`
	PasswordHash string     `gorm:"type:varchar(255);not null" json:"-"` // 不序列化到 JSON
	Role         UserRole   `gorm:"type:varchar(20);not null;default:'viewer'" json:"role"`
	Status       UserStatus `gorm:"type:varchar(20);not null;default:'active'" json:"status"`
	DisplayName  string     `gorm:"type:varchar(255)" json:"display_name"`
	Phone        string     `gorm:"type:varchar(50)" json:"phone"`
	LastLoginAt  *time.Time `gorm:"type:timestamptz" json:"last_login_at"`
	LastLoginIP  string     `gorm:"type:varchar(45)" json:"last_login_ip"` // 支持 IPv6
	LoginCount   int        `gorm:"not null;default:0" json:"login_count"`
	CreatedAt    time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt    *time.Time `gorm:"type:timestamptz;index" json:"deleted_at,omitempty"`

	// Relations
	Tenant *Tenant `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}

// BeforeCreate GORM hook
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	now := time.Now().UTC()
	u.CreatedAt = now
	u.UpdatedAt = now
	return nil
}

// BeforeUpdate GORM hook
func (u *User) BeforeUpdate(tx *gorm.DB) error {
	u.UpdatedAt = time.Now().UTC()
	return nil
}

// IsDeleted 检查用户是否已被软删除
func (u *User) IsDeleted() bool {
	return u.DeletedAt != nil
}
