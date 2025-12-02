-- ============================================================
-- EDR Platform - Asset Management Tables
-- Migration: 000002_create_asset_tables.down.sql
-- Description: 回滚资产管理表
-- ============================================================

-- 删除触发器
DROP TRIGGER IF EXISTS update_assets_updated_at ON assets;
DROP TRIGGER IF EXISTS update_asset_groups_updated_at ON asset_groups;

-- 删除函数
DROP FUNCTION IF EXISTS update_updated_at_column();

-- 删除表（按外键依赖顺序）
DROP TABLE IF EXISTS asset_change_logs;
DROP TABLE IF EXISTS software_inventory;
DROP TABLE IF EXISTS asset_group_members;
DROP TABLE IF EXISTS asset_groups;
DROP TABLE IF EXISTS assets;
