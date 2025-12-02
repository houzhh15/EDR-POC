package asset

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// AssetService 资产服务
type AssetService struct {
	repo      *AssetRepository
	statusMgr AgentStatusManager
	changelog *ChangeLogger
	logger    *zap.Logger
}

// NewAssetService 创建资产服务实例
func NewAssetService(repo *AssetRepository, statusMgr AgentStatusManager, changelog *ChangeLogger, logger *zap.Logger) *AssetService {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}
	return &AssetService{
		repo:      repo,
		statusMgr: statusMgr,
		changelog: changelog,
		logger:    logger,
	}
}

// 监控字段列表（变更时需要记录日志）
var monitoredFields = []string{
	"hostname",
	"os_version",
	"ip_addresses",
	"agent_version",
}

// RegisterOrUpdateFromHeartbeat 从心跳注册或更新资产（实现 grpc.AssetRegistrar 接口）
func (s *AssetService) RegisterOrUpdateFromHeartbeat(ctx context.Context, agentID, tenantID, hostname, osType, agentVersion string, ipAddresses []string) error {
	req := &RegisterAssetRequest{
		AgentID:      agentID,
		TenantID:     tenantID,
		Hostname:     hostname,
		OSType:       normalizeOSType(osType),
		AgentVersion: agentVersion,
		IPAddresses:  ipAddresses,
	}
	_, err := s.RegisterOrUpdateAsset(ctx, req)
	return err
}

// normalizeOSType 标准化操作系统类型
func normalizeOSType(osFamily string) string {
	switch strings.ToLower(osFamily) {
	case "darwin", "macos":
		return "macos"
	case "windows", "win32", "win64":
		return "windows"
	case "linux":
		return "linux"
	default:
		return osFamily
	}
}

// RegisterOrUpdateAsset 注册或更新资产
// 如果是首次注册，创建新资产；如果已存在，更新并记录变更
func (s *AssetService) RegisterOrUpdateAsset(ctx context.Context, req *RegisterAssetRequest) (*Asset, error) {
	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		return nil, ErrInvalidRequest.WithMessage("invalid tenant_id format")
	}

	// 查询是否已存在
	existing, err := s.repo.FindByAgentID(ctx, tenantID, req.AgentID)
	if err != nil && !errors.Is(err, ErrAssetNotFound) {
		return nil, fmt.Errorf("find existing asset: %w", err)
	}

	now := time.Now()

	if existing == nil {
		// 首次注册：创建新资产
		return s.createNewAsset(ctx, tenantID, req, now)
	}

	// 已存在：检测变更并更新
	return s.updateExistingAsset(ctx, existing, req, now)
}

// createNewAsset 创建新资产
func (s *AssetService) createNewAsset(ctx context.Context, tenantID uuid.UUID, req *RegisterAssetRequest, now time.Time) (*Asset, error) {
	asset := &Asset{
		AgentID:      req.AgentID,
		TenantID:     tenantID,
		Hostname:     req.Hostname,
		OSType:       req.OSType,
		OSVersion:    req.OSVersion,
		Architecture: req.Architecture,
		IPAddresses:  StringSlice(req.IPAddresses),
		MACAddresses: StringSlice(req.MACAddresses),
		AgentVersion: req.AgentVersion,
		Status:       AssetStatusOnline,
		LastSeenAt:   &now,
		FirstSeenAt:  now,
	}

	created, err := s.repo.Create(ctx, asset)
	if err != nil {
		return nil, fmt.Errorf("create asset: %w", err)
	}

	// 记录首次注册日志
	if s.changelog != nil {
		_ = s.changelog.LogChange(ctx, created.ID, "status", "", "registered", "system")
	}

	// 更新 Redis 状态
	if s.statusMgr != nil {
		info := &HeartbeatInfo{
			Hostname:     req.Hostname,
			IPAddress:    firstOrEmpty(req.IPAddresses),
			AgentVersion: req.AgentVersion,
			OSFamily:     req.OSType,
			Status:       "online",
		}
		if err := s.statusMgr.UpdateHeartbeat(ctx, req.AgentID, req.TenantID, info); err != nil {
			s.logger.Warn("failed to update heartbeat in Redis",
				zap.String("agent_id", req.AgentID),
				zap.Error(err),
			)
			// 不阻断主流程
		}
	}

	s.logger.Info("new asset registered",
		zap.String("id", created.ID.String()),
		zap.String("agent_id", req.AgentID),
		zap.String("hostname", req.Hostname),
	)

	return created, nil
}

