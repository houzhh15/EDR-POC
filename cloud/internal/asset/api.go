// Package asset 提供资产管理 HTTP API 处理器
package asset

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// AssetHandler HTTP API 处理器
type AssetHandler struct {
	assetService    *AssetService
	groupService    *GroupService
	softwareService *SoftwareService
	changeLogger    *ChangeLogger
	logger          *zap.Logger
}

// NewAssetHandler 创建 AssetHandler 实例
func NewAssetHandler(
	assetService *AssetService,
	groupService *GroupService,
	softwareService *SoftwareService,
	changeLogger *ChangeLogger,
	logger *zap.Logger,
) *AssetHandler {
	return &AssetHandler{
		assetService:    assetService,
		groupService:    groupService,
		softwareService: softwareService,
		changeLogger:    changeLogger,
		logger:          logger,
	}
}

// RegisterRoutes 注册路由
func (h *AssetHandler) RegisterRoutes(rg *gin.RouterGroup) {
	// 资产管理 API
	assets := rg.Group("/assets")
	{
		assets.GET("", h.ListAssets)
		assets.GET("/:id", h.GetAsset)
		assets.PUT("/:id", h.UpdateAsset)
		assets.DELETE("/:id", h.DeleteAsset)
		assets.GET("/:id/software", h.GetAssetSoftware)
		assets.GET("/:id/changes", h.GetAssetChanges)
	}

	// 分组管理 API
	groups := rg.Group("/asset-groups")
	{
		groups.GET("", h.ListGroups)
		groups.POST("", h.CreateGroup)
		groups.PUT("/:id", h.UpdateGroup)
		groups.DELETE("/:id", h.DeleteGroup)
		groups.POST("/:id/assets", h.AddAssetToGroup)
		groups.DELETE("/:id/assets/:assetId", h.RemoveAssetFromGroup)
		groups.GET("/:id/assets", h.GetGroupAssets)
	}

	// 软件搜索 API
	rg.GET("/software/search", h.SearchSoftware)
}

// getTenantID 从上下文获取租户ID（从JWT中提取）
func (h *AssetHandler) getTenantID(c *gin.Context) (uuid.UUID, bool) {
	// 优先从JWT context获取
	if tenantIDStr, exists := c.Get("tenant_id"); exists {
		if tid, ok := tenantIDStr.(string); ok {
			if id, err := uuid.Parse(tid); err == nil {
				return id, true
			}
		}
		if tid, ok := tenantIDStr.(uuid.UUID); ok {
			return tid, true
		}
	}

	// 从Header获取（开发测试用）
	if tenantIDStr := c.GetHeader("X-Tenant-ID"); tenantIDStr != "" {
		if id, err := uuid.Parse(tenantIDStr); err == nil {
			return id, true
		}
	}

	h.respondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "tenant_id not found in context")
	return uuid.Nil, false
}

// respondError 返回错误响应
func (h *AssetHandler) respondError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	})
}

// respondAssetError 返回资产错误响应
func (h *AssetHandler) respondAssetError(c *gin.Context, err error) {
	if assetErr, ok := err.(*AssetError); ok {
		c.JSON(assetErr.HTTPStatus, gin.H{
			"error": gin.H{
				"code":    assetErr.Code,
				"message": assetErr.Message,
			},
		})
		return
	}

	h.logger.Error("internal error", zap.Error(err))
	h.respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
}

// ==================== 资产管理 API ====================

// ListAssets 获取资产列表
// GET /api/v1/assets
func (h *AssetHandler) ListAssets(c *gin.Context) {
	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}

	// 解析查询参数
	opts := &QueryOptions{
		Pagination: Pagination{
			Page:     h.parseIntDefault(c.Query("page"), 1),
			PageSize: h.parseIntDefault(c.Query("page_size"), 20),
		},
		Status:    c.Query("status"),
		OSType:    c.Query("os_type"),
		Hostname:  c.Query("hostname"),
		IP:        c.Query("ip"),
		SortBy:    c.Query("sort_by"),
		SortOrder: c.Query("sort_order"),
	}

	// 解析分组ID
	if groupIDStr := c.Query("group_id"); groupIDStr != "" {
		if groupID, err := uuid.Parse(groupIDStr); err == nil {
			opts.GroupID = &groupID
		}
	}

	assets, total, err := h.assetService.ListAssets(c.Request.Context(), tenantID, opts)
	if err != nil {
		h.respondAssetError(c, err)
		return
	}

	// 转换为响应类型
	assetResponses := make([]AssetResponse, len(assets))
	for i, a := range assets {
		assetResponses[i].FromAsset(a)
	}

	// 构建分页响应
	opts.Pagination.Normalize()
	totalPages := int(total) / opts.Pagination.PageSize
	if int(total)%opts.Pagination.PageSize != 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, AssetListResponse{
		Data: assetResponses,
		Pagination: Pagination{
			Page:      opts.Pagination.Page,
			PageSize:  opts.Pagination.PageSize,
			Total:     int(total),
			TotalPage: totalPages,
		},
	})
}

