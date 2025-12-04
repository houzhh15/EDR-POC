// Package opensearch provides a Go client SDK for OpenSearch.
// It includes bulk indexing, query building, and index management capabilities.
package opensearch

import (
	"errors"
	"fmt"
)

// 预定义错误类型
var (
	// ErrConnectionFailed 连接失败
	ErrConnectionFailed = errors.New("opensearch: connection failed")

	// ErrBulkPartialFailure 批量操作部分失败
	ErrBulkPartialFailure = errors.New("opensearch: bulk partial failure")

	// ErrIndexNotFound 索引不存在
	ErrIndexNotFound = errors.New("opensearch: index not found")

	// ErrMappingConflict 映射冲突
	ErrMappingConflict = errors.New("opensearch: mapping conflict")

	// ErrTimeout 请求超时
	ErrTimeout = errors.New("opensearch: request timeout")

	// ErrRateLimited 被限流
	ErrRateLimited = errors.New("opensearch: rate limited")

	// ErrClientClosed 客户端已关闭
	ErrClientClosed = errors.New("opensearch: client closed")

	// ErrInvalidConfig 无效配置
	ErrInvalidConfig = errors.New("opensearch: invalid config")

	// ErrInvalidResponse 无效响应
	ErrInvalidResponse = errors.New("opensearch: invalid response")

	// ErrUnauthorized 认证失败
	ErrUnauthorized = errors.New("opensearch: unauthorized")

	// ErrForbidden 权限不足
	ErrForbidden = errors.New("opensearch: forbidden")
)

// BulkError 批量操作错误详情
type BulkError struct {
	Index      string `json:"_index"`
	DocumentID string `json:"_id"`
	Type       string `json:"type"`
	Reason     string `json:"reason"`
	Status     int    `json:"status"`
}

// Error implements error interface
func (e *BulkError) Error() string {
	return fmt.Sprintf("bulk error: index=%s id=%s type=%s reason=%s status=%d",
		e.Index, e.DocumentID, e.Type, e.Reason, e.Status)
}

// BulkErrors 批量操作错误列表
type BulkErrors struct {
	Errors []*BulkError
}

// Error implements error interface
func (e *BulkErrors) Error() string {
	return fmt.Sprintf("bulk operation failed with %d errors", len(e.Errors))
}

// Unwrap returns the underlying error
func (e *BulkErrors) Unwrap() error {
	return ErrBulkPartialFailure
}

// ResponseError OpenSearch 响应错误
type ResponseError struct {
	StatusCode int
	Type       string
	Reason     string
	RootCause  []struct {
		Type   string `json:"type"`
		Reason string `json:"reason"`
	}
}

// Error implements error interface
func (e *ResponseError) Error() string {
	return fmt.Sprintf("opensearch error: status=%d type=%s reason=%s",
		e.StatusCode, e.Type, e.Reason)
}

// IsNotFound 检查是否为索引不存在错误
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrIndexNotFound) {
		return true
	}
	var respErr *ResponseError
	if errors.As(err, &respErr) {
		return respErr.StatusCode == 404
	}
	return false
}

// IsConflict 检查是否为版本冲突错误
func IsConflict(err error) bool {
	if err == nil {
		return false
	}
	var respErr *ResponseError
	if errors.As(err, &respErr) {
		return respErr.StatusCode == 409
	}
	return false
}

// IsTimeout 检查是否为超时错误
func IsTimeout(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrTimeout)
}

// IsRateLimited 检查是否为限流错误
func IsRateLimited(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrRateLimited) {
		return true
	}
	var respErr *ResponseError
	if errors.As(err, &respErr) {
		return respErr.StatusCode == 429
	}
	return false
}

// IsRetryable 检查错误是否可重试
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}
	if IsTimeout(err) || IsRateLimited(err) {
		return true
	}
	var respErr *ResponseError
	if errors.As(err, &respErr) {
		// 5xx 错误通常可重试
		return respErr.StatusCode >= 500 && respErr.StatusCode < 600
	}
	return errors.Is(err, ErrConnectionFailed)
}
