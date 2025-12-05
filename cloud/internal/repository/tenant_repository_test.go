package repository

import (
"context"
"testing"
"time"

"github.com/DATA-DOG/go-sqlmock"
"github.com/google/uuid"
"github.com/stretchr/testify/assert"
"github.com/stretchr/testify/require"
"go.uber.org/zap"
"gorm.io/driver/postgres"
"gorm.io/gorm"

"github.com/houzhh15/EDR-POC/cloud/internal/repository/models"
)

func setupMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, func()) {
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)

	dialector := postgres.New(postgres.Config{
Conn:       sqlDB,
DriverName: "postgres",
})

	db, err := gorm.Open(dialector, &gorm.Config{
		SkipDefaultTransaction: true,
	})
	require.NoError(t, err)

	cleanup := func() {
		sqlDB.Close()
	}

	return db, mock, cleanup
}

func TestTenantRepository_Create(t *testing.T) {
	db, mock, cleanup := setupMockDB(t)
	defer cleanup()

	logger := zap.NewNop()
	repo := NewTenantRepository(db, logger)

	tenant := &models.Tenant{
		ID:          uuid.New(),
		Name:        "test-tenant",
		DisplayName: "Test Tenant",
		Status:      models.TenantStatusActive,
		MaxAgents:   100,
	}

	// GORM uses Query with RETURNING for PostgreSQL
	// settings needs to be []byte for JSONB
	rows := sqlmock.NewRows([]string{"id", "settings", "created_at", "updated_at"}).
		AddRow(tenant.ID, []byte("{}"), time.Now(), time.Now())

	mock.ExpectQuery(`INSERT INTO "tenants"`).
		WillReturnRows(rows)

	err := repo.Create(context.Background(), tenant)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTenantRepository_FindByID(t *testing.T) {
	db, mock, cleanup := setupMockDB(t)
	defer cleanup()

	logger := zap.NewNop()
	repo := NewTenantRepository(db, logger)

	tenantID := uuid.New()

	rows := sqlmock.NewRows([]string{
"id", "name", "display_name", "status",
"max_agents", "max_events_per_day", "contact_email", "contact_phone",
"settings", "created_at", "updated_at",
}).AddRow(
tenantID, "test-tenant", "Test Tenant", "active",
100, 10000000, "", "",
[]byte("{}"), time.Now(), time.Now(),
	)

	mock.ExpectQuery(`SELECT \* FROM "tenants" WHERE`).
		WillReturnRows(rows)

	tenant, err := repo.FindByID(context.Background(), tenantID)
	assert.NoError(t, err)
	assert.NotNil(t, tenant)
	assert.Equal(t, tenantID, tenant.ID)
	assert.Equal(t, "test-tenant", tenant.Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTenantRepository_FindByID_NotFound(t *testing.T) {
	db, mock, cleanup := setupMockDB(t)
	defer cleanup()

	logger := zap.NewNop()
	repo := NewTenantRepository(db, logger)

	tenantID := uuid.New()

	mock.ExpectQuery(`SELECT \* FROM "tenants" WHERE`).
		WillReturnError(gorm.ErrRecordNotFound)

	tenant, err := repo.FindByID(context.Background(), tenantID)
	assert.NoError(t, err) // 未找到不返回错误
	assert.Nil(t, tenant)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTenantRepository_FindAll(t *testing.T) {
	db, mock, cleanup := setupMockDB(t)
	defer cleanup()

	logger := zap.NewNop()
	repo := NewTenantRepository(db, logger)

	// Mock count query
	mock.ExpectQuery(`SELECT count\(\*\) FROM "tenants"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	// Mock select query
	rows := sqlmock.NewRows([]string{
"id", "name", "display_name", "status",
"max_agents", "max_events_per_day", "contact_email", "contact_phone",
"settings", "created_at", "updated_at",
}).
		AddRow(uuid.New(), "tenant1", "Tenant 1", "active", 100, 10000000, "", "", []byte("{}"), time.Now(), time.Now()).
		AddRow(uuid.New(), "tenant2", "Tenant 2", "active", 100, 10000000, "", "", []byte("{}"), time.Now(), time.Now())

	mock.ExpectQuery(`SELECT \* FROM "tenants"`).
		WillReturnRows(rows)

	opts := models.ListOptions{
		Limit:  10,
		Offset: 0,
	}
	tenants, total, err := repo.FindAll(context.Background(), opts)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, tenants, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTenantRepository_UpdateStatus(t *testing.T) {
	db, mock, cleanup := setupMockDB(t)
	defer cleanup()

	logger := zap.NewNop()
	repo := NewTenantRepository(db, logger)

	tenantID := uuid.New()

	mock.ExpectExec(`UPDATE "tenants" SET`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateStatus(context.Background(), tenantID, models.TenantStatusSuspended)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTenantRepository_Delete(t *testing.T) {
	db, mock, cleanup := setupMockDB(t)
	defer cleanup()

	logger := zap.NewNop()
	repo := NewTenantRepository(db, logger)

	tenantID := uuid.New()

	mock.ExpectExec(`DELETE FROM "tenants" WHERE`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Delete(context.Background(), tenantID)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_Create(t *testing.T) {
	db, mock, cleanup := setupMockDB(t)
	defer cleanup()

	logger := zap.NewNop()
	repo := NewUserRepository(db, logger)

	tenantID := uuid.New()
	user := &models.User{
		ID:           uuid.New(),
		TenantID:     tenantID,
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hashed_password",
		Role:         models.UserRoleViewer,
		Status:       models.UserStatusActive,
	}

	// GORM uses Query with RETURNING for PostgreSQL
	rows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
		AddRow(user.ID, time.Now(), time.Now())

	mock.ExpectQuery(`INSERT INTO "users"`).
		WillReturnRows(rows)

	err := repo.Create(context.Background(), user)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_FindByID_WithTenantScope(t *testing.T) {
	db, mock, cleanup := setupMockDB(t)
	defer cleanup()

	logger := zap.NewNop()
	repo := NewUserRepository(db, logger)

	tenantID := uuid.New()
	userID := uuid.New()

	rows := sqlmock.NewRows([]string{
"id", "tenant_id", "username", "email", "password_hash",
"role", "status", "display_name", "phone",
"last_login_at", "last_login_ip", "login_count",
"created_at", "updated_at", "deleted_at",
}).AddRow(
userID, tenantID, "testuser", "test@example.com", "hashed",
"viewer", "active", "Test User", "",
nil, "", 0,
time.Now(), time.Now(), nil,
	)

	mock.ExpectQuery(`SELECT \* FROM "users" WHERE`).
		WillReturnRows(rows)

	user, err := repo.FindByID(context.Background(), tenantID, userID)
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, userID, user.ID)
	assert.Equal(t, tenantID, user.TenantID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_SoftDelete(t *testing.T) {
	db, mock, cleanup := setupMockDB(t)
	defer cleanup()

	logger := zap.NewNop()
	repo := NewUserRepository(db, logger)

	tenantID := uuid.New()
	userID := uuid.New()

	mock.ExpectExec(`UPDATE "users" SET`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Delete(context.Background(), tenantID, userID)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAlertRepository_CountByStatus(t *testing.T) {
	db, mock, cleanup := setupMockDB(t)
	defer cleanup()

	logger := zap.NewNop()
	repo := NewAlertRepository(db, logger)

	tenantID := uuid.New()

	rows := sqlmock.NewRows([]string{"status", "count"}).
		AddRow("open", 10).
		AddRow("acknowledged", 5).
		AddRow("resolved", 20)

	mock.ExpectQuery(`SELECT status, COUNT\(\*\) as count FROM "alerts"`).
		WillReturnRows(rows)

	counts, err := repo.CountByStatus(context.Background(), tenantID)
	assert.NoError(t, err)
	assert.Equal(t, int64(10), counts[models.AlertStatusOpen])
	assert.Equal(t, int64(5), counts[models.AlertStatusAcknowledged])
	assert.Equal(t, int64(20), counts[models.AlertStatusResolved])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Test for Scopes
func TestTenantScope(t *testing.T) {
	tenantID := uuid.New()
	scope := TenantScope(tenantID)
	assert.NotNil(t, scope)
}

func TestNotDeletedScope(t *testing.T) {
	scope := NotDeletedScope()
	assert.NotNil(t, scope)
}

func TestPaginationScope(t *testing.T) {
	tests := []struct {
		name   string
		limit  int
		offset int
	}{
		{"normal", 10, 0},
		{"zero limit", 0, 0},
		{"negative limit", -1, 0},
		{"over max", 200, 0},
		{"negative offset", 10, -5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
scope := PaginationScope(tt.limit, tt.offset)
assert.NotNil(t, scope)
})
	}
}