// GetAsset 获取资产详情
// GET /api/v1/assets/:id
func (h *AssetHandler) GetAsset(c *gin.Context) {
	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}

	assetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid asset ID format")
		return
	}

	asset, err := h.assetService.GetAsset(c.Request.Context(), tenantID, assetID)
	if err != nil {
		h.respondAssetError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": asset})
}

// UpdateAsset 更新资产
// PUT /api/v1/assets/:id
func (h *AssetHandler) UpdateAsset(c *gin.Context) {
	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}

	assetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid asset ID format")
		return
	}

	var req UpdateAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	asset, err := h.assetService.UpdateAsset(c.Request.Context(), tenantID, assetID, &req)
	if err != nil {
		h.respondAssetError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": asset})
}

// DeleteAsset 删除资产
// DELETE /api/v1/assets/:id
func (h *AssetHandler) DeleteAsset(c *gin.Context) {
	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}

	assetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid asset ID format")
		return
	}

	if err := h.assetService.DeleteAsset(c.Request.Context(), tenantID, assetID); err != nil {
		h.respondAssetError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Asset deleted successfully"})
}

// GetAssetSoftware 获取资产软件清单
// GET /api/v1/assets/:id/software
func (h *AssetHandler) GetAssetSoftware(c *gin.Context) {
	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}

	assetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid asset ID format")
		return
	}

	// 验证资产存在且属于该租户
	if _, err := h.assetService.GetAsset(c.Request.Context(), tenantID, assetID); err != nil {
		h.respondAssetError(c, err)
		return
	}

	opts := &SoftwareQueryOptions{
		Pagination: Pagination{
			Page:     h.parseIntDefault(c.Query("page"), 1),
			PageSize: h.parseIntDefault(c.Query("page_size"), 50),
		},
		Name:      c.Query("name"),
		Publisher: c.Query("publisher"),
	}

	software, total, err := h.softwareService.GetSoftwareByAsset(c.Request.Context(), assetID, opts)
	if err != nil {
		h.respondAssetError(c, err)
		return
	}

	// 转换为响应类型
	softwareResponses := make([]SoftwareResponse, len(software))
	for i, s := range software {
		softwareResponses[i].FromSoftwareInventory(s)
	}

	opts.Pagination.Normalize()
	totalPages := int(total) / opts.Pagination.PageSize
	if int(total)%opts.Pagination.PageSize != 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, SoftwareListResponse{
		Data: softwareResponses,
		Pagination: Pagination{
			Page:      opts.Pagination.Page,
			PageSize:  opts.Pagination.PageSize,
			Total:     int(total),
			TotalPage: totalPages,
		},
	})
}

// GetAssetChanges 获取资产变更历史
// GET /api/v1/assets/:id/changes
func (h *AssetHandler) GetAssetChanges(c *gin.Context) {
	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}

	assetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid asset ID format")
		return
	}

	// 验证资产存在且属于该租户
	if _, err := h.assetService.GetAsset(c.Request.Context(), tenantID, assetID); err != nil {
		h.respondAssetError(c, err)
		return
	}

	limit := h.parseIntDefault(c.Query("limit"), 50)
	if limit > 200 {
		limit = 200
	}

	opts := &ChangeLogQueryOptions{
		Pagination: Pagination{
			Page:     1,
			PageSize: limit,
		},
	}

	changes, _, err := h.changeLogger.GetChangeHistory(c.Request.Context(), assetID, opts)
	if err != nil {
		h.respondAssetError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": changes})
}

// ==================== 分组管理 API ====================

// ListGroups 获取分组树
// GET /api/v1/asset-groups
func (h *AssetHandler) ListGroups(c *gin.Context) {
	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}

	tree, err := h.groupService.GetGroupTree(c.Request.Context(), tenantID)
	if err != nil {
		h.respondAssetError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": tree})
}

