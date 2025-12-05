-- ============================================================
-- EDR Platform - Tenant Tables
-- Migration: 000003_create_tenant_tables.up.sql
-- Description: 创建租户管理表
-- ============================================================

-- 创建 update_updated_at 触发器函数（如不存在）
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 租户表
CREATE TABLE IF NOT EXISTS tenants (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                VARCHAR(100) NOT NULL,
    display_name        VARCHAR(255) NOT NULL,
    status              VARCHAR(20) NOT NULL DEFAULT 'active',
    max_agents          INT NOT NULL DEFAULT 100,
    max_events_per_day  BIGINT NOT NULL DEFAULT 10000000,
    contact_email       VARCHAR(255),
    contact_phone       VARCHAR(50),
    settings            JSONB DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 唯一索引：租户名称
CREATE UNIQUE INDEX idx_tenants_name ON tenants(name);

-- 状态索引
CREATE INDEX idx_tenants_status ON tenants(status);

-- 更新时间触发器
CREATE TRIGGER trigger_tenants_updated_at
    BEFORE UPDATE ON tenants
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

-- 注释
COMMENT ON TABLE tenants IS '租户信息表';
COMMENT ON COLUMN tenants.name IS '租户唯一标识符，用于API';
COMMENT ON COLUMN tenants.status IS '状态: active, suspended, deleted';
COMMENT ON COLUMN tenants.settings IS '扩展配置，JSONB格式';
