package repository

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// 预定义错误
var (
	ErrNotFound    = errors.New("record not found")
	ErrDuplicate   = errors.New("duplicate entry")
	ErrForeignKey  = errors.New("foreign key constraint violation")
	ErrConnection  = errors.New("database connection error")
	ErrTransaction = errors.New("transaction error")
)

// IsDuplicateError 检查是否为唯一约束冲突
func IsDuplicateError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505" // unique_violation
	}
	return false
}

// IsForeignKeyError 检查是否为外键约束冲突
func IsForeignKeyError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23503" // foreign_key_violation
	}
	return false
}

// IsNotFoundError 检查是否为记录未找到错误
func IsNotFoundError(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

// WrapError 包装数据库错误
func WrapError(err error, operation string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil // 未找到不视为错误
	}
	if IsDuplicateError(err) {
		return fmt.Errorf("%s: %w", operation, ErrDuplicate)
	}
	if IsForeignKeyError(err) {
		return fmt.Errorf("%s: %w", operation, ErrForeignKey)
	}
	return fmt.Errorf("%s: %w", operation, err)
}
