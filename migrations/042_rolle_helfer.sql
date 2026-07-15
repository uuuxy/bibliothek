-- =============================================================================
-- Migration 042: Rolle 'helfer' im ENUM benutzer_rolle ergänzen
-- =============================================================================
-- Die Rolle HELFER war bisher unerreichbar: auth/jwt.go definiert sie, die
-- role_permissions-Tabelle seedet ihre Rechte und Router.svelte verzweigt auf
-- role === 'helfer' — aber das ENUM benutzer_rolle kannte nur admin/lehrer/
-- mitarbeiter, sodass sie niemandem zugewiesen werden konnte.
--
-- Hinweis: ALTER TYPE ... ADD VALUE ist ab PostgreSQL 12 auch innerhalb einer
-- Transaktion erlaubt (der neue Wert ist erst nach dem Commit benutzbar) —
-- der Migrations-Runner fährt jede Datei in einer eigenen TX.
-- =============================================================================

ALTER TYPE benutzer_rolle ADD VALUE IF NOT EXISTS 'helfer';
