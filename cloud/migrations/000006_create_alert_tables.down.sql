-- ============================================================
-- EDR Platform - Alert Tables Rollback
-- Migration: 000006_create_alert_tables.down.sql
-- ============================================================

DROP TRIGGER IF EXISTS trigger_alerts_updated_at ON alerts;
DROP INDEX IF EXISTS idx_alerts_tenant_status_time;
DROP INDEX IF EXISTS idx_alerts_policy_id;
DROP INDEX IF EXISTS idx_alerts_asset_id;
DROP INDEX IF EXISTS idx_alerts_created_at;
DROP INDEX IF EXISTS idx_alerts_severity;
DROP INDEX IF EXISTS idx_alerts_status;
DROP INDEX IF EXISTS idx_alerts_tenant_id;
DROP TABLE IF EXISTS alerts;
