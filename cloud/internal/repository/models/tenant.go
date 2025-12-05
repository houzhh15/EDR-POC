package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TenantStatus 租户状态
type TenantStatus string

const (
	TenantStatusActive    TenantStatus = "active"
	TenantStatusSuspended TenantStatus = "suspended"
	TenantStatusDeleted   TenantStatus = "deleted"
)

// IsValid 验证状态值
func (s TenantStatus) IsValid() bool {
	switch s {
	case TenantStatusActive, TenantStatusSuspended, TenantStatusDeleted:
		return true
	}
	return false
}

// String 返回状态字符串
func (s TenantStatus) String() string {
	return string(s)
}

// Tenant 租户模型
type Tenant struct {
	ID              uuid.UUID    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name            string       `gorm:"type:varchar(100);uniqueIndex;not null" json:"name"`
	DisplayName     string       `gorm:"type:varchar(255);not null" json:"display_name"`
	Status          TenantStatus `gorm:"type:varchar(20);not null;default:'active'" json:"status"`
	MaxAgents       int          `gorm:"not null;default:100" json:"max_agents"`
	MaxEventsPerDay int64        `gorm:"not null;default:10000000" json:"max_events_per_day"`
	ContactEmail    string       `gorm:"type:varchar(255)" json:"contact_email"`
	ContactPhone    string       `gorm:"type:varchar(50)" json:"contact_phone"`
	Settings        JSONMap      `gorm:"type:jsonb;default:'{}'" json:"settings"`
	CreatedAt       time.Time    `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt       time.Time    `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`

	// Relations
	Users    []User   `gorm:"foreignKey:TenantID" json:"users,omitempty"`
	Policies []Policy `gorm:"foreignKey:TenantID" json:"policies,omitempty"`
}

// TableName 指定表名
func (Tenant) TableName() string {
	return "tenants"
}

// BeforeCreate GORM hook
func (t *Tenant) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	now := time.Now().UTC()
	t.CreatedAt = now
	t.UpdatedAt = now
	return nil
}

// BeforeUpdate GORM hook
func (t *Tenant) BeforeUpdate(tx *gorm.DB) error {
	t.UpdatedAt = time.Now().UTC()
	return nil
}
