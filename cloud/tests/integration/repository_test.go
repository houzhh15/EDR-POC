//go:build integration
// +build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/houzhh15/EDR-POC/cloud/internal/repository"
	"github.com/houzhh15/EDR-POC/cloud/internal/repository/models"
	"github.com/houzhh15/EDR-POC/cloud/pkg/database"
)

type RepositoryIntegrationSuite struct {
	suite.Suite
	container  testcontainers.Container
	db         *gorm.DB
	logger     *zap.Logger
	tenantRepo repository.TenantRepository
	userRepo   repository.UserRepository
	policyRepo repository.PolicyRepository
	alertRepo  repository.AlertRepository
	testTenant *models.Tenant
}

func (s *RepositoryIntegrationSuite) SetupSuite() {
	ctx := context.Background()
	s.logger = zap.NewNop()

	// Start PostgreSQL container
	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "edr_test",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(s.T(), err)
	s.container = container

	host, err := container.Host(ctx)
	require.NoError(s.T(), err)

	port, err := container.MappedPort(ctx, "5432")
	require.NoError(s.T(), err)

	dsn := fmt.Sprintf("host=%s port=%s user=test password=test dbname=edr_test sslmode=disable",
		host, port.Port())

	// Connect to database
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		SkipDefaultTransaction: true,
	})
	require.NoError(s.T(), err)
	s.db = db

	// Run migrations
	migrator, err := database.NewMigrator(dsn, "file://../../migrations")
	require.NoError(s.T(), err)
	err = migrator.Up()
	require.NoError(s.T(), err)

	// Initialize repositories
	s.tenantRepo = repository.NewTenantRepository(db, s.logger)
	s.userRepo = repository.NewUserRepository(db, s.logger)
	s.policyRepo = repository.NewPolicyRepository(db, s.logger)
	s.alertRepo = repository.NewAlertRepository(db, s.logger)
}

func (s *RepositoryIntegrationSuite) TearDownSuite() {
	if s.container != nil {
		s.container.Terminate(context.Background())
	}
}

func (s *RepositoryIntegrationSuite) SetupTest() {
	// Create test tenant for each test
	s.testTenant = &models.Tenant{
		ID:          uuid.New(),
		Name:        fmt.Sprintf("test-tenant-%d", time.Now().UnixNano()),
		DisplayName: "Integration Test Tenant",
		Status:      models.TenantStatusActive,
		MaxUsers:    100,
		MaxAssets:   1000,
		MaxPolicies: 50,
	}
	err := s.tenantRepo.Create(context.Background(), s.testTenant)
	require.NoError(s.T(), err)
}

func (s *RepositoryIntegrationSuite) TearDownTest() {
	// Cleanup test data
	if s.testTenant != nil {
		s.db.Exec("DELETE FROM alerts WHERE tenant_id = ?", s.testTenant.ID)
		s.db.Exec("DELETE FROM policies WHERE tenant_id = ?", s.testTenant.ID)
		s.db.Exec("DELETE FROM users WHERE tenant_id = ?", s.testTenant.ID)
		s.db.Exec("DELETE FROM tenants WHERE id = ?", s.testTenant.ID)
	}
}

func (s *RepositoryIntegrationSuite) TestTenantCRUD() {
	ctx := context.Background()

	// Test FindByID
	found, err := s.tenantRepo.FindByID(ctx, s.testTenant.ID)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), found)
	assert.Equal(s.T(), s.testTenant.Name, found.Name)

	// Test FindByName
	found, err = s.tenantRepo.FindByName(ctx, s.testTenant.Name)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), found)

	// Test UpdateStatus
	err = s.tenantRepo.UpdateStatus(ctx, s.testTenant.ID, models.TenantStatusSuspended)
	assert.NoError(s.T(), err)

	found, _ = s.tenantRepo.FindByID(ctx, s.testTenant.ID)
	assert.Equal(s.T(), models.TenantStatusSuspended, found.Status)

	// Test FindAll
	tenants, total, err := s.tenantRepo.FindAll(ctx, models.ListOptions{Limit: 10})
	assert.NoError(s.T(), err)
	assert.GreaterOrEqual(s.T(), total, int64(1))
	assert.NotEmpty(s.T(), tenants)
}

