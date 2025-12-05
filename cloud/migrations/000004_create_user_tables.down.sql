-- ============================================================
-- EDR Platform - User Tables Rollback
-- Migration: 000004_create_user_tables.down.sql
-- ============================================================

DROP TRIGGER IF EXISTS trigger_users_updated_at ON users;
DROP INDEX IF EXISTS idx_users_deleted_at;
DROP INDEX IF EXISTS idx_users_tenant_id;
DROP INDEX IF EXISTS idx_users_tenant_email;
DROP INDEX IF EXISTS idx_users_tenant_username;
DROP TABLE IF EXISTS users;
