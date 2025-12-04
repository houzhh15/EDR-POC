// Package comm 提供 Agent 与云端的通信功能
package comm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/houzhh15/EDR-POC/agent/main-go/internal/log"
	pb "github.com/houzhh15/EDR-POC/agent/main-go/pkg/proto/edr/v1"
)

// PolicyApplier 策略应用接口
type PolicyApplier interface {
	// ApplyPolicy 应用单个策略更新
	ApplyPolicy(ctx context.Context, update *pb.PolicyUpdate) error
}

// PolicyConfig 策略同步配置
type PolicyConfig struct {
	AgentID         string        // Agent ID
	SyncInterval    time.Duration // 同步间隔
	RetryInterval   time.Duration // 重试间隔
	MaxRetries      int           // 最大重试次数
	ChecksumEnabled bool          // 是否启用校验和验证
	CurrentVersion  string        // 当前策略版本
	PolicyTypes     []string      // 订阅的策略类型 (detection/response/collection)
	PolicyStorePath string        // 策略本地存储路径（用于离线启动）
}

// PolicyClient 策略同步客户端
type PolicyClient struct {
	config  PolicyConfig
	conn    *Connection
	logger  *log.Logger
	applier PolicyApplier

	mu             sync.RWMutex
	currentVersion string
	lastSyncTime   time.Time
	isRunning      bool
	stopCh         chan struct{}
}

// NewPolicyClient 创建策略客户端
func NewPolicyClient(conn *Connection, config PolicyConfig, logger *log.Logger) *PolicyClient {
	if config.SyncInterval <= 0 {
		config.SyncInterval = 5 * time.Minute
	}
	if config.RetryInterval <= 0 {
		config.RetryInterval = 30 * time.Second
	}
	if config.MaxRetries <= 0 {
		config.MaxRetries = 3
	}

	return &PolicyClient{
		config:         config,
		conn:           conn,
		logger:         logger,
		currentVersion: config.CurrentVersion,
		stopCh:         make(chan struct{}),
	}
}

// SetApplier 设置策略应用器
func (p *PolicyClient) SetApplier(applier PolicyApplier) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.applier = applier
}

// Start 启动周期性策略同步
func (p *PolicyClient) Start(ctx context.Context) {
	p.mu.Lock()
	if p.isRunning {
		p.mu.Unlock()
		return
	}
	p.isRunning = true
	p.stopCh = make(chan struct{})
	p.mu.Unlock()

	p.logger.Info("Policy client starting",
		zap.String("agent_id", p.config.AgentID),
		zap.Duration("sync_interval", p.config.SyncInterval),
	)

	ticker := time.NewTicker(p.config.SyncInterval)
	defer ticker.Stop()

	// 启动时立即同步一次
	p.Sync(ctx)

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("Policy client stopped")
			p.mu.Lock()
			p.isRunning = false
			p.mu.Unlock()
			return
		case <-p.stopCh:
			p.logger.Info("Policy client stopped by signal")
			p.mu.Lock()
			p.isRunning = false
			p.mu.Unlock()
			return
		case <-ticker.C:
			p.Sync(ctx)
		}
	}
}

// Stop 停止策略同步
func (p *PolicyClient) Stop() {
	p.mu.Lock()
	if !p.isRunning {
		p.mu.Unlock()
		return
	}
	p.mu.Unlock()

	close(p.stopCh)
}

// Sync 执行一次策略同步
func (p *PolicyClient) Sync(ctx context.Context) error {
	conn := p.conn.GetConn()
	if conn == nil {
		p.logger.Warn("Cannot sync policy: not connected")
		return fmt.Errorf("not connected")
	}

	p.mu.RLock()
	currentVersion := p.currentVersion
	p.mu.RUnlock()

	p.logger.Debug("Starting policy sync",
		zap.String("current_version", currentVersion),
	)

	client := pb.NewAgentServiceClient(conn)

	// 带重试的策略获取
	var lastErr error
	for retry := 0; retry <= p.config.MaxRetries; retry++ {
		if retry > 0 {
			p.logger.Debug("Retrying policy sync",
				zap.Int("attempt", retry+1),
				zap.Int("max_retries", p.config.MaxRetries),
			)
			time.Sleep(p.config.RetryInterval)
		}

		err := p.fetchAndApplyPolicies(ctx, client, currentVersion)
		if err == nil {
			return nil
		}
		lastErr = err

		// 检查是否为不可重试错误
		if !isRetryableError(err) {
			break
		}
	}

	p.logger.Error("Policy sync failed after retries",
		zap.Error(lastErr),
		zap.Int("max_retries", p.config.MaxRetries),
	)
	return lastErr
}