// updateExistingAsset 更新已有资产
func (s *AssetService) updateExistingAsset(ctx context.Context, existing *Asset, req *RegisterAssetRequest, now time.Time) (*Asset, error) {
	// 检测变更
	changes := s.detectChanges(existing, req)

	// 应用变更到实体
	existing.Hostname = req.Hostname
	existing.OSVersion = req.OSVersion
	existing.Architecture = req.Architecture
	existing.IPAddresses = StringSlice(req.IPAddresses)
	existing.MACAddresses = StringSlice(req.MACAddresses)
	existing.AgentVersion = req.AgentVersion
	existing.Status = AssetStatusOnline
	existing.LastSeenAt = &now

	// 更新数据库
	updated, err := s.repo.Update(ctx, existing)
	if err != nil {
		return nil, fmt.Errorf("update asset: %w", err)
	}

	// 记录变更日志
	if len(changes) > 0 && s.changelog != nil {
		if err := s.changelog.LogMultipleChanges(ctx, existing.ID, changes, "agent"); err != nil {
			s.logger.Warn("failed to log changes",
				zap.String("asset_id", existing.ID.String()),
				zap.Error(err),
			)
		}
	}

	// 更新 Redis 状态
	if s.statusMgr != nil {
		info := &HeartbeatInfo{
			Hostname:     req.Hostname,
			IPAddress:    firstOrEmpty(req.IPAddresses),
			AgentVersion: req.AgentVersion,
			OSFamily:     req.OSType,
			Status:       "online",
		}
		if err := s.statusMgr.UpdateHeartbeat(ctx, req.AgentID, req.TenantID, info); err != nil {
			s.logger.Warn("failed to update heartbeat in Redis",
				zap.String("agent_id", req.AgentID),
				zap.Error(err),
			)
		}
	}

	if len(changes) > 0 {
		s.logger.Info("asset updated with changes",
			zap.String("id", existing.ID.String()),
			zap.Int("changes_count", len(changes)),
		)
	}

	return updated, nil
}

// detectChanges 检测监控字段的变更
func (s *AssetService) detectChanges(existing *Asset, req *RegisterAssetRequest) []FieldChange {
	var changes []FieldChange

	// hostname
	if existing.Hostname != req.Hostname {
		changes = append(changes, FieldChange{
			FieldName: "hostname",
			OldValue:  existing.Hostname,
			NewValue:  req.Hostname,
		})
	}

	// os_version
	if existing.OSVersion != req.OSVersion {
		changes = append(changes, FieldChange{
			FieldName: "os_version",
			OldValue:  existing.OSVersion,
			NewValue:  req.OSVersion,
		})
	}

	// ip_addresses
	oldIPs := strings.Join([]string(existing.IPAddresses), ",")
	newIPs := strings.Join(req.IPAddresses, ",")
	if oldIPs != newIPs {
		changes = append(changes, FieldChange{
			FieldName: "ip_addresses",
			OldValue:  oldIPs,
			NewValue:  newIPs,
		})
	}

	// agent_version
	if existing.AgentVersion != req.AgentVersion {
		changes = append(changes, FieldChange{
			FieldName: "agent_version",
			OldValue:  existing.AgentVersion,
			NewValue:  req.AgentVersion,
		})
	}

	return changes
}

// GetAsset 获取单个资产详情
func (s *AssetService) GetAsset(ctx context.Context, tenantID, assetID uuid.UUID) (*Asset, error) {
	return s.repo.FindByID(ctx, tenantID, assetID)
}

