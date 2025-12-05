-- ============================================================
-- EDR Platform - Alert Tables
-- Migration: 000006_create_alert_tables.up.sql
-- Description: 创建告警管理表
-- ============================================================

-- 告警表
CREATE TABLE IF NOT EXISTS alerts (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    asset_id            UUID REFERENCES assets(id) ON DELETE SET NULL,
    policy_id           UUID REFERENCES policies(id) ON DELETE SET NULL,
    rule_name           VARCHAR(255) NOT NULL,
    severity            VARCHAR(20) NOT NULL,
    title               VARCHAR(500) NOT NULL,
    description         TEXT,
    status              VARCHAR(20) NOT NULL DEFAULT 'open',
    source_event_ids    TEXT[],
    context             JSONB DEFAULT '{}',
    assigned_to         UUID REFERENCES users(id) ON DELETE SET NULL,
    acknowledged_at     TIMESTAMPTZ,
    acknowledged_by     UUID REFERENCES users(id) ON DELETE SET NULL,
    resolved_at         TIMESTAMPTZ,
    resolved_by         UUID REFERENCES users(id) ON DELETE SET NULL,
    resolution          TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 租户ID索引
CREATE INDEX idx_alerts_tenant_id ON alerts(tenant_id);

-- 状态索引
CREATE INDEX idx_alerts_status ON alerts(tenant_id, status);

-- 严重程度索引
CREATE INDEX idx_alerts_severity ON alerts(tenant_id, severity);

-- 创建时间索引（用于时间范围查询）
CREATE INDEX idx_alerts_created_at ON alerts(tenant_id, created_at DESC);

-- 资产关联索引
CREATE INDEX idx_alerts_asset_id ON alerts(asset_id) WHERE asset_id IS NOT NULL;

-- 策略关联索引
CREATE INDEX idx_alerts_policy_id ON alerts(policy_id) WHERE policy_id IS NOT NULL;

-- 复合索引：租户+状态+时间（常用查询）
CREATE INDEX idx_alerts_tenant_status_time 
    ON alerts(tenant_id, status, created_at DESC);

-- 更新时间触发器
CREATE TRIGGER trigger_alerts_updated_at
    BEFORE UPDATE ON alerts
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

-- 注释
COMMENT ON TABLE alerts IS '安全告警表';
COMMENT ON COLUMN alerts.severity IS '严重程度: critical, high, medium, low, info';
COMMENT ON COLUMN alerts.status IS '状态: open, acknowledged, in_progress, resolved, false_positive';
COMMENT ON COLUMN alerts.context IS '告警上下文，JSONB格式';