// CreateGroup 创建分组
// POST /api/v1/asset-groups
func (h *AssetHandler) CreateGroup(c *gin.Context) {
	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}

	var req CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	group, err := h.groupService.CreateGroup(c.Request.Context(), tenantID, &req)
	if err != nil {
		h.respondAssetError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": group})
}

// UpdateGroup 更新分组
// PUT /api/v1/asset-groups/:id
func (h *AssetHandler) UpdateGroup(c *gin.Context) {
	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}

	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid group ID format")
		return
	}

	var req UpdateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	group, err := h.groupService.UpdateGroup(c.Request.Context(), tenantID, groupID, &req)
	if err != nil {
		h.respondAssetError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": group})
}

// DeleteGroup 删除分组
// DELETE /api/v1/asset-groups/:id
func (h *AssetHandler) DeleteGroup(c *gin.Context) {
	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}

	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid group ID format")
		return
	}

	if err := h.groupService.DeleteGroup(c.Request.Context(), tenantID, groupID); err != nil {
		h.respondAssetError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Group deleted successfully"})
}

// AddAssetToGroup 将资产添加到分组
// POST /api/v1/asset-groups/:id/assets
func (h *AssetHandler) AddAssetToGroup(c *gin.Context) {
	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}

	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid group ID format")
		return
	}

	var req struct {
		AssetID string `json:"asset_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	assetID, err := uuid.Parse(req.AssetID)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid asset_id format")
		return
	}

	// 验证资产存在且属于该租户
	if _, err := h.assetService.GetAsset(c.Request.Context(), tenantID, assetID); err != nil {
		h.respondAssetError(c, err)
		return
	}

	if err := h.groupService.AssignAsset(c.Request.Context(), tenantID, groupID, assetID); err != nil {
		h.respondAssetError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Asset added to group successfully"})
}

// RemoveAssetFromGroup 从分组移除资产
// DELETE /api/v1/asset-groups/:id/assets/:assetId
func (h *AssetHandler) RemoveAssetFromGroup(c *gin.Context) {
	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}

	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid group ID format")
		return
	}

	assetID, err := uuid.Parse(c.Param("assetId"))
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid asset ID format")
		return
	}

	if err := h.groupService.RemoveAsset(c.Request.Context(), tenantID, groupID, assetID); err != nil {
		h.respondAssetError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Asset removed from group successfully"})
}

// GetGroupAssets 获取分组下的资产
// GET /api/v1/asset-groups/:id/assets
func (h *AssetHandler) GetGroupAssets(c *gin.Context) {
	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}

	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid group ID format")
		return
	}

	opts := &QueryOptions{
		Pagination: Pagination{
			Page:     h.parseIntDefault(c.Query("page"), 1),
			PageSize: h.parseIntDefault(c.Query("page_size"), 20),
		},
	}
	opts.Pagination.Normalize()

	assets, total, err := h.groupService.GetAssetsByGroup(c.Request.Context(), tenantID, groupID, opts)
	if err != nil {
		h.respondAssetError(c, err)
		return
	}

	// 转换为响应类型
	assetResponses := make([]AssetResponse, len(assets))
	for i, a := range assets {
		assetResponses[i].FromAsset(a)
	}

	totalPages := int(total) / opts.Pagination.PageSize
	if int(total)%opts.Pagination.PageSize != 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, AssetListResponse{
		Data: assetResponses,
		Pagination: Pagination{
			Page:      opts.Pagination.Page,
			PageSize:  opts.Pagination.PageSize,
			Total:     int(total),
			TotalPage: totalPages,
		},
	})
}

// ==================== 软件搜索 API ====================

// SearchSoftware 跨资产搜索软件
// GET /api/v1/software/search
func (h *AssetHandler) SearchSoftware(c *gin.Context) {
	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}

	name := c.Query("name")
	if name == "" {
		h.respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "name parameter is required")
		return
	}

	pagination := &Pagination{
		Page:     h.parseIntDefault(c.Query("page"), 1),
		PageSize: h.parseIntDefault(c.Query("page_size"), 50),
	}
	pagination.Normalize()

	results, total, err := h.softwareService.SearchSoftware(c.Request.Context(), tenantID, name, pagination)
	if err != nil {
		h.respondAssetError(c, err)
		return
	}

	totalPages := int(total) / pagination.PageSize
	if int(total)%pagination.PageSize != 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, gin.H{
		"data": results,
		"pagination": PaginationResponse{
			Page:       pagination.Page,
			PageSize:   pagination.PageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

// ==================== 辅助方法 ====================

func (h *AssetHandler) parseIntDefault(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	return defaultVal
}
