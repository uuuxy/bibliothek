-- Migration 107: Die drei Beispiel-Lieferanten des Programmstarts sind weg.
--
-- db/seed.go legte vom 30.05. bis zum 07.09.2026 bei leerer Tabelle drei Beispiel-
-- Lieferanten an — „Klett Verlag" bestellung@klett.de K-99281, „Cornelsen"
-- service@cornelsen.de C-88123, „Westermann" order@westermann.de W-77441. In der ersten
-- Bauwoche gewollt: Startdaten, damit das Bestellwesen ohne Vorarbeit bedienbar war.
-- Das Problem war nie die Absicht, sondern der Ort — Startdaten im Boot-Pfad sind auf
-- einer echten Anlage von echten Händlern nicht zu unterscheiden. Der Bestellweg (internal/service/order_service.go) schickt die Bestellmail wirklich an
-- diese Adresse, die Historie meldet „gesendet", und die Schule wartet auf Bücher, die nie
-- kommen. Auf dem Test-Server standen die drei real in der Datenbank, eine Bestellung
-- ging an einen davon. Auf jeder frischen Anlage wäre beim ersten Start dasselbe passiert.
--
-- Dieselbe Bugklasse wie Migration 106, nur mit Daten statt Schema: Der Boot tut etwas,
-- das nirgends steht. Die Selbstprüfung zählte Demo-Schüler aus seed_demo.sql — aber
-- nicht, was der Go-Prozess selbst anlegt.
--
-- Gelöscht wird nur das exakte Tripel aus dem alten Seed. Ein Eintrag, den jemand mit den
-- echten Daten des Händlers überschrieben hat, bleibt. Bestellungen an einen gelöschten
-- Eintrag bleiben als Beleg erhalten: bestellungen_verlauf trägt Name, Adresse und
-- Kundennummer selbst, der Fremdschlüssel steht auf ON DELETE SET NULL (Migration 037).
--
-- Reste — etwa ein umbenannter Eintrag mit der Beispiel-Adresse — meldet ab jetzt die
-- Selbstprüfung als kritisch (repository/betriebszustand.go, BeispielLieferanten).

DELETE FROM lieferanten
 WHERE (name, email, kundennummer) IN (
	('Klett Verlag', 'bestellung@klett.de', 'K-99281'),
	('Cornelsen',    'service@cornelsen.de', 'C-88123'),
	('Westermann',   'order@westermann.de',  'W-77441')
 );
