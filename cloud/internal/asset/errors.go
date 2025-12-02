package asset

import (
	"fmt"
	"net/http"
)

// AssetError 资产服务错误
type AssetError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"-"`
	Err        error  `json:"-"`
}

// Error 实现 error 接口
func (e *AssetError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap 返回包装的底层错误
func (e *AssetError) Unwrap() error {
	return e.Err
}

// WithError 附加底层错误
func (e *AssetError) WithError(err error) *AssetError {
	return &AssetError{
		Code:       e.Code,
		Message:    e.Message,
		HTTPStatus: e.HTTPStatus,
		Err:        err,
	}
}

// WithMessage 附加自定义消息
func (e *AssetError) WithMessage(msg string) *AssetError {
	return &AssetError{
		Code:       e.Code,
		Message:    msg,
		HTTPStatus: e.HTTPStatus,
		Err:        e.Err,
	}
}

// 预定义错误常量
var (
	// ErrAssetNotFound 资产不存在
	ErrAssetNotFound = &AssetError{
		Code:       "ASSET_NOT_FOUND",
		Message:    "Asset not found",
		HTTPStatus: http.StatusNotFound,
	}

	// ErrGroupNotFound 分组不存在
	ErrGroupNotFound = &AssetError{
		Code:       "GROUP_NOT_FOUND",
		Message:    "Asset group not found",
		HTTPStatus: http.StatusNotFound,
	}

	// ErrGroupHasChildren 分组存在子分组无法删除
	ErrGroupHasChildren = &AssetError{
		Code:       "GROUP_HAS_CHILDREN",
		Message:    "Cannot delete group with children",
		HTTPStatus: http.StatusConflict,
	}

	// ErrInvalidRequest 请求参数无效
	ErrInvalidRequest = &AssetError{
		Code:       "INVALID_REQUEST",
		Message:    "Invalid request parameters",
		HTTPStatus: http.StatusBadRequest,
	}

	// ErrGroupDepthExceeded 分组层级超限
	ErrGroupDepthExceeded = &AssetError{
		Code:       "GROUP_DEPTH_EXCEEDED",
		Message:    "Group depth exceeds maximum allowed level (5)",
		HTTPStatus: http.StatusBadRequest,
	}

	// ErrDuplicateAsset 资产重复
	ErrDuplicateAsset = &AssetError{
		Code:       "DUPLICATE_ASSET",
		Message:    "Asset with this agent_id already exists",
		HTTPStatus: http.StatusConflict,
	}

	// ErrDuplicateGroupName 分组名重复
	ErrDuplicateGroupName = &AssetError{
		Code:       "DUPLICATE_GROUP_NAME",
		Message:    "Group name already exists in the same parent",
		HTTPStatus: http.StatusConflict,
	}

	// ErrInternalError 内部服务错误
	ErrInternalError = &AssetError{
		Code:       "INTERNAL_ERROR",
		Message:    "Internal server error",
		HTTPStatus: http.StatusInternalServerError,
	}

	// ErrUnauthorized 未授权
	ErrUnauthorized = &AssetError{
		Code:       "UNAUTHORIZED",
		Message:    "Unauthorized access",
		HTTPStatus: http.StatusUnauthorized,
	}

	// ErrForbidden 禁止访问
	ErrForbidden = &AssetError{
		Code:       "FORBIDDEN",
		Message:    "Access forbidden",
		HTTPStatus: http.StatusForbidden,
	}

	// ErrSoftwareNotFound 软件记录不存在
	ErrSoftwareNotFound = &AssetError{
		Code:       "SOFTWARE_NOT_FOUND",
		Message:    "Software inventory not found",
		HTTPStatus: http.StatusNotFound,
	}

	// ErrAssetAlreadyInGroup 资产已在分组中
	ErrAssetAlreadyInGroup = &AssetError{
		Code:       "ASSET_ALREADY_IN_GROUP",
		Message:    "Asset is already a member of this group",
		HTTPStatus: http.StatusConflict,
	}

	// ErrAssetNotInGroup 资产不在分组中
	ErrAssetNotInGroup = &AssetError{
		Code:       "ASSET_NOT_IN_GROUP",
		Message:    "Asset is not a member of this group",
		HTTPStatus: http.StatusNotFound,
	}
)

// IsAssetError 检查是否为资产服务错误
func IsAssetError(err error) bool {
	_, ok := err.(*AssetError)
	return ok
}

// GetAssetError 获取资产服务错误（如果是的话）
func GetAssetError(err error) *AssetError {
	if assetErr, ok := err.(*AssetError); ok {
		return assetErr
	}
	return nil
}
