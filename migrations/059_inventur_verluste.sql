-- Welche Exemplare eine Inventur als Verlust gebucht hat.
--
-- FinishInventurSession markiert alle nicht erfassten Exemplare des Scopes als
-- ausgesondert (Grund VERLUST) und gab bisher nur eine ZAHL zurück: „47 Verluste".
-- Welche 47, erfuhr niemand — und danach war es auch nicht mehr herleitbar: Durch die
-- Aussonderung fallen die Exemplare aus der Scope-Bedingung, die Menge lässt sich also
-- nicht rückwirkend berechnen.
--
-- Praktisch heisst das: Man kann nicht nachsehen, ob ein Buch wirklich fehlt oder nur
-- im falschen Regal stand, kann keine Liste zum Nachsuchen ausdrucken und der Schule
-- nicht sagen, was abhandengekommen ist.
CREATE TABLE IF NOT EXISTS inventur_verluste (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	session_id UUID NOT NULL REFERENCES inventur_sessions(id) ON DELETE CASCADE,
	exemplar_id UUID REFERENCES buecher_exemplare(id) ON DELETE SET NULL,

	-- Abschrift statt reinem Verweis: Der Bericht muss auch dann noch lesbar sein, wenn
	-- das Exemplar oder der Titel später endgültig gelöscht wird (ON DELETE SET NULL
	-- oben). Ein Verlustbericht, der nach einer Bereinigung nur noch leere Zeilen zeigt,
	-- wäre wertlos — und genau dann braucht man ihn, wenn jemand nachfragt.
	barcode_id TEXT NOT NULL,
	titel TEXT NOT NULL,
	autor TEXT NOT NULL DEFAULT '',
	signatur TEXT NOT NULL DEFAULT '',

	gebucht_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Der Bericht liest immer nach Session.
CREATE INDEX IF NOT EXISTS idx_inventur_verluste_session
	ON inventur_verluste (session_id);

-- Ein Exemplar darf in derselben Session nur einmal als Verlust stehen. Schützt gegen
-- einen doppelten Abschluss, der die Liste sonst still verdoppeln würde.
CREATE UNIQUE INDEX IF NOT EXISTS idx_inventur_verluste_einmalig
	ON inventur_verluste (session_id, exemplar_id)
	WHERE exemplar_id IS NOT NULL;

COMMENT ON TABLE inventur_verluste IS
	'Abschrift der bei einem Inventur-Abschluss als Verlust gebuchten Exemplare — Grundlage des Fehlbestandsberichts.';
