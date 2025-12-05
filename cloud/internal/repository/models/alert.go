package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AlertSeverity 告警严重程度
type AlertSeverity string

const (
	AlertSeverityCritical AlertSeverity = "critical"
	AlertSeverityHigh     AlertSeverity = "high"
	AlertSeverityMedium   AlertSeverity = "medium"
	AlertSeverityLow      AlertSeverity = "low"
	AlertSeverityInfo     AlertSeverity = "info"
)

// IsValid 验证严重程度值
func (s AlertSeverity) IsValid() bool {
	switch s {
	case AlertSeverityCritical, AlertSeverityHigh, AlertSeverityMedium, AlertSeverityLow, AlertSeverityInfo:
		return true
	}
	return false
}

// String 返回严重程度字符串
func (s AlertSeverity) String() string {
	return string(s)
}

// AlertStatus 告警状态
type AlertStatus string

const (
	AlertStatusOpen          AlertStatus = "open"
	AlertStatusAcknowledged  AlertStatus = "acknowledged"
	AlertStatusInProgress    AlertStatus = "in_progress"
	AlertStatusResolved      AlertStatus = "resolved"
	AlertStatusFalsePositive AlertStatus = "false_positive"
)

// IsValid 验证状态值
func (s AlertStatus) IsValid() bool {
	switch s {
	case AlertStatusOpen, AlertStatusAcknowledged, AlertStatusInProgress, AlertStatusResolved, AlertStatusFalsePositive:
		return true
	}
	return false
}

// String 返回状态字符串
func (s AlertStatus) String() string {
	return string(s)
}

// Alert 告警模型
type Alert struct {
	ID             uuid.UUID     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID       uuid.UUID     `gorm:"type:uuid;not null;index" json:"tenant_id"`
	AssetID        *uuid.UUID    `gorm:"type:uuid;index" json:"asset_id"`
	PolicyID       *uuid.UUID    `gorm:"type:uuid;index" json:"policy_id"`
	RuleName       string        `gorm:"type:varchar(255);not null" json:"rule_name"`
	Severity       AlertSeverity `gorm:"type:varchar(20);not null;index" json:"severity"`
	Title          string        `gorm:"type:varchar(500);not null" json:"title"`
	Description    string        `gorm:"type:text" json:"description"`
	Status         AlertStatus   `gorm:"type:varchar(20);not null;default:'open';index" json:"status"`
	SourceEventIDs StringSlice   `gorm:"type:text[]" json:"source_event_ids"`
	Context        AlertContext  `gorm:"type:jsonb;default:'{}'" json:"context"`
	AssignedTo     *uuid.UUID    `gorm:"type:uuid" json:"assigned_to"`
	AcknowledgedAt *time.Time    `gorm:"type:timestamptz" json:"acknowledged_at"`
	AcknowledgedBy *uuid.UUID    `gorm:"type:uuid" json:"acknowledged_by"`
	ResolvedAt     *time.Time    `gorm:"type:timestamptz" json:"resolved_at"`
	ResolvedBy     *uuid.UUID    `gorm:"type:uuid" json:"resolved_by"`
	Resolution     string        `gorm:"type:text" json:"resolution"`
	CreatedAt      time.Time     `gorm:"not null;default:CURRENT_TIMESTAMP;index" json:"created_at"`
	UpdatedAt      time.Time     `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`

	// Relations
	Tenant   *Tenant `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Policy   *Policy `gorm:"foreignKey:PolicyID" json:"policy,omitempty"`
	Assignee *User   `gorm:"foreignKey:AssignedTo" json:"assignee,omitempty"`
}

// TableName 指定表名
func (Alert) TableName() string {
	return "alerts"
}

// BeforeCreate GORM hook
func (a *Alert) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	now := time.Now().UTC()
	a.CreatedAt = now
	a.UpdatedAt = now
	return nil
}

// BeforeUpdate GORM hook
func (a *Alert) BeforeUpdate(tx *gorm.DB) error {
	a.UpdatedAt = time.Now().UTC()
	return nil
}

// AlertContext 告警上下文
type AlertContext struct {
	Process   *ProcessInfo           `json:"process,omitempty"`
	Network   *NetworkInfo           `json:"network,omitempty"`
	File      *FileInfo              `json:"file,omitempty"`
	MITRE     *MITREInfo             `json:"mitre,omitempty"`
	RawEvents []interface{}          `json:"raw_events,omitempty"`
	Custom    map[string]interface{} `json:"custom,omitempty"`
}

// Value 实现 driver.Valuer
func (c AlertContext) Value() (driver.Value, error) {
	return json.Marshal(c)
}

// Scan 实现 sql.Scanner
func (c *AlertContext) Scan(value interface{}) error {
	if value == nil {
		*c = AlertContext{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, c)
}

// ProcessInfo 进程信息
type ProcessInfo struct {
	PID         int    `json:"pid"`
	Name        string `json:"name"`
	CommandLine string `json:"command_line"`
	User        string `json:"user"`
	ParentPID   int    `json:"parent_pid"`
	Hash        string `json:"hash"`
}

// NetworkInfo 网络信息
type NetworkInfo struct {
	SourceIP        string `json:"source_ip"`
	SourcePort      int    `json:"source_port"`
	DestinationIP   string `json:"destination_ip"`
	DestinationPort int    `json:"destination_port"`
	Protocol        string `json:"protocol"`
}

// FileInfo 文件信息
type FileInfo struct {
	Path     string `json:"path"`
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	Hash     string `json:"hash"`
	Modified string `json:"modified"`
}

// MITREInfo MITRE ATT&CK 信息
type MITREInfo struct {
	TacticID      string   `json:"tactic_id"`
	TacticName    string   `json:"tactic_name"`
	TechniqueID   string   `json:"technique_id"`
	TechniqueName string   `json:"technique_name"`
	SubTechniques []string `json:"sub_techniques"`
}
