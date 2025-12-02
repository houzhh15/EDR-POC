// Package integration 提供资产管理模块的集成测试
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/houzhh15/EDR-POC/cloud/internal/asset"
)

// mockAgentStatusManager 模拟状态管理器
type mockAgentStatusManager struct {
	agents map[string]bool
}

func (m *mockAgentStatusManager) UpdateHeartbeat(ctx context.Context, agentID, tenantID string, info *asset.HeartbeatInfo) error {
	m.agents[agentID] = true
	return nil
}

func (m *mockAgentStatusManager) IsOnline(ctx context.Context, agentID string) (bool, error) {
	return m.agents[agentID], nil
}

func (m *mockAgentStatusManager) GetStatus(ctx context.Context, agentID string) (*asset.AgentStatus, error) {
	return nil, nil
}

func (m *mockAgentStatusManager) ListOnlineAgents(ctx context.Context, tenantID string) ([]string, error) {
	var agents []string
	for id, online := range m.agents {
		if online {
			agents = append(agents, id)
		}
	}
	return agents, nil
}

func (m *mockAgentStatusManager) CountOnlineAgents(ctx context.Context, tenantID string) (int64, error) {
	count := int64(0)
	for _, online := range m.agents {
		if online {
			count++
		}
	}
	return count, nil
}

// testContext 测试上下文
type testContext struct {
	db              *gorm.DB
	redis           *redis.Client
	logger          *zap.Logger
	router          *gin.Engine
	assetService    *asset.AssetService
	groupService    *asset.GroupService
	softwareService *asset.SoftwareService
	changeLogger    *asset.ChangeLogger
	statusManager   asset.AgentStatusManager
	handler         *asset.AssetHandler
	tenantA         uuid.UUID
	tenantB         uuid.UUID
}

// setupTestContext 设置测试上下文
func setupTestContext(t *testing.T) *testContext {
	t.Helper()

	logger, _ := zap.NewDevelopment()

	// 使用内存 SQLite 数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 自动迁移
	err = db.AutoMigrate(
		&asset.Asset{},
		&asset.AssetGroup{},
		&asset.AssetGroupMember{},
		&asset.SoftwareInventory{},
		&asset.AssetChangeLog{},
	)
	require.NoError(t, err)

	// 检查是否有真实 Redis 可用
	redisAddr := os.Getenv("TEST_REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:16379"
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	// 测试 Redis 连接
	ctx := context.Background()
	var statusManager asset.AgentStatusManager
	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Logf("Redis not available at %s, using mock status manager", redisAddr)
		statusManager = &mockAgentStatusManager{agents: make(map[string]bool)}
	} else {
		statusManager = asset.NewAgentStatusManager(redisClient, logger)
	}

	// 初始化服务
	repo := asset.NewAssetRepository(db, logger)
	changeLogger := asset.NewChangeLogger(db, logger)

	assetService := asset.NewAssetService(repo, statusManager, changeLogger, logger)
	groupService := asset.NewGroupService(db, logger)
	softwareService := asset.NewSoftwareService(db, logger)

	handler := asset.NewAssetHandler(assetService, groupService, softwareService, changeLogger, logger)

	// 设置 Gin 路由
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 添加租户中间件（模拟 JWT 提取）
	router.Use(func(c *gin.Context) {
		tenantID := c.GetHeader("X-Tenant-ID")
		if tenantID != "" {
			c.Set("tenant_id", tenantID)
		}
		c.Next()
	})

	v1 := router.Group("/api/v1")
	handler.RegisterRoutes(v1)

	return &testContext{
		db:              db,
		redis:           redisClient,
		logger:          logger,
		router:          router,
		assetService:    assetService,
		groupService:    groupService,
		softwareService: softwareService,
		changeLogger:    changeLogger,
		statusManager:   statusManager,
		handler:         handler,
		tenantA:         uuid.New(),
		tenantB:         uuid.New(),
	}
}

// cleanup 清理测试资源
func (tc *testContext) cleanup() {
	if tc.redis != nil {
		tc.redis.Close()
	}
}