// fetchAndApplyPolicies 通过 SyncPolicy 流获取并应用策略
func (p *PolicyClient) fetchAndApplyPolicies(ctx context.Context, client pb.AgentServiceClient, currentVersion string) error {
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	req := &pb.PolicyRequest{
		AgentId:        p.config.AgentID,
		CurrentVersion: currentVersion,
		PolicyTypes:    p.config.PolicyTypes,
	}

	stream, err := client.SyncPolicy(ctx, req)
	if err != nil {
		return fmt.Errorf("sync policy failed: %w", err)
	}

	// 存储分块策略（按 policy_id 分组）
	policyChunks := make(map[string][]*pb.PolicyUpdate)
	var latestVersion string
	var updatedCount int

	for {
		update, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("receive policy update failed: %w", err)
		}

		p.logger.Debug("Received policy update",
			zap.String("policy_id", update.PolicyId),
			zap.String("policy_type", update.PolicyType),
			zap.String("version", update.Version),
			zap.Int32("chunk_index", update.ChunkIndex),
			zap.Int32("total_chunks", update.TotalChunks),
			zap.Bool("is_complete", update.IsComplete),
		)

		// 收集分块
		policyChunks[update.PolicyId] = append(policyChunks[update.PolicyId], update)

		// 当一个策略完整接收后，立即应用
		if update.IsComplete {
			chunks := policyChunks[update.PolicyId]

			// 合并分块内容
			mergedUpdate, err := p.mergePolicyChunks(chunks)
			if err != nil {
				p.logger.Error("Failed to merge policy chunks",
					zap.String("policy_id", update.PolicyId),
					zap.Error(err),
				)
				continue
			}

			// 验证校验和
			if p.config.ChecksumEnabled && mergedUpdate.Checksum != "" {
				if err := p.verifyPolicyChecksum(mergedUpdate); err != nil {
					p.logger.Error("Policy checksum verification failed",
						zap.String("policy_id", update.PolicyId),
						zap.Error(err),
					)
					continue
				}
			}

			// 应用策略
			if err := p.applyPolicyUpdate(ctx, mergedUpdate); err != nil {
				p.logger.Error("Failed to apply policy",
					zap.String("policy_id", update.PolicyId),
					zap.Error(err),
				)
				continue
			}

			updatedCount++
			if update.Version > latestVersion {
				latestVersion = update.Version
			}

			// 清理已处理的分块
			delete(policyChunks, update.PolicyId)
		}
	}

	// 更新版本
	if latestVersion != "" {
		p.mu.Lock()
		p.currentVersion = latestVersion
		p.lastSyncTime = time.Now()
		p.mu.Unlock()
	}

	p.logger.Info("Policy sync completed",
		zap.Int("policies_updated", updatedCount),
		zap.String("latest_version", latestVersion),
	)

	return nil
}

// mergePolicyChunks 合并策略分块
func (p *PolicyClient) mergePolicyChunks(chunks []*pb.PolicyUpdate) (*pb.PolicyUpdate, error) {
	if len(chunks) == 0 {
		return nil, fmt.Errorf("no chunks to merge")
	}

	// 单块策略
	if len(chunks) == 1 {
		return chunks[0], nil
	}

	// 按 chunk_index 排序
	for i := 0; i < len(chunks)-1; i++ {
		for j := i + 1; j < len(chunks); j++ {
			if chunks[i].ChunkIndex > chunks[j].ChunkIndex {
				chunks[i], chunks[j] = chunks[j], chunks[i]
			}
		}
	}

	// 合并内容
	var totalSize int
	for _, chunk := range chunks {
		totalSize += len(chunk.Content)
	}

	mergedContent := make([]byte, 0, totalSize)
	for _, chunk := range chunks {
		mergedContent = append(mergedContent, chunk.Content...)
	}

	// 返回合并后的策略（使用最后一个分块的元数据）
	last := chunks[len(chunks)-1]
	return &pb.PolicyUpdate{
		PolicyId:    last.PolicyId,
		PolicyType:  last.PolicyType,
		Version:     last.Version,
		Content:     mergedContent,
		ContentType: last.ContentType,
		Checksum:    last.Checksum,
		Action:      last.Action,
		IsComplete:  true,
	}, nil
}

// verifyPolicyChecksum 验证策略内容的 SHA256 校验和
func (p *PolicyClient) verifyPolicyChecksum(update *pb.PolicyUpdate) error {
	hash := sha256.New()
	if _, err := hash.Write(update.Content); err != nil {
		return err
	}
	calculated := hex.EncodeToString(hash.Sum(nil))

	if calculated != update.Checksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", update.Checksum, calculated)
	}

	return nil
}

// applyPolicyUpdate 应用单个策略更新
func (p *PolicyClient) applyPolicyUpdate(ctx context.Context, update *pb.PolicyUpdate) error {
	p.mu.RLock()
	applier := p.applier
	p.mu.RUnlock()

	if applier == nil {
		p.logger.Warn("No policy applier set, skipping policy application",
			zap.String("policy_id", update.PolicyId),
		)
		return nil
	}

	return applier.ApplyPolicy(ctx, update)
}

// CurrentVersion 返回当前策略版本
func (p *PolicyClient) CurrentVersion() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.currentVersion
}

// LastSyncTime 返回最后同步时间
func (p *PolicyClient) LastSyncTime() time.Time {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.lastSyncTime
}

// IsRunning 返回是否正在运行
func (p *PolicyClient) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.isRunning
}

// isRetryableError 判断错误是否可重试
func isRetryableError(err error) bool {
	st, ok := status.FromError(err)
	if !ok {
		return true // 非 gRPC 错误默认重试
	}

	switch st.Code() {
	case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted, codes.Aborted:
		return true
	case codes.InvalidArgument, codes.NotFound, codes.PermissionDenied, codes.Unauthenticated:
		return false
	default:
		return true
	}
}
