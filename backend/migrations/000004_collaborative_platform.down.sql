-- Migration: 000004_collaborative_platform.down.sql
-- Reverts collaborative platform changes

DROP TABLE IF EXISTS environment_events CASCADE;
DROP TABLE IF EXISTS change_requests CASCADE;
DROP TABLE IF EXISTS images CASCADE;
DROP TABLE IF EXISTS builds CASCADE;
DROP TABLE IF EXISTS build_plans CASCADE;
DROP TABLE IF EXISTS environment_services CASCADE;
DROP TABLE IF EXISTS environments CASCADE;
DROP TABLE IF EXISTS workspace_members CASCADE;
DROP TABLE IF EXISTS workspaces CASCADE;
DROP TABLE IF EXISTS runtime_profiles CASCADE;
DROP TABLE IF EXISTS share_links CASCADE;