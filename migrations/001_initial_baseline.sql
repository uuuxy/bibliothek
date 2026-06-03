-- =============================================================================
-- Migration 001: Initial baseline
-- =============================================================================
-- The initial schema (tables, indexes, triggers) was bootstrapped via schema.sql
-- which is mounted into the Postgres container as an init script
-- (docker-entrypoint-initdb.d). This migration exists purely to anchor the
-- version sequence at 001 and to ensure schema_migrations is created before
-- the runner attempts to record subsequent versions.
--
-- Safe to re-run: CREATE TABLE IF NOT EXISTS is idempotent.
-- =============================================================================

CREATE TABLE IF NOT EXISTS schema_migrations (
    version     VARCHAR(255) PRIMARY KEY,
    applied_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);
