-- Rolle "lehrer" in "kollegium" umbenennen.
--
-- Warum: Das Wort war im System DOPPELT belegt und hat real zu einer falschen Auskunft
-- geführt. Es gibt
--   * die Anmelde-Rolle benutzer.rolle = 'lehrer'  → schaltet „Mein Portal" frei
--   * den Entleiher schueler.klasse = 'lehrer'     → Lehrkraft als Ausleiher, eigene
--     Behandlung im Mahnwesen
-- Beim Suchen nach „wo reserviert man Klassensätze" landet man abwechselnd im einen und
-- im anderen Begriff. „kollegium" benennt eindeutig die Personengruppe mit Zugang und
-- lässt „lehrer" für den Entleiher frei.
--
-- Die Rolle selbst bleibt fachlich unverändert: dieselben Rechte, dieselbe Sichtbarkeit.
-- Es ist eine Umbenennung, keine Rechteänderung.
--
-- Wiederholbar und für frische Datenbanken geeignet: schema.sql legt den Typ bereits mit
-- 'kollegium' an, dann findet der Block unten nichts vor und tut nichts.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_enum e
        JOIN pg_type t ON t.oid = e.enumtypid
        WHERE t.typname = 'benutzer_rolle' AND e.enumlabel = 'lehrer'
    ) THEN
        ALTER TYPE benutzer_rolle RENAME VALUE 'lehrer' TO 'kollegium';
    END IF;
END $$;

-- Die Rechtetabelle führt die Rolle als freien Text in GROSSSCHREIBUNG und hat den
-- Primärschlüssel (role, permission). Deshalb erst die Zeilen wegräumen, für die es
-- KOLLEGIUM schon gibt — sonst bricht das UPDATE am Primärschlüssel. Alles andere wird
-- umgehängt statt neu geschrieben, damit im Rechte-Manager vorgenommene Anpassungen
-- erhalten bleiben.
--
-- Der Existenz-Riegel ist PFLICHT, nicht Vorsicht: role_permissions steht NICHT in
-- schema.sql, sondern wird von db.InitPermissions angelegt — und das läuft in main.go
-- erst NACH RunMigrations (Zeile 130 gegen Zeile 124). Auf einer frischen Datenbank
-- existiert die Tabelle hier also noch nicht, und ein ungeschütztes DELETE ließe die
-- Migration scheitern. Da RunMigrations beim Fehler abbricht, würde die Anwendung gar
-- nicht erst hochfahren. Ohne Tabelle ist auch nichts zu tun: Der Seed schreibt danach
-- ohnehin direkt KOLLEGIUM.
DO $$
BEGIN
    IF to_regclass('public.role_permissions') IS NOT NULL THEN
        DELETE FROM role_permissions alt
         WHERE alt.role = 'LEHRER'
           AND EXISTS (
                SELECT 1 FROM role_permissions neu
                 WHERE neu.role = 'KOLLEGIUM' AND neu.permission = alt.permission
           );

        UPDATE role_permissions SET role = 'KOLLEGIUM' WHERE role = 'LEHRER';
    END IF;
END $$;
