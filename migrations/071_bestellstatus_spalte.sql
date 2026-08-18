-- 071: Der Bestellstatus bekommt eine eigene Spalte mit Prüfregel.
--
-- Bis hier steuerte das FREITEXT-Feld zustand_notiz das Bestellwesen: Exemplare
-- mit "Im Zulauf…"/"Bestellt…" galten als laufende Bestellung und wurden aus
-- OPAC- und Inventur-Zählungen ausgeblendet (LIKE-Muster an vier Stellen).
-- Eine harmlose Personal-Notiz wie "Bestellt am 3.9. neu, alter Band verloren"
-- ließ ein Exemplar damit still aus dem Katalog verschwinden — Befund F1 der
-- unabhängigen Prüfung (bewertung/datenbank-pruefbericht.md).
--
-- Neu: bestellstatus trägt den Zustand der Bestell-Pipeline, NULL = kein
-- laufender Bestellvorgang. Die Notiz bleibt reiner Menschentext (der
-- Lieferantenname im Notiztext bleibt als Anzeige erhalten, steuert aber
-- nichts mehr).
--
-- Wiederholbar gehalten (IF NOT EXISTS / benannte Constraint-Prüfung), siehe
-- Lehre aus Migration 066: Ein Teardown darf eine Migration nicht unbrauchbar machen.

ALTER TABLE buecher_exemplare ADD COLUMN IF NOT EXISTS bestellstatus TEXT DEFAULT NULL;

DO $$
BEGIN
	IF NOT EXISTS (
		SELECT 1 FROM pg_constraint WHERE conname = 'chk_exemplar_bestellstatus'
	) THEN
		ALTER TABLE buecher_exemplare
			ADD CONSTRAINT chk_exemplar_bestellstatus
			CHECK (bestellstatus IN ('bestellt', 'im_zulauf'));
	END IF;
END $$;

-- Backfill aus den bisherigen Magie-Texten. Nur echte Pipeline-Exemplare tragen
-- diese Präfixe systemgeschrieben (api/order_service.go); menschliche Notizen,
-- die zufällig so beginnen, waren genau der Fehler und lassen sich rückwirkend
-- nicht unterscheiden — der Backfill übernimmt den bisherigen Ist-Zustand,
-- ab jetzt entscheidet nur noch die Spalte.
UPDATE buecher_exemplare
SET bestellstatus = 'im_zulauf'
WHERE bestellstatus IS NULL AND zustand_notiz LIKE 'Im Zulauf%';

UPDATE buecher_exemplare
SET bestellstatus = 'bestellt'
WHERE bestellstatus IS NULL
  AND (zustand_notiz = 'bestellt' OR zustand_notiz LIKE 'Bestellt%');

-- Teilindex für die Wareneingangs-Liste (alle offenen Bestell-Exemplare).
CREATE INDEX IF NOT EXISTS idx_exemplare_bestellstatus
	ON buecher_exemplare (bestellstatus) WHERE bestellstatus IS NOT NULL;