func (s *RepositoryIntegrationSuite) TestUserCRUD() {
	ctx := context.Background()

	// Create user
	user := &models.User{
		ID:           uuid.New(),
		TenantID:     s.testTenant.ID,
		Username:     "integration-test-user",
		Email:        "integration@test.com",
		PasswordHash: "hashed_password",
		DisplayName:  "Integration Test User",
		Role:         models.UserRoleUser,
		Status:       models.UserStatusActive,
	}
	err := s.userRepo.Create(ctx, user)
	assert.NoError(s.T(), err)

	// Test FindByID with tenant scope
	found, err := s.userRepo.FindByID(ctx, s.testTenant.ID, user.ID)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), found)
	assert.Equal(s.T(), user.Username, found.Username)

	// Test FindByUsername
	found, err = s.userRepo.FindByUsername(ctx, s.testTenant.ID, user.Username)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), found)

	// Test tenant isolation
	otherTenantID := uuid.New()
	found, err = s.userRepo.FindByID(ctx, otherTenantID, user.ID)
	assert.NoError(s.T(), err)
	assert.Nil(s.T(), found) // Should not find user in different tenant

	// Test UpdateLastLogin
	err = s.userRepo.UpdateLastLogin(ctx, s.testTenant.ID, user.ID, "192.168.1.1")
	assert.NoError(s.T(), err)

	found, _ = s.userRepo.FindByID(ctx, s.testTenant.ID, user.ID)
	assert.NotNil(s.T(), found.LastLoginAt)
	assert.Equal(s.T(), "192.168.1.1", found.LastLoginIP)
	assert.Equal(s.T(), 1, found.LoginCount)

	// Test soft delete
	err = s.userRepo.Delete(ctx, s.testTenant.ID, user.ID)
	assert.NoError(s.T(), err)

	// Should not find soft-deleted user
	found, err = s.userRepo.FindByID(ctx, s.testTenant.ID, user.ID)
	assert.NoError(s.T(), err)
	assert.Nil(s.T(), found)
}

func (s *RepositoryIntegrationSuite) TestPolicyCRUD() {
	ctx := context.Background()

	// Create policy
	policy := &models.Policy{
		ID:          uuid.New(),
		TenantID:    s.testTenant.ID,
		Name:        "Integration Test Policy",
		Description: "Test policy for integration tests",
		Type:        models.PolicyTypeDetection,
		Priority:    10,
		Enabled:     true,
		Config: models.PolicyConfig{
			Rules: []string{"rule1", "rule2"},
		},
	}
	err := s.policyRepo.Create(ctx, policy)
	assert.NoError(s.T(), err)

	// Test FindByID
	found, err := s.policyRepo.FindByID(ctx, s.testTenant.ID, policy.ID)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), found)
	assert.Equal(s.T(), policy.Name, found.Name)

	// Test FindEnabled
	policies, err := s.policyRepo.FindEnabled(ctx, s.testTenant.ID, models.PolicyTypeDetection)
	assert.NoError(s.T(), err)
	assert.NotEmpty(s.T(), policies)

	// Test SetEnabled
	err = s.policyRepo.SetEnabled(ctx, s.testTenant.ID, policy.ID, false)
	assert.NoError(s.T(), err)

	policies, _ = s.policyRepo.FindEnabled(ctx, s.testTenant.ID, models.PolicyTypeDetection)
	assert.Empty(s.T(), policies) // Disabled policy should not appear

	// Test FindAll with filter
	enabled := true
	filter := repository.PolicyFilter{
		Type:    models.PolicyTypeDetection,
		Enabled: &enabled,
	}
	policies, total, err := s.policyRepo.FindAll(ctx, s.testTenant.ID, models.ListOptions{Limit: 10}, filter)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), int64(0), total) // All disabled
}

func (s *RepositoryIntegrationSuite) TestAlertCRUD() {
	ctx := context.Background()

	// Create alert
	alert := &models.Alert{
		ID:          uuid.New(),
		TenantID:    s.testTenant.ID,
		Title:       "Integration Test Alert",
		Description: "Test alert for integration tests",
		Severity:    models.AlertSeverityHigh,
		Status:      models.AlertStatusNew,
		Source:      "integration-test",
		Context: models.AlertContext{
			Process: &models.ProcessInfo{
				PID:  12345,
				Name: "test.exe",
			},
		},
	}
	err := s.alertRepo.Create(ctx, alert)
	assert.NoError(s.T(), err)

	// Test FindByID
	found, err := s.alertRepo.FindByID(ctx, s.testTenant.ID, alert.ID)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), found)
	assert.Equal(s.T(), alert.Title, found.Title)

	// Test Acknowledge
	userID := uuid.New()
	err = s.alertRepo.Acknowledge(ctx, s.testTenant.ID, alert.ID, userID)
	assert.NoError(s.T(), err)

	found, _ = s.alertRepo.FindByID(ctx, s.testTenant.ID, alert.ID)
	assert.Equal(s.T(), models.AlertStatusAcknowledged, found.Status)
	assert.NotNil(s.T(), found.AcknowledgedAt)
	assert.Equal(s.T(), userID, *found.AcknowledgedBy)

	// Test Resolve
	err = s.alertRepo.Resolve(ctx, s.testTenant.ID, alert.ID, userID, "Fixed the issue")
	assert.NoError(s.T(), err)

	found, _ = s.alertRepo.FindByID(ctx, s.testTenant.ID, alert.ID)
	assert.Equal(s.T(), models.AlertStatusResolved, found.Status)
	assert.NotNil(s.T(), found.ResolvedAt)
	assert.Equal(s.T(), "Fixed the issue", found.Resolution)

	// Test CountByStatus
	counts, err := s.alertRepo.CountByStatus(ctx, s.testTenant.ID)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), int64(1), counts[models.AlertStatusResolved])

	// Test CountBySeverity
	severityCounts, err := s.alertRepo.CountBySeverity(ctx, s.testTenant.ID)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), int64(1), severityCounts[models.AlertSeverityHigh])
}

func TestRepositoryIntegrationSuite(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}
	suite.Run(t, new(RepositoryIntegrationSuite))
}
