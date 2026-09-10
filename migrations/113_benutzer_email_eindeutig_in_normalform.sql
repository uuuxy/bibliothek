-- Migration 113: Eine E-Mail-Adresse gehört genau einem Konto — in der Normalform, in der
-- die Anmeldung sucht.
--
-- Die Anmeldung findet das Konto über LOWER(email) = LOWER($1) LIMIT 1 (auth/handlers.go).
-- UNIQUE lag aber auf dem Rohtext, und die Prüfung beim Anlegen verglich exakt: Eine
-- Selbstanmeldung legte „erika.muster@…" an, die Bibliothek daneben „Erika.Muster@…" —
-- beides ging durch, und welches der beiden Konten ein Login öffnete, entschied die
-- Speicherreihenfolge der Tabelle (Bestands-Durchgang 10.09.2026). Mit dem delegierten
-- Recht manage_users ließ sich so auch das Konto des Administrators aussperren.
--
-- Stehen bereits zwei Schreibweisen derselben Adresse in der Tabelle, bricht diese
-- Migration LAUT ab, statt still weiterzulaufen: Welches der beiden Konten bleibt, ist
-- eine Entscheidung eines Menschen (Rolle, aktiv, Verlauf), keine einer Migration. Die
-- Meldung nennt die Abfrage, die die Paare zeigt.

DO $$
BEGIN
	IF EXISTS (SELECT 1 FROM benutzer GROUP BY lower(email) HAVING count(*) > 1) THEN
		RAISE EXCEPTION 'Migration 113: benutzer.email enthält dieselbe Adresse in verschiedener Schreibweise. Paare anzeigen mit: SELECT lower(email), array_agg(id || '' '' || email || '' aktiv='' || aktiv) FROM benutzer GROUP BY 1 HAVING count(*) > 1; — ein Konto je Paar zusammenführen oder löschen, dann erneut starten.';
	END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS uniq_benutzer_email_lower ON benutzer (lower(email));
