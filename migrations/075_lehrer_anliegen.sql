-- 075: Anliegen der Lehrkräfte — Wünsche und Meldungen (Betreiber-Entscheidung 18.08.2026).
--
-- Zwei Alltagsfälle, EIN Mechanismus: "Ich möchte in der 8G3 den Markl 2"
-- (Wunsch) und "die 8G3 hat die falschen Bücher bekommen" (Meldung). Die
-- Lehrkraft trägt es im Kollegiums-Portal ein, die LMF arbeitet die Liste in
-- Ruhe ab statt Klassen vor der Tür; beim Abhaken geht eine Mail an die
-- Lehrkraft. Bewusst OHNE Wunschphase/Stichtag (Wünschen geht immer), ohne
-- Prioritäten, ohne Kommentar-Threads — einfach halten.
--
-- klasse ist Freitext (auch Oberstufen-Kurse); titel_id ist gesetzt, wenn der
-- Wunsch aus dem Katalog kam, sonst trägt titel_text die freie Angabe.
CREATE TABLE IF NOT EXISTS lehrer_anliegen (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	art TEXT NOT NULL CONSTRAINT chk_anliegen_art CHECK (art IN ('wunsch', 'meldung')),
	titel_text TEXT NOT NULL,
	titel_id UUID REFERENCES buecher_titel(id) ON DELETE SET NULL,
	isbn TEXT NOT NULL DEFAULT '',
	klasse TEXT NOT NULL DEFAULT '',
	kommentar TEXT NOT NULL DEFAULT '',
	-- SET NULL statt RESTRICT: Ein gelöschtes Konto darf offene Anliegen nicht
	-- blockieren; die Mail entfällt dann schlicht (gleiche Haltung wie bei den
	-- Klassensatz-Reservierungen).
	angefordert_von UUID REFERENCES benutzer(id) ON DELETE SET NULL,
	erstellt_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
	erledigt_am TIMESTAMP WITH TIME ZONE,
	erledigt_notiz TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_anliegen_offen ON lehrer_anliegen (erstellt_am) WHERE erledigt_am IS NULL;
