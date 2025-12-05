package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PolicyType 策略类型
type PolicyType string

const (
	PolicyTypeDetection  PolicyType = "detection"  // 检测规则
	PolicyTypeResponse   PolicyType = "response"   // 响应策略
	PolicyTypeCompliance PolicyType = "compliance" // 合规检查
)

// IsValid 验证类型值
func (t PolicyType) IsValid() bool {
	switch t {
	case PolicyTypeDetection, PolicyTypeResponse, PolicyTypeCompliance:
		return true
	}
	return false
}

// String 返回类型字符串
func (t PolicyType) String() string {
	return string(t)
}

// Policy 策略模型
type Policy struct {
	ID          uuid.UUID    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID    uuid.UUID    `gorm:"type:uuid;not null;index" json:"tenant_id"`
	Name        string       `gorm:"type:varchar(255);not null" json:"name"`
	Description string       `gorm:"type:text" json:"description"`
	Type        PolicyType   `gorm:"type:varchar(50);not null" json:"type"`
	Priority    int          `gorm:"not null;default:50" json:"priority"` // 1-100，数值越大优先级越高
	Enabled     bool         `gorm:"not null;default:true" json:"enabled"`
	Config      PolicyConfig `gorm:"type:jsonb;not null;default:'{}'" json:"config"`
	Version     int          `gorm:"not null;default:1" json:"version"`
	CreatedBy   *uuid.UUID   `gorm:"type:uuid" json:"created_by"`
	UpdatedBy   *uuid.UUID   `gorm:"type:uuid" json:"updated_by"`
	CreatedAt   time.Time    `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt   time.Time    `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt   *time.Time   `gorm:"type:timestamptz;index" json:"deleted_at,omitempty"`

	// Relations
	Tenant  *Tenant `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Creator *User   `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
}

// TableName 指定表名
func (Policy) TableName() string {
	return "policies"
}

// BeforeCreate GORM hook
func (p *Policy) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	now := time.Now().UTC()
	p.CreatedAt = now
	p.UpdatedAt = now
	return nil
}

// BeforeUpdate GORM hook
func (p *Policy) BeforeUpdate(tx *gorm.DB) error {
	p.UpdatedAt = time.Now().UTC()
	return nil
}

// IsDeleted 检查策略是否已被软删除
func (p *Policy) IsDeleted() bool {
	return p.DeletedAt != nil
}

// PolicyConfig 策略配置
type PolicyConfig struct {
	Rules      []RuleConfig    `json:"rules,omitempty"`
	Actions    []ActionConfig  `json:"actions,omitempty"`
	Schedule   *ScheduleConfig `json:"schedule,omitempty"`
	Thresholds map[string]int  `json:"thresholds,omitempty"`
}

// Value 实现 driver.Valuer
func (c PolicyConfig) Value() (driver.Value, error) {
	return json.Marshal(c)
}

// Scan 实现 sql.Scanner
func (c *PolicyConfig) Scan(value interface{}) error {
	if value == nil {
		*c = PolicyConfig{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, c)
}

// RuleConfig 规则配置
type RuleConfig struct {
	Name      string            `json:"name"`
	Condition string            `json:"condition"` // CEL 表达式
	Severity  string            `json:"severity"`
	Tags      []string          `json:"tags"`
	Metadata  map[string]string `json:"metadata"`
}

// ActionConfig 动作配置
type ActionConfig struct {
	Type   string                 `json:"type"` // alert, block, isolate
	Params map[string]interface{} `json:"params"`
}

// ScheduleConfig 调度配置
type ScheduleConfig struct {
	Enabled  bool   `json:"enabled"`
	CronExpr string `json:"cron_expr"`
	Timezone string `json:"timezone"`
}