// GetAssetByAgentID 按 AgentID 获取资产
func (s *AssetService) GetAssetByAgentID(ctx context.Context, tenantID uuid.UUID, agentID string) (*Asset, error) {
	return s.repo.FindByAgentID(ctx, tenantID, agentID)
}

// ListAssets 分页查询资产列表
func (s *AssetService) ListAssets(ctx context.Context, tenantID uuid.UUID, opts *QueryOptions) ([]*Asset, int64, error) {
	if opts == nil {
		opts = &QueryOptions{}
	}

	if err := opts.Validate(); err != nil {
		return nil, 0, err
	}

	return s.repo.FindAll(ctx, tenantID, opts)
}

// UpdateAsset 更新资产信息（通过 API 更新）
func (s *AssetService) UpdateAsset(ctx context.Context, tenantID, assetID uuid.UUID, req *UpdateAssetRequest) (*Asset, error) {
	existing, err := s.repo.FindByID(ctx, tenantID, assetID)
	if err != nil {
		return nil, err
	}

	var changes []FieldChange

	// 选择性更新字段
	if req.Hostname != nil && *req.Hostname != existing.Hostname {
		changes = append(changes, FieldChange{
			FieldName: "hostname",
			OldValue:  existing.Hostname,
			NewValue:  *req.Hostname,
		})
		existing.Hostname = *req.Hostname
	}

	if req.OSVersion != nil && *req.OSVersion != existing.OSVersion {
		changes = append(changes, FieldChange{
			FieldName: "os_version",
			OldValue:  existing.OSVersion,
			NewValue:  *req.OSVersion,
		})
		existing.OSVersion = *req.OSVersion
	}

	if req.Architecture != nil {
		existing.Architecture = *req.Architecture
	}

	if req.IPAddresses != nil {
		oldIPs := strings.Join([]string(existing.IPAddresses), ",")
		newIPs := strings.Join(req.IPAddresses, ",")
		if oldIPs != newIPs {
			changes = append(changes, FieldChange{
				FieldName: "ip_addresses",
				OldValue:  oldIPs,
				NewValue:  newIPs,
			})
		}
		existing.IPAddresses = StringSlice(req.IPAddresses)
	}

	if req.MACAddresses != nil {
		existing.MACAddresses = StringSlice(req.MACAddresses)
	}

	if req.AgentVersion != nil && *req.AgentVersion != existing.AgentVersion {
		changes = append(changes, FieldChange{
			FieldName: "agent_version",
			OldValue:  existing.AgentVersion,
			NewValue:  *req.AgentVersion,
		})
		existing.AgentVersion = *req.AgentVersion
	}

	// 更新数据库
	updated, err := s.repo.Update(ctx, existing)
	if err != nil {
		return nil, err
	}

	// 记录变更日志
	if len(changes) > 0 && s.changelog != nil {
		_ = s.changelog.LogMultipleChanges(ctx, assetID, changes, "api")
	}

	return updated, nil
}

// DeleteAsset 软删除资产
func (s *AssetService) DeleteAsset(ctx context.Context, tenantID, assetID uuid.UUID) error {
	return s.repo.SoftDelete(ctx, tenantID, assetID)
}

// GetAssetChanges 获取资产变更历史
func (s *AssetService) GetAssetChanges(ctx context.Context, assetID uuid.UUID, opts *ChangeLogQueryOptions) ([]*AssetChangeLog, int64, error) {
	if s.changelog == nil {
		return nil, 0, nil
	}
	return s.changelog.GetChangeHistory(ctx, assetID, opts)
}

// CountByStatus 按状态统计资产数量
func (s *AssetService) CountByStatus(ctx context.Context, tenantID uuid.UUID) (map[AssetStatus]int64, error) {
	return s.repo.CountByStatus(ctx, tenantID)
}

// firstOrEmpty 返回切片的第一个元素或空字符串
func firstOrEmpty(slice []string) string {
	if len(slice) > 0 {
		return slice[0]
	}
	return ""
}
