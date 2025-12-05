// Package models 定义数据库模型和公共类型
package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// StringSlice TEXT[] 类型，用于 PostgreSQL TEXT[] 存储
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
	for _, c := range str {
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

// UUIDSlice UUID[] 类型
type UUIDSlice []uuid.UUID

// Value 实现 driver.Valuer 接口
func (s UUIDSlice) Value() (driver.Value, error) {
	if s == nil || len(s) == 0 {
		return "{}", nil
	}
	result := "{"
	for i, v := range s {
		if i > 0 {
			result += ","
		}
		result += fmt.Sprintf("\"%s\"", v.String())
	}
	result += "}"
	return result, nil
}

// Scan 实现 sql.Scanner 接口
func (s *UUIDSlice) Scan(value interface{}) error {
	if value == nil {
		*s = []uuid.UUID{}
		return nil
	}

	var str string
	switch v := value.(type) {
	case []byte:
		str = string(v)
	case string:
		str = v
	default:
		return fmt.Errorf("failed to scan UUIDSlice: expected []byte or string, got %T", value)
	}

	if str == "{}" || str == "" {
		*s = []uuid.UUID{}
		return nil
	}

	// 移除首尾的 { }
	str = str[1 : len(str)-1]
	if str == "" {
		*s = []uuid.UUID{}
		return nil
	}

	// 解析 UUID 列表
	var result []uuid.UUID
	var current string
	inQuote := false
	for _, c := range str {
		switch c {
		case '"':
			inQuote = !inQuote
		case ',':
			if !inQuote {
				if current != "" {
					uid, err := uuid.Parse(current)
					if err != nil {
						return fmt.Errorf("failed to parse UUID %q: %w", current, err)
					}
					result = append(result, uid)
				}
				current = ""
			} else {
				current += string(c)
			}
		default:
			current += string(c)
		}
	}
	if current != "" {
		uid, err := uuid.Parse(current)
		if err != nil {
			return fmt.Errorf("failed to parse UUID %q: %w", current, err)
		}
		result = append(result, uid)
	}

	*s = result
	return nil
}

// JSONMap JSONB 类型的 map
type JSONMap map[string]interface{}

// Value 实现 driver.Valuer 接口
func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	return json.Marshal(m)
}

// Scan 实现 sql.Scanner 接口
func (m *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*m = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, m)
}

// ListOptions 分页和排序选项
type ListOptions struct {
	Offset  int    `json:"offset"`
	Limit   int    `json:"limit"`
	OrderBy string `json:"order_by"`
	Order   string `json:"order"` // "asc" or "desc"
}

// DefaultListOptions 默认分页选项
func DefaultListOptions() ListOptions {
	return ListOptions{
		Offset:  0,
		Limit:   20,
		OrderBy: "created_at",
		Order:   "desc",
	}
}

// Normalize 规范化分页选项
func (o *ListOptions) Normalize() {
	if o.Limit <= 0 {
		o.Limit = 20
	}
	if o.Limit > 100 {
		o.Limit = 100
	}
	if o.Offset < 0 {
		o.Offset = 0
	}
	if o.OrderBy == "" {
		o.OrderBy = "created_at"
	}
	if o.Order != "asc" && o.Order != "desc" {
		o.Order = "desc"
	}
}

// FilterOptions 过滤选项
type FilterOptions struct {
	Status    string     `json:"status"`
	StartTime *time.Time `json:"start_time"`
	EndTime   *time.Time `json:"end_time"`
	Search    string     `json:"search"`
}