// TestAssetIntegration_FullFlow 测试完整的资产管理流程
func TestAssetIntegration_FullFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tc := setupTestContext(t)
	defer tc.cleanup()

	ctx := context.Background()

	// Step 1: 注册新资产
	t.Run("RegisterAsset", func(t *testing.T) {
		req := &asset.RegisterAssetRequest{
			AgentID:      "agent-001",
			TenantID:     tc.tenantA.String(),
			Hostname:     "workstation-001",
			OSType:       "windows",
			OSVersion:    "Windows 10 Pro",
			Architecture: "amd64",
			IPAddresses:  []string{"192.168.1.100"},
			MACAddresses: []string{"00:11:22:33:44:55"},
			AgentVersion: "1.0.0",
		}

		a, err := tc.assetService.RegisterOrUpdateAsset(ctx, req)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, a.ID)
		assert.Equal(t, "workstation-001", a.Hostname)
		assert.Equal(t, asset.AssetStatusOnline, a.Status)
	})

	// Step 2: 更新资产（模拟心跳+变更检测）
	t.Run("UpdateAssetWithChanges", func(t *testing.T) {
		req := &asset.RegisterAssetRequest{
			AgentID:      "agent-001",
			TenantID:     tc.tenantA.String(),
			Hostname:     "workstation-001-renamed",
			OSType:       "windows",
			OSVersion:    "Windows 11 Pro",
			Architecture: "amd64",
			IPAddresses:  []string{"192.168.1.101"},
			MACAddresses: []string{"00:11:22:33:44:55"},
			AgentVersion: "1.0.1",
		}

		a, err := tc.assetService.RegisterOrUpdateAsset(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, "workstation-001-renamed", a.Hostname)
		assert.Equal(t, "Windows 11 Pro", a.OSVersion)
	})

	// Step 3: 验证变更日志
	t.Run("VerifyChangeLog", func(t *testing.T) {
		a, err := tc.assetService.GetAssetByAgentID(ctx, tc.tenantA, "agent-001")
		require.NoError(t, err)

		opts := &asset.ChangeLogQueryOptions{
			Pagination: asset.Pagination{Page: 1, PageSize: 100},
		}
		changes, total, err := tc.changeLogger.GetChangeHistory(ctx, a.ID, opts)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, int64(4))
		assert.NotEmpty(t, changes)
	})

	// Step 4: 创建分组
	var groupID uuid.UUID
	t.Run("CreateGroup", func(t *testing.T) {
		req := &asset.CreateGroupRequest{
			Name:        "Development",
			Description: "Development workstations",
		}
		group, err := tc.groupService.CreateGroup(ctx, tc.tenantA, req)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, group.ID)
		groupID = group.ID
	})

	// Step 5: 将资产分配到分组
	t.Run("AssignAssetToGroup", func(t *testing.T) {
		a, err := tc.assetService.GetAssetByAgentID(ctx, tc.tenantA, "agent-001")
		require.NoError(t, err)

		err = tc.groupService.AssignAsset(ctx, tc.tenantA, groupID, a.ID)
		require.NoError(t, err)

		opts := &asset.QueryOptions{
			Pagination: asset.Pagination{Page: 1, PageSize: 10},
		}
		assets, total, err := tc.groupService.GetAssetsByGroup(ctx, tc.tenantA, groupID, opts)
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, assets, 1)
	})

	// Step 6: 更新软件清单
	t.Run("UpdateSoftwareInventory", func(t *testing.T) {
		a, err := tc.assetService.GetAssetByAgentID(ctx, tc.tenantA, "agent-001")
		require.NoError(t, err)

		items := []asset.SoftwareItem{
			{Name: "Visual Studio Code", Version: "1.85.0", Publisher: "Microsoft"},
			{Name: "Google Chrome", Version: "120.0.0", Publisher: "Google"},
			{Name: "Docker Desktop", Version: "4.25.0", Publisher: "Docker Inc"},
		}

		err = tc.softwareService.UpdateSoftwareInventory(ctx, a.ID, items)
		require.NoError(t, err)

		opts := &asset.SoftwareQueryOptions{
			Pagination: asset.Pagination{Page: 1, PageSize: 50},
		}
		software, total, err := tc.softwareService.GetSoftwareByAsset(ctx, a.ID, opts)
		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		assert.Len(t, software, 3)
	})
}

// TestAssetIntegration_MultiTenantIsolation 测试多租户隔离
func TestAssetIntegration_MultiTenantIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tc := setupTestContext(t)
	defer tc.cleanup()

	ctx := context.Background()

	// 为租户A创建资产
	reqA := &asset.RegisterAssetRequest{
		AgentID:      "tenant-a-agent-001",
		TenantID:     tc.tenantA.String(),
		Hostname:     "tenant-a-host",
		OSType:       "linux",
		OSVersion:    "Ubuntu 22.04",
		Architecture: "amd64",
		AgentVersion: "1.0.0",
	}
	_, err := tc.assetService.RegisterOrUpdateAsset(ctx, reqA)
	require.NoError(t, err)

	// 为租户B创建资产
	reqB := &asset.RegisterAssetRequest{
		AgentID:      "tenant-b-agent-001",
		TenantID:     tc.tenantB.String(),
		Hostname:     "tenant-b-host",
		OSType:       "windows",
		OSVersion:    "Windows Server 2022",
		Architecture: "amd64",
		AgentVersion: "1.0.0",
	}
	_, err = tc.assetService.RegisterOrUpdateAsset(ctx, reqB)
	require.NoError(t, err)

	// 验证租户A只能看到自己的资产
	t.Run("TenantA_CanOnlySeeOwnAssets", func(t *testing.T) {
		opts := &asset.QueryOptions{
			Pagination: asset.Pagination{Page: 1, PageSize: 100},
		}
		assets, total, err := tc.assetService.ListAssets(ctx, tc.tenantA, opts)
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, assets, 1)
		assert.Equal(t, "tenant-a-host", assets[0].Hostname)
	})

	// 验证租户B只能看到自己的资产
	t.Run("TenantB_CanOnlySeeOwnAssets", func(t *testing.T) {
		opts := &asset.QueryOptions{
			Pagination: asset.Pagination{Page: 1, PageSize: 100},
		}
		assets, total, err := tc.assetService.ListAssets(ctx, tc.tenantB, opts)
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, assets, 1)
		assert.Equal(t, "tenant-b-host", assets[0].Hostname)
	})

	// 验证租户A无法访问租户B的资产
	t.Run("TenantA_CannotAccessTenantB_Asset", func(t *testing.T) {
		assetB, err := tc.assetService.GetAssetByAgentID(ctx, tc.tenantB, "tenant-b-agent-001")
		require.NoError(t, err)

		_, err = tc.assetService.GetAsset(ctx, tc.tenantA, assetB.ID)
		assert.Error(t, err)
	})
}

