-- ============================================================
-- EDR Platform - Tenant Tables Rollback
-- Migration: 000003_create_tenant_tables.down.sql
-- ============================================================

DROP TRIGGER IF EXISTS trigger_tenants_updated_at ON tenants;
DROP INDEX IF EXISTS idx_tenants_status;
DROP INDEX IF EXISTS idx_tenants_name;
DROP TABLE IF EXISTS tenants;
-- 注意：不删除 update_updated_at 函数，因为可能被其他表使用
