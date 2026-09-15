-- Migration 116: Bewegungsstempel für das Nachbuchen (Offline-Betrieb der Theke, Stufe 2).
--
-- Ein offline gescannter Vorgang kommt später an — Minuten oder Stunden. Dazwischen kann
-- dasselbe Buch an der anderen Theke zurückgenommen und neu ausgegeben worden sein. Damit
-- der Server beim Nachbuchen die WIRKLICHKEIT bucht und nicht den alten Scan, braucht er
-- zwei Zeitpunkte, die es bisher nicht gab:
--
--   ausleihen.erfasst_am            Wann der Vorgang gescannt wurde. Online ist das der
--                                   Moment der Buchung (Default); beim Nachbuchen der
--                                   Scan-Zeitpunkt vom Theken-Rechner, höchstens Serverzeit.
--   buecher_exemplare.letzte_bewegung_am
--                                   Wann das Exemplar zuletzt bewegt wurde: Ausleihe,
--                                   Rückgabe, Rückholen (Fund), Aussonderung. Der Wächter
--                                   des Nachbuchens vergleicht den Scan damit: älter als die
--                                   letzte Bewegung heißt „veraltet", und der Eintrag wird
--                                   gemeldet statt gebucht.
--
-- Geschrieben wird der Stempel von den Schreibern selbst, NICHT vom generischen
-- aktualisiert_am-Trigger: Der schlüge auch bei einem Cover-Upload oder einer Notiz an, und
-- das Nachbuchen muss die Scan-Zeit eintragen können, nicht „jetzt".
--
-- NULL in letzte_bewegung_am heißt „keine bekannte Bewegung" — der Wächter lässt den Scan
-- dann durch. Für den Bestand wird die letzte bekannte Bewegung aus den Ausleihen
-- zurückgerechnet; erfasst_am bekommt für vorhandene Zeilen das Ausleihdatum, den einzigen
-- Scan-Zeitpunkt, den es dafür gibt.

ALTER TABLE ausleihen
	ADD COLUMN erfasst_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP;

UPDATE ausleihen SET erfasst_am = ausgeliehen_am;

ALTER TABLE buecher_exemplare
	ADD COLUMN letzte_bewegung_am TIMESTAMP WITH TIME ZONE;

UPDATE buecher_exemplare e
   SET letzte_bewegung_am = m.letzte
  FROM (
	SELECT exemplar_id, max(greatest(ausgeliehen_am, coalesce(rueckgabe_am, ausgeliehen_am))) AS letzte
	  FROM ausleihen
	 WHERE exemplar_id IS NOT NULL
	 GROUP BY exemplar_id
  ) m
 WHERE m.exemplar_id = e.id;
