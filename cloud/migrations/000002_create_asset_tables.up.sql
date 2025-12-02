-- ============================================================
-- EDR Platform - Asset Management Tables
-- Migration: 000002_create_asset_tables.up.sql
-- Description: 创建资产管理相关的5张核心表
-- ============================================================

-- 1. assets 表 - 资产主表
CREATE TABLE IF NOT EXISTS assets (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id        VARCHAR(64) NOT NULL,
    tenant_id       UUID NOT NULL,
    hostname        VARCHAR(255) NOT NULL,
    os_type         VARCHAR(32) NOT NULL,
    os_version      VARCHAR(128),
    architecture    VARCHAR(32),
    ip_addresses    TEXT[],
    mac_addresses   TEXT[],
    agent_version   VARCHAR(32),
    status          VARCHAR(16) NOT NULL DEFAULT 'unknown',
    last_seen_at    TIMESTAMP WITH TIME ZONE,
    first_seen_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMP WITH TIME ZONE
);

-- 唯一约束: 每个租户下 agent_id 唯一
CREATE UNIQUE INDEX IF NOT EXISTS uk_assets_tenant_agent 
    ON assets(tenant_id, agent_id) WHERE deleted_at IS NULL;

-- 常用查询索引
CREATE INDEX IF NOT EXISTS idx_assets_tenant_status 
    ON assets(tenant_id, status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_assets_hostname 
    ON assets(tenant_id, hostname) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_assets_last_seen 
    ON assets(last_seen_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_assets_deleted 
    ON assets(deleted_at) WHERE deleted_at IS NOT NULL;

-- 2. asset_groups 表 - 资产分组（支持树形结构）
CREATE TABLE IF NOT EXISTS asset_groups (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    name            VARCHAR(128) NOT NULL,
    description     TEXT,
    type            VARCHAR(32) NOT NULL DEFAULT 'custom',
    parent_id       UUID REFERENCES asset_groups(id) ON DELETE CASCADE,
    path            VARCHAR(1024) NOT NULL DEFAULT '/',
    level           INTEGER NOT NULL DEFAULT 1,
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 唯一约束: 同一父分组下名称唯一
CREATE UNIQUE INDEX IF NOT EXISTS uk_groups_tenant_parent_name 
    ON asset_groups(tenant_id, COALESCE(parent_id, '00000000-0000-0000-0000-000000000000'::UUID), name);

-- 物化路径索引，加速树形查询
CREATE INDEX IF NOT EXISTS idx_groups_path 
    ON asset_groups(tenant_id, path);
CREATE INDEX IF NOT EXISTS idx_groups_parent 
    ON asset_groups(parent_id);

-- 3. asset_group_members 表 - 资产分组关联（多对多）
CREATE TABLE IF NOT EXISTS asset_group_members (
    asset_id        UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    group_id        UUID NOT NULL REFERENCES asset_groups(id) ON DELETE CASCADE,
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (asset_id, group_id)
);

CREATE INDEX IF NOT EXISTS idx_group_members_group 
    ON asset_group_members(group_id);
CREATE INDEX IF NOT EXISTS idx_group_members_asset 
    ON asset_group_members(asset_id);

-- 4. software_inventory 表 - 软件清单
CREATE TABLE IF NOT EXISTS software_inventory (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_id        UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    name            VARCHAR(512) NOT NULL,
    version         VARCHAR(128) NOT NULL,
    publisher       VARCHAR(256),
    install_date    TIMESTAMP WITH TIME ZONE,
    install_path    VARCHAR(1024),
    size            BIGINT DEFAULT 0,
    updated_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 唯一约束: 每个资产的软件名+版本唯一
CREATE UNIQUE INDEX IF NOT EXISTS uk_software_asset_name_ver 
    ON software_inventory(asset_id, name, version);

-- 软件名索引，支持跨资产搜索
CREATE INDEX IF NOT EXISTS idx_software_name 
    ON software_inventory(LOWER(name) varchar_pattern_ops);
CREATE INDEX IF NOT EXISTS idx_software_asset 
    ON software_inventory(asset_id);

-- 5. asset_change_logs 表 - 资产变更日志
CREATE TABLE IF NOT EXISTS asset_change_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_id        UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    field_name      VARCHAR(64) NOT NULL,
    old_value       TEXT,
    new_value       TEXT,
    changed_by      VARCHAR(128),
    changed_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 变更历史查询索引
CREATE INDEX IF NOT EXISTS idx_change_logs_asset 
    ON asset_change_logs(asset_id);
CREATE INDEX IF NOT EXISTS idx_change_logs_time 
    ON asset_change_logs(changed_at DESC);
CREATE INDEX IF NOT EXISTS idx_change_logs_asset_time 
    ON asset_change_logs(asset_id, changed_at DESC);

-- ============================================================
-- 触发器: 自动更新 updated_at
-- ============================================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_assets_updated_at
    BEFORE UPDATE ON assets
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_asset_groups_updated_at
    BEFORE UPDATE ON asset_groups
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
