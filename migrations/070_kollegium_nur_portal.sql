-- Rechte der Rolle KOLLEGIUM auf den Portal-Zweck zurücknehmen.
--
-- Befund vom 10.08.2026 auf dem Schulserver: Ein Kollegiums-Konto sah 10 von 15
-- Menüpunkten, darunter Schülerdatei, Mahnwesen, System-Logs und Einstellungen. Das Menü
-- ist rechtegesteuert (frontend/src/lib/menu.js), und role_permissions führte für
-- KOLLEGIUM manage_users, audit_logs, view_stats, view_students, view_books und
-- perform_actions auf true. Es war keine reine Anzeigefrage: Dieselbe Tabelle entscheidet
-- in RequirePermission, die API hätte es also auch zugelassen.
--
-- Zweck der Rolle ist eine einzige Funktion — im Kollegiums-Portal einen Klassensatz
-- reservieren. Die Suche darin läuft über den öffentlichen OPAC, das Absenden über
-- create_reservations (neu, siehe Migration unten und api/routes_books.go). Beides fasst
-- keine Personendaten an. Alles andere wird entzogen.
--
-- Der Entzug ist bewusst ein Reset, kein Merge: Was hier auf false geht, war nie eine
-- Betreiber-Entscheidung, sondern die Vorgabe aus db/seed.go plus das, was Migration 069
-- von der Vorgängerrolle LEHRER übernommen hat. Wer einer Lehrkraft danach gezielt mehr
-- geben will, tut das im PermissionManager — Migrationen laufen nur einmal
-- (schema_migrations), eine spätere Vergabe wird also nicht wieder zurückgedreht.
--
-- Der Existenz-Riegel ist PFLICHT, nicht Vorsicht: role_permissions steht NICHT in
-- schema.sql, sondern wird erst nach den Migrationen von db.InitPermissions angelegt. Auf
-- einer frischen Datenbank findet dieser Block also nichts vor und tut nichts — dort
-- schreibt der Seed direkt die korrigierten Vorgaben.
DO $$
BEGIN
    IF to_regclass('public.role_permissions') IS NULL THEN
        RETURN;
    END IF;

    UPDATE role_permissions
       SET allowed = false
     WHERE role = 'KOLLEGIUM'
       AND permission <> 'create_reservations'
       AND allowed;

    -- DO NOTHING statt DO UPDATE: Steht die Zeile schon auf false, hat ein Administrator
    -- sie bewusst entzogen. Diese Migration nimmt Rechte weg, sie vergibt keine zurück.
    INSERT INTO role_permissions (role, permission, allowed)
    VALUES ('KOLLEGIUM', 'create_reservations', true)
    ON CONFLICT (role, permission) DO NOTHING;
END $$;
