package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TenantScope 多租户过滤
func TenantScope(tenantID uuid.UUID) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("tenant_id = ?", tenantID)
	}
}

// NotDeletedScope 软删除过滤
func NotDeletedScope() func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("deleted_at IS NULL")
	}
}

// StatusScope 状态过滤
func StatusScope(status string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if status == "" {
			return db
		}
		return db.Where("status = ?", status)
	}
}

// EnabledScope 启用状态过滤
func EnabledScope(enabled bool) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("enabled = ?", enabled)
	}
}

// PaginationScope 分页
func PaginationScope(limit, offset int) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if limit <= 0 {
			limit = 20
		}
		if limit > 100 {
			limit = 100
		}
		if offset < 0 {
			offset = 0
		}
		return db.Offset(offset).Limit(limit)
	}
}

// SearchScope 模糊搜索
func SearchScope(field, keyword string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if keyword == "" {
			return db
		}
		return db.Where(field+" ILIKE ?", "%"+keyword+"%")
	}
}

// OrderScope 排序
func OrderScope(orderBy, order string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if orderBy == "" {
			orderBy = "created_at"
		}
		if order != "asc" && order != "desc" {
			order = "desc"
		}
		return db.Order(orderBy + " " + order)
	}
}
