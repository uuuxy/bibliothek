-- Migration 117: Nachbuch-Meldungen (Offline-Betrieb der Theke, Stufe 2).
--
-- Ein offline gescannter Vorgang wird später nachgebucht — und dabei wird die
-- Wirklichkeit gebucht, nicht der Scan (Entscheidung Peter, 13.09.2026, c): Lag das Buch
-- inzwischen bei jemand anderem, wird dort zurückgenommen und neu ausgeliehen; ist der
-- Scan älter als die letzte Bewegung des Exemplars, wird er abgewiesen; scheitert die
-- Ausleihe an Sperre, Limit oder Vormerkung, bleibt die Rücknahme und die Ausleihe wird
-- gemeldet. Nichts davon verschwindet still: Jede Abweichung vom Scan steht hier, mit
-- Barcode, Grund und Beteiligten, bis jemand aus der Bibliothek sie quittiert.
--
-- Personenbezug: Ausleiher und Vorbesitzer als Fremdschlüssel (Schüler oder Lehrkraft),
-- dazu der Ausweis-Barcode als Text, wenn er sich beim Nachbuchen nicht auflösen ließ.
-- Beide Personenspalten wandern beim Zusammenführen mit, werden über SpurTilgungen
-- getilgt, stehen in der Art.-15-Auskunft und in dsgvoSchuelerQuellen. Quittierte
-- Meldungen fallen nach der Lesehistorie-Frist, höchstens nach 30 Tagen
-- (PredikatNachbuchMeldungen); offene erscheinen nach 14 Tagen als Warnung in der
-- Betriebsbereitschaft. Sichtbar nur mit view_students — nicht für Helfer.
--
-- idempotency_key ist eindeutig: Wiederholt der Theken-Rechner einen Eintrag, entsteht
-- keine zweite Meldung — auch nach Ablauf des 24-Stunden-Antwort-Caches nicht.

CREATE TABLE nachbuch_meldungen (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	idempotency_key UUID NOT NULL,
	exemplar_id UUID REFERENCES buecher_exemplare(id) ON DELETE SET NULL,
	barcode TEXT NOT NULL,
	ergebnis TEXT NOT NULL,
	grund TEXT,
	ausleiher_schueler_id UUID REFERENCES schueler(id) ON DELETE SET NULL,
	ausleiher_benutzer_id UUID REFERENCES benutzer(id) ON DELETE SET NULL,
	vorbesitzer_schueler_id UUID REFERENCES schueler(id) ON DELETE SET NULL,
	vorbesitzer_benutzer_id UUID REFERENCES benutzer(id) ON DELETE SET NULL,
	ausweis_text TEXT,
	gescannt_am TIMESTAMP WITH TIME ZONE NOT NULL,
	erstellt_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
	quittiert_von UUID REFERENCES benutzer(id) ON DELETE SET NULL,
	quittiert_am TIMESTAMP WITH TIME ZONE,
	CONSTRAINT uniq_nachbuch_meldungen_schluessel UNIQUE (idempotency_key),
	CONSTRAINT chk_nachbuch_ergebnis CHECK (ergebnis IN (
		'ausgeliehen', 'umgebucht', 'bereits_ausgeliehen', 'zurueckgegeben',
		'nur_reaktiviert', 'nicht_gebucht', 'veraltet'))
);

CREATE INDEX idx_nachbuch_meldungen_offen ON nachbuch_meldungen (erstellt_am) WHERE quittiert_am IS NULL;
CREATE INDEX idx_nachbuch_meldungen_ausleiher ON nachbuch_meldungen (ausleiher_schueler_id) WHERE ausleiher_schueler_id IS NOT NULL;
CREATE INDEX idx_nachbuch_meldungen_vorbesitzer ON nachbuch_meldungen (vorbesitzer_schueler_id) WHERE vorbesitzer_schueler_id IS NOT NULL;
