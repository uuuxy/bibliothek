-- Index für die Sortierung des Logbuchs.
--
-- /api/audit zeigt die jüngsten Einträge (ORDER BY timestamp DESC LIMIT 1000). Ohne
-- Index muss Postgres dafür die gesamte Tabelle sortieren — bei 247.000 Zeilen auf dem
-- Prüfstand rund 100 ms, nur um 1000 davon zu behalten. Das Logbuch wächst im Betrieb
-- unbegrenzt weiter: Der Aufwand steigt mit jeder protokollierten Aktion, der Nutzen
-- bleibt bei 1000 Zeilen.
--
-- Mit Index wird daraus ein Rückwärts-Scan, der nach 1000 Treffern aufhört — unabhängig
-- davon, wie groß die Tabelle geworden ist.
--
-- DESC passend zur Abfrage: Ein aufsteigender Index taugte hier zwar auch (Postgres kann
-- ihn rückwärts lesen), aber die Absicht steht so an der Stelle, an der sie gilt.
CREATE INDEX IF NOT EXISTS idx_audit_log_timestamp_desc
	ON audit_log (timestamp DESC);