// TestAssetIntegration_RESTAPI 测试 REST API 端点
func TestAssetIntegration_RESTAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tc := setupTestContext(t)
	defer tc.cleanup()

	ctx := context.Background()
	tenantID := tc.tenantA

	// 先创建测试资产
	req := &asset.RegisterAssetRequest{
		AgentID:      "api-test-agent",
		TenantID:     tenantID.String(),
		Hostname:     "api-test-host",
		OSType:       "linux",
		OSVersion:    "Ubuntu 22.04",
		Architecture: "amd64",
		AgentVersion: "1.0.0",
	}
	createdAsset, err := tc.assetService.RegisterOrUpdateAsset(ctx, req)
	require.NoError(t, err)

	// 测试 GET /api/v1/assets
	t.Run("GET_Assets_List", func(t *testing.T) {
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/api/v1/assets?page=1&page_size=10", nil)
		httpReq.Header.Set("X-Tenant-ID", tenantID.String())
		tc.router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp asset.AssetListResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(resp.Data), 1)
	})

	// 测试 GET /api/v1/assets/:id
	t.Run("GET_Asset_ByID", func(t *testing.T) {
		w := httptest.NewRecorder()
		url := fmt.Sprintf("/api/v1/assets/%s", createdAsset.ID.String())
		httpReq, _ := http.NewRequest("GET", url, nil)
		httpReq.Header.Set("X-Tenant-ID", tenantID.String())
		tc.router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	// 测试 GET /api/v1/asset-groups
	t.Run("GET_Groups_Tree", func(t *testing.T) {
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/api/v1/asset-groups", nil)
		httpReq.Header.Set("X-Tenant-ID", tenantID.String())
		tc.router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	// 测试 POST /api/v1/asset-groups
	t.Run("POST_CreateGroup", func(t *testing.T) {
		body := map[string]string{
			"name":        "Test Group",
			"description": "Created via API test",
		}
		jsonBody, _ := json.Marshal(body)

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/asset-groups", bytes.NewBuffer(jsonBody))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("X-Tenant-ID", tenantID.String())
		tc.router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	// 测试 401 - 缺少租户ID
	t.Run("GET_Assets_Unauthorized", func(t *testing.T) {
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/api/v1/assets", nil)
		tc.router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	// 测试 404 - 资产不存在
	t.Run("GET_Asset_NotFound", func(t *testing.T) {
		w := httptest.NewRecorder()
		url := fmt.Sprintf("/api/v1/assets/%s", uuid.New().String())
		httpReq, _ := http.NewRequest("GET", url, nil)
		httpReq.Header.Set("X-Tenant-ID", tenantID.String())
		tc.router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// TestAssetIntegration_GroupHierarchy 测试分组层级
func TestAssetIntegration_GroupHierarchy(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tc := setupTestContext(t)
	defer tc.cleanup()

	ctx := context.Background()
	tenantID := tc.tenantA

	// 创建根分组
	root, err := tc.groupService.CreateGroup(ctx, tenantID, &asset.CreateGroupRequest{
		Name:        "Root",
		Description: "Root group",
	})
	require.NoError(t, err)
	assert.Equal(t, 0, root.Level) // 根分组 level=0

	// 创建子分组
	child, err := tc.groupService.CreateGroup(ctx, tenantID, &asset.CreateGroupRequest{
		Name:        "Child",
		Description: "Child group",
		ParentID:    &root.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, child.Level) // 子分组 level=1

	// 获取分组树
	tree, err := tc.groupService.GetGroupTree(ctx, tenantID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(tree), 1)

	// 测试删除有子分组的分组应该失败
	err = tc.groupService.DeleteGroup(ctx, tenantID, root.ID)
	assert.Error(t, err)

	// 删除子分组后可以删除父分组
	err = tc.groupService.DeleteGroup(ctx, tenantID, child.ID)
	require.NoError(t, err)

	err = tc.groupService.DeleteGroup(ctx, tenantID, root.ID)
	require.NoError(t, err)
}
