-- =============================================================================
-- Migration 122: Rechte der Rolle 'leitung' ab Werk
-- =============================================================================
-- Warum eine Migration und nicht nur der Seed: db/seed.go schreibt die
-- Rechte-Vorgabe mit ON CONFLICT DO NOTHING — es werden also nur FEHLENDE
-- Zeilen eingefügt. Das genügt hier sogar, weil es für LEITUNG noch keine Zeile
-- gibt. Die Migration steht trotzdem daneben, damit die Rechte der neuen Rolle
-- eine Bestandsanlage zum bekannten Zeitpunkt erreichen und nicht erst beim
-- nächsten Start irgendeines Prozesses — und damit die Selbstprüfung
-- (api/betriebsbereitschaft.go) direkt nach dem Einspielen keine Abweichung
-- gegen die Code-Vorgabe meldet.
--
-- Das Soll wird ABGELEITET, nicht abgeschrieben: LEITUNG ist ADMIN minus die
-- zwei Türen der Systempflege — manage_users (Benutzer & Rechte) und
-- manage_settings (Einstellungen). Eine abgeschriebene Rechteliste in SQL wäre
-- eine zweite Wahrheit neben db/seed.go und liefe beim nächsten neuen Recht
-- still auseinander: Die Leitung bekäme es nicht, und niemand merkte es.
--
-- Warum die Leitung kein manage_users bekommt, obwohl sie die Bibliothek führt:
-- Mit dem Recht ändert man die E-Mail-Adresse eines Kontos, und die Anmeldung
-- erkennt eine Person allein an ihrer E-Mail (auth/handlers.go). Die
-- Rechtevergabe ist damit der Weg in JEDES Konto der Anlage. Der Admin kann das
-- Recht der Leitung erteilen; ein Admin-KONTO bleibt ihr auch dann verschlossen
-- (api/user_admin_eskalation.go).
--
-- ADMIN ist in der Rechte-Matrix nicht abwählbar („Administrator hat immer alle
-- Rechte", PermissionsEditor.svelte), seine Zeilen stehen deshalb verlässlich
-- auf true. Die beiden Ausnahmen werden trotzdem ausdrücklich auf false
-- gesetzt, statt sich auf den Quellwert zu verlassen.
-- =============================================================================

INSERT INTO role_permissions (role, permission, allowed)
SELECT
    'LEITUNG',
    rp.permission,
    CASE WHEN rp.permission IN ('manage_users', 'manage_settings')
         THEN false
         ELSE rp.allowed
    END
FROM role_permissions rp
WHERE UPPER(rp.role) = 'ADMIN'
ON CONFLICT (role, permission) DO NOTHING;
