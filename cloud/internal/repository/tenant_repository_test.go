package repository

import (
	"context"
	"regexp"
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
	sqlDB, mock, err := sqlmock.New()
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
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "tenants"`)).
		WithArgs(
			sqlmock.AnyArg(), // id
			sqlmock.AnyArg(), // name
			sqlmock.AnyArg(), // display_name
			sqlmock.AnyArg(), // description
			sqlmock.AnyArg(), // status
			sqlmock.AnyArg(), // settings
			sqlmock.AnyArg(), // max_users
			sqlmock.AnyArg(), // max_assets
			sqlmock.AnyArg(), // max_policies
			sqlmock.AnyArg(), // contact_email
			sqlmock.AnyArg(), // contact_phone
			sqlmock.AnyArg(), // created_at
			sqlmock.AnyArg(), // updated_at
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

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
		"id", "name", "display_name", "description", "status",
		"settings", "max_users", "max_assets", "max_policies",
		"contact_email", "contact_phone", "created_at", "updated_at",
	}).AddRow(
		tenantID, "test-tenant", "Test Tenant", "", "active",
		nil, 100, 1000, 50, "", "", time.Now(), time.Now(),
	)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tenants" WHERE id = $1`)).
		WithArgs(tenantID).
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

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tenants" WHERE id = $1`)).
		WithArgs(tenantID).
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
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "tenants"`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	// Mock select query
	rows := sqlmock.NewRows([]string{
		"id", "name", "display_name", "description", "status",
		"settings", "max_users", "max_assets", "max_policies",
		"contact_email", "contact_phone", "created_at", "updated_at",
	}).
		AddRow(uuid.New(), "tenant1", "Tenant 1", "", "active", nil, 100, 1000, 50, "", "", time.Now(), time.Now()).
		AddRow(uuid.New(), "tenant2", "Tenant 2", "", "active", nil, 100, 1000, 50, "", "", time.Now(), time.Now())

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tenants"`)).
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

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "tenants" SET "status"=$1 WHERE id = $2`)).
		WithArgs(models.TenantStatusSuspended, tenantID).
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

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "tenants" WHERE id = $1`)).
		WithArgs(tenantID).
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
		Role:         models.UserRoleUser,
		Status:       models.UserStatusActive,
	}

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "users"`)).
		WillReturnResult(sqlmock.NewResult(0, 1))

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
		"role", "status", "display_name", "avatar_url",
		"last_login_at", "last_login_ip", "login_count",
		"created_at", "updated_at", "deleted_at",
	}).AddRow(
		userID, tenantID, "testuser", "test@example.com", "hashed",
		"user", "active", "Test User", "",
		nil, "", 0,
		time.Now(), time.Now(), nil,
	)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE tenant_id = $1 AND deleted_at IS NULL AND id = $2`)).
		WithArgs(tenantID, userID).
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

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "users" SET "deleted_at"=$1 WHERE tenant_id = $2 AND id = $3`)).
		WithArgs(sqlmock.AnyArg(), tenantID, userID).
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
		AddRow("new", 10).
		AddRow("acknowledged", 5).
		AddRow("resolved", 20)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT status, COUNT(*) as count FROM "alerts" WHERE tenant_id = $1 GROUP BY "status"`)).
		WithArgs(tenantID).
		WillReturnRows(rows)

	counts, err := repo.CountByStatus(context.Background(), tenantID)
	assert.NoError(t, err)
	assert.Equal(t, int64(10), counts[models.AlertStatusNew])
	assert.Equal(t, int64(5), counts[models.AlertStatusAcknowledged])
	assert.Equal(t, int64(20), counts[models.AlertStatusResolved])
	assert.NoError(t, mock.ExpectationsWereMet())
}
