-- ============================================================
-- EDR Platform - Policy Tables Rollback
-- Migration: 000005_create_policy_tables.down.sql
-- ============================================================

DROP TRIGGER IF EXISTS trigger_policies_updated_at ON policies;
DROP INDEX IF EXISTS idx_policies_deleted_at;
DROP INDEX IF EXISTS idx_policies_enabled;
DROP INDEX IF EXISTS idx_policies_type;
DROP INDEX IF EXISTS idx_policies_tenant_id;
DROP INDEX IF EXISTS idx_policies_tenant_name;
DROP TABLE IF EXISTS policies;
