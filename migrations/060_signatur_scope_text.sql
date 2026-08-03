-- =============================================================================
-- Migration 060: Signatur-Inventur auf die echte Signatur — die MDM-Welt war tot
-- =============================================================================
-- Es gab zwei Signatur-Welten. Migration 038 hat eine davon zur einzigen Quelle
-- erklärt: buecher_titel.signatur (Text — das, was physisch auf dem Buchrücken
-- klebt). Die andere, Tabelle `signatures` samt buecher_titel.signature_id aus
-- Migration 021, hat nie jemand gepflegt: Geschrieben hat die Spalte einzig
-- Migration 021 selbst (abgeleitet aus subject). Buchformular, Update und Import
-- fassen sie nicht an, das Go-Modell hat gar kein Feld dafür.
--
-- Zwei belegte Folgen dieser toten Welt:
--
--   * Die Inventur „nur bestimmte Signatur" filterte auf t.signature_id und traf
--     damit NULL Exemplare — bei einer Auswahlliste, die nur noch Testrückstände
--     anbot. Nach dem Littera-Import wäre das so geblieben, denn der Import
--     schreibt signatur (Text), nicht signature_id.
--
--   * Der Fehlbestandsbericht (Migration 059) übernahm seine Signatur-Abschrift
--     per Join auf `signatures` — also immer den Leerstring. Ausgerechnet die
--     Sortierung „nach Signatur", mit der man mit dem Zettel durchs Regal läuft,
--     sortierte damit nach nichts.
--
-- Diese Migration stellt den Inventur-Scope auf den Signatur-Text um und entfernt
-- die tote Welt. Es geht dabei keine Information verloren: signature_id war
-- vollständig aus subject abgeleitet (Migration 021, Schritt 4), und subject
-- bleibt unangetastet.
--
-- Der Scope ist ab jetzt ein PRÄFIX der Signatur, kein Fremdschlüssel. Eine
-- Signatur ist eine Regaladresse; eine kürzere Adresse meint einen größeren
-- Regalbereich. „BIB Deu" erfasst damit auch „BIB Deu 5 KRÜ", „BIB Deu 5 KRÜ"
-- nur sich selbst. Die Grenze läuft am Leerzeichen, damit „BIB De" nicht
-- versehentlich „BIB Deu" mitnimmt.
-- =============================================================================

ALTER TABLE inventur_sessions ADD COLUMN IF NOT EXISTS scope_signatur TEXT;

-- Bestehende Signatur-Sessions behalten ihren Bereich: Name aus der alten
-- Tabelle, ersatzweise das ohnehin gespeicherte Label.
UPDATE inventur_sessions s
SET scope_signatur = g.name
FROM signatures g
WHERE s.signature_id = g.id
  AND COALESCE(btrim(s.scope_signatur), '') = '';

UPDATE inventur_sessions
SET scope_signatur = COALESCE(NULLIF(btrim(scope_label), ''), 'Unbekannt')
WHERE scope_type = 'signature'
  AND COALESCE(btrim(scope_signatur), '') = '';

-- CHECK: eine Signatur-Session braucht jetzt einen nicht-leeren Signatur-Text.
ALTER TABLE inventur_sessions DROP CONSTRAINT IF EXISTS chk_inv_session_scope;
ALTER TABLE inventur_sessions ADD CONSTRAINT chk_inv_session_scope
    CHECK (
        scope_type IN ('global', 'signature', 'filter')
        AND (scope_type <> 'signature' OR COALESCE(btrim(scope_signatur), '') <> '')
        AND (scope_type <> 'filter' OR scope_subject IS NOT NULL OR scope_grade IS NOT NULL)
    );

-- Nur EINE offene Session je Signatur-Bereich (wie zuvor je signature_id).
DROP INDEX IF EXISTS idx_inv_session_offen_signature;
CREATE UNIQUE INDEX IF NOT EXISTS idx_inv_session_offen_signature
    ON inventur_sessions (btrim(scope_signatur))
    WHERE abgeschlossen_am IS NULL AND scope_type = 'signature';

-- Die tote Welt entfernen. Erst die Verweise, dann die Tabelle.
ALTER TABLE inventur_sessions DROP COLUMN IF EXISTS signature_id;
ALTER TABLE buecher_titel     DROP COLUMN IF EXISTS signature_id;
DROP TABLE IF EXISTS signatures;

-- Trägt die Signaturliste (DISTINCT) und den Gleichheitsteil des Scopes.
CREATE INDEX IF NOT EXISTS idx_buecher_titel_signatur
    ON buecher_titel (btrim(signatur))
    WHERE COALESCE(btrim(signatur), '') <> '';
