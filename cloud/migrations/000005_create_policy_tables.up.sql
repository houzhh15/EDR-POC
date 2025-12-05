-- ============================================================
-- EDR Platform - Policy Tables
-- Migration: 000005_create_policy_tables.up.sql
-- Description: 创建策略管理表
-- ============================================================

-- 策略表
CREATE TABLE IF NOT EXISTS policies (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name            VARCHAR(255) NOT NULL,
    description     TEXT,
    type            VARCHAR(50) NOT NULL,
    priority        INT NOT NULL DEFAULT 50,
    enabled         BOOLEAN NOT NULL DEFAULT true,
    config          JSONB NOT NULL DEFAULT '{}',
    version         INT NOT NULL DEFAULT 1,
    created_by      UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_by      UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at      TIMESTAMPTZ
);

-- 租户内策略名唯一（仅未删除）
CREATE UNIQUE INDEX idx_policies_tenant_name 
    ON policies(tenant_id, name) WHERE deleted_at IS NULL;

-- 租户ID索引
CREATE INDEX idx_policies_tenant_id ON policies(tenant_id);

-- 类型索引
CREATE INDEX idx_policies_type ON policies(type);

-- 启用状态复合索引
CREATE INDEX idx_policies_enabled ON policies(tenant_id, enabled);

-- 软删除索引
CREATE INDEX idx_policies_deleted_at ON policies(deleted_at) WHERE deleted_at IS NOT NULL;

-- 更新时间触发器
CREATE TRIGGER trigger_policies_updated_at
    BEFORE UPDATE ON policies
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

-- 注释
COMMENT ON TABLE policies IS '策略配置表';
COMMENT ON COLUMN policies.type IS '策略类型: detection, response, compliance';
COMMENT ON COLUMN policies.priority IS '优先级: 1-100，数值越大优先级越高';
COMMENT ON COLUMN policies.config IS '策略配置，JSONB格式';
