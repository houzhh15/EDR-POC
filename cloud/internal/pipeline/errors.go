// Package pipeline provides error handling for the event processing pipeline.
package pipeline

import (
	"errors"
	"fmt"
	"net"
)

// PipelineError 管线错误
type PipelineError struct {
	Stage     string // decode, enrich, normalize, write
	EventID   string
	Err       error
	Retryable bool
}

// Error 实现 error 接口
func (e *PipelineError) Error() string {
	return fmt.Sprintf("pipeline error at %s (event_id=%s): %v", e.Stage, e.EventID, e.Err)
}

// Unwrap 返回原始错误
func (e *PipelineError) Unwrap() error {
	return e.Err
}

// NewPipelineError 创建新的管线错误
func NewPipelineError(stage, eventID string, err error, retryable bool) *PipelineError {
	return &PipelineError{
		Stage:     stage,
		EventID:   eventID,
		Err:       err,
		Retryable: retryable,
	}
}

// Error stages
const (
	StageConsume   = "consume"
	StageDecode    = "decode"
	StageEnrich    = "enrich"
	StageNormalize = "normalize"
	StageWrite     = "write"
	StageBatch     = "batch"
)

// Predefined errors
var (
	ErrUnsupportedEventType = errors.New("unsupported event type")
	ErrInvalidEventData     = errors.New("invalid event data")
	ErrEnrichmentFailed     = errors.New("enrichment failed")
	ErrNormalizationFailed  = errors.New("normalization failed")
	ErrWriteFailed          = errors.New("write failed")
	ErrPipelineClosed       = errors.New("pipeline is closed")
	ErrConsumerClosed       = errors.New("consumer is closed")
)

// IsRetryable 判断错误是否可重试
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// 检查是否是 PipelineError
	var pErr *PipelineError
	if errors.As(err, &pErr) {
		return pErr.Retryable
	}

	// 网络超时错误可重试
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout() || netErr.Temporary()
	}

	// 上下文取消和超时不重试
	if errors.Is(err, net.ErrClosed) {
		return false
	}

	// 默认假设不可重试
	return false
}

// WrapDecodeError 包装解码错误
func WrapDecodeError(eventID string, err error) *PipelineError {
	return &PipelineError{
		Stage:     StageDecode,
		EventID:   eventID,
		Err:       err,
		Retryable: false, // 解码错误通常不可重试
	}
}

// WrapEnrichError 包装丰富化错误
func WrapEnrichError(eventID string, err error) *PipelineError {
	return &PipelineError{
		Stage:     StageEnrich,
		EventID:   eventID,
		Err:       err,
		Retryable: IsRetryable(err),
	}
}

// WrapNormalizeError 包装标准化错误
func WrapNormalizeError(eventID string, err error) *PipelineError {
	return &PipelineError{
		Stage:     StageNormalize,
		EventID:   eventID,
		Err:       err,
		Retryable: false, // 标准化错误通常不可重试
	}
}

// WrapWriteError 包装写入错误
func WrapWriteError(eventID string, err error) *PipelineError {
	return &PipelineError{
		Stage:     StageWrite,
		EventID:   eventID,
		Err:       err,
		Retryable: IsRetryable(err),
	}
}
