-- ============================================================
-- EDR Platform - User Tables
-- Migration: 000004_create_user_tables.up.sql
-- Description: 创建用户管理表
-- ============================================================

-- 用户表
CREATE TABLE IF NOT EXISTS users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    username        VARCHAR(100) NOT NULL,
    email           VARCHAR(255) NOT NULL,
    password_hash   VARCHAR(255) NOT NULL,
    role            VARCHAR(20) NOT NULL DEFAULT 'viewer',
    status          VARCHAR(20) NOT NULL DEFAULT 'active',
    display_name    VARCHAR(255),
    phone           VARCHAR(50),
    last_login_at   TIMESTAMPTZ,
    last_login_ip   VARCHAR(45),
    login_count     INT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at      TIMESTAMPTZ
);

-- 租户内用户名唯一（仅未删除）
CREATE UNIQUE INDEX idx_users_tenant_username 
    ON users(tenant_id, username) WHERE deleted_at IS NULL;

-- 租户内邮箱唯一（仅未删除）
CREATE UNIQUE INDEX idx_users_tenant_email 
    ON users(tenant_id, email) WHERE deleted_at IS NULL;

-- 租户ID索引
CREATE INDEX idx_users_tenant_id ON users(tenant_id);

-- 软删除索引
CREATE INDEX idx_users_deleted_at ON users(deleted_at) WHERE deleted_at IS NOT NULL;

-- 更新时间触发器
CREATE TRIGGER trigger_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

-- 注释
COMMENT ON TABLE users IS '用户信息表';
COMMENT ON COLUMN users.role IS '角色: admin, operator, viewer';
COMMENT ON COLUMN users.status IS '状态: active, inactive, locked';
COMMENT ON COLUMN users.password_hash IS 'bcrypt 哈希密码';
