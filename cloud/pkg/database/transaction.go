package database

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// TxFunc 事务函数类型
type TxFunc func(tx *gorm.DB) error

// WithTransaction 在事务中执行函数
// 如果函数返回 error 或 panic，事务将回滚
// 否则事务将提交
func WithTransaction(db *gorm.DB, fn TxFunc) error {
	return db.Transaction(func(tx *gorm.DB) error {
		return fn(tx)
	})
}

// WithTransactionCtx 带上下文的事务执行
func WithTransactionCtx(ctx context.Context, db *gorm.DB, fn TxFunc) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(tx)
	})
}

// TxOptions 事务选项
type TxOptions struct {
	IsolationLevel string // 隔离级别
	ReadOnly       bool   // 只读事务
}

// WithTransactionOpts 带选项的事务执行
func WithTransactionOpts(db *gorm.DB, opts TxOptions, fn TxFunc) error {
	// GORM 默认使用 READ COMMITTED 隔离级别
	// 如需更高隔离级别，需要手动设置
	tx := db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r) // 重新抛出 panic
		}
	}()

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// WithNestedTransaction 嵌套事务（使用 SavePoint）
func WithNestedTransaction(tx *gorm.DB, name string, fn TxFunc) error {
	// 创建 SavePoint
	if err := tx.SavePoint(name).Error; err != nil {
		return fmt.Errorf("failed to create savepoint %s: %w", name, err)
	}

	defer func() {
		if r := recover(); r != nil {
			tx.RollbackTo(name)
			panic(r)
		}
	}()

	if err := fn(tx); err != nil {
		tx.RollbackTo(name)
		return err
	}

	return nil
}
