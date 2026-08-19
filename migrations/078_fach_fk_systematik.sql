-- =============================================================================
-- Migration 078: Fach wird Fremdschlüssel auf die Systematik (Prüfbericht F3)
-- =============================================================================
-- buecher_titel.subject hielt die Systematik-Bezeichnung als freien Text. Die
-- Kopplung existierte nur in der Anwendung (Rename-Propagation im Handler,
-- Lösch-Schutz als 409) — jeder andere Schreibweg konnte lautlos driften.
-- Jetzt garantiert die Datenbank sie: subject → systematik_kategorien(bezeichnung)
-- mit ON UPDATE CASCADE (Umbenennen zieht alle Titel mit) und ON DELETE RESTRICT.
--
-- Lehre aus Migration 021→060: Ein FK lebt nur, wenn ALLE Schreibpfade ihn
-- bedienen. Die Importe registrieren unbekannte Fächer deshalb ab jetzt selbst
-- (inventur.StelleFaecherSicher); dieser Backfill holt den Altbestand nach.
-- =============================================================================

-- 1) Bezeichnungen trimmen (Kürzel trimmt der Handler seit jeher selbst).
UPDATE systematik_kategorien SET bezeichnung = btrim(bezeichnung)
WHERE bezeichnung <> btrim(bezeichnung);

-- 2) Dubletten, die sich nur in Groß-/Kleinschreibung unterschieden, zusammenlegen:
--    die älteste Zeile gewinnt. (Titel hängen als Text an der Bezeichnung und werden
--    in Schritt 4 auf die überlebende Schreibweise gezogen.)
DELETE FROM systematik_kategorien k
USING systematik_kategorien aelter
WHERE lower(k.bezeichnung) = lower(aelter.bezeichnung)
  AND k.id <> aelter.id
  AND (aelter.erstellt_am < k.erstellt_am
       OR (aelter.erstellt_am = k.erstellt_am AND aelter.id < k.id));

-- 3) Titel-Fächer säubern: Rand-Leerzeichen weg, Leerwert wird NULL (ein FK gilt
--    nur für Nicht-NULL-Werte; '' wäre sonst ein Fach, das es nie gab).
UPDATE buecher_titel SET subject = btrim(subject)
WHERE subject IS NOT NULL AND subject <> btrim(subject);
UPDATE buecher_titel SET subject = NULL WHERE subject = '';

-- 4) Titel auf die kanonische Schreibweise der Systematik ziehen ("deutsch" → "Deutsch").
UPDATE buecher_titel t SET subject = k.bezeichnung
FROM systematik_kategorien k
WHERE t.subject IS NOT NULL
  AND lower(t.subject) = lower(k.bezeichnung)
  AND t.subject <> k.bezeichnung;

-- 5) Fächer, die bisher nur in Titeln existieren, in der Systematik nachregistrieren.
--    Kürzel-Kandidat: Bezeichnung ohne Leerzeichen (Kürzel bilden Signatur-Vorschläge
--    und dürfen keine tragen). Kollidiert der Kandidat — mit einem bestehenden Kürzel
--    oder innerhalb dieses Backfills — entscheidet ein Hash-Suffix.
INSERT INTO systematik_kategorien (kuerzel, bezeichnung)
SELECT CASE
         WHEN EXISTS (SELECT 1 FROM systematik_kategorien k WHERE k.kuerzel = f.kandidat)
              OR count(*) OVER (PARTITION BY f.kandidat) > 1
         THEN left(f.kandidat, 41) || '~' || substr(md5(lower(f.bez)), 1, 8)
         ELSE f.kandidat
       END,
       f.bez
FROM (
    SELECT DISTINCT ON (lower(subject))
           subject AS bez,
           left(replace(subject, ' ', ''), 50) AS kandidat
    FROM buecher_titel
    WHERE subject IS NOT NULL
      AND NOT EXISTS (SELECT 1 FROM systematik_kategorien k
                      WHERE lower(k.bezeichnung) = lower(buecher_titel.subject))
    ORDER BY lower(subject), subject
) f;

-- 6) Schreibvarianten, deren kanonische Form eben erst registriert wurde, nachziehen
--    (Schritt 5 nimmt je lower(subject) genau EINE Schreibweise auf).
UPDATE buecher_titel t SET subject = k.bezeichnung
FROM systematik_kategorien k
WHERE t.subject IS NOT NULL
  AND lower(t.subject) = lower(k.bezeichnung)
  AND t.subject <> k.bezeichnung;

-- 7) Offene Inventur-Sessions folgen der kanonischen Schreibweise, sonst zählt ihr
--    Filter-Scope ab jetzt null Exemplare. Abgeschlossene Sessions sind Historie
--    und bleiben unangetastet.
UPDATE inventur_sessions s SET scope_subject = k.bezeichnung
FROM systematik_kategorien k
WHERE s.abgeschlossen_am IS NULL
  AND s.scope_subject IS NOT NULL
  AND lower(btrim(s.scope_subject)) = lower(k.bezeichnung)
  AND s.scope_subject <> k.bezeichnung;

-- 8) Verankern: Bezeichnung wird eindeutig (exakt für den FK, case-insensitiv als
--    Hygiene gegen künftige "deutsch"-neben-"Deutsch"-Anlagen), subject wächst auf
--    die Breite der Bezeichnung, dann der Fremdschlüssel selbst.
ALTER TABLE buecher_titel ALTER COLUMN subject TYPE VARCHAR(255);

ALTER TABLE systematik_kategorien
    ADD CONSTRAINT uniq_systematik_bezeichnung UNIQUE (bezeichnung);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_systematik_bezeichnung_ci
    ON systematik_kategorien (lower(bezeichnung));

CREATE INDEX IF NOT EXISTS idx_titel_subject
    ON buecher_titel (subject) WHERE subject IS NOT NULL;

ALTER TABLE buecher_titel
    ADD CONSTRAINT fk_titel_subject_systematik
    FOREIGN KEY (subject) REFERENCES systematik_kategorien (bezeichnung)
    ON UPDATE CASCADE ON DELETE RESTRICT;
