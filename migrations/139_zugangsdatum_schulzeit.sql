-- =============================================================================
-- Migration 139: das Zugangsdatum neuer Exemplare ist der Kalendertag der Schule
-- =============================================================================
-- Handanlage, Sammelimport und Bestand-Nachziehen schreiben kein erworben_am, sie nehmen
-- die Vorgabe der Spalte. Die war CURRENT_DATE, also der Tag der Datenbank-Sitzung — im
-- Image UTC, zwischen Mitternacht in Berlin und 2 Uhr (im Winter 1 Uhr) der Vortag. Der
-- INSERT-Zweig von stempel_zugang_am() (Migration 129) übernimmt den Wert als zugang_am,
-- und das Zugangsbuch ist der Nachweis zum Stichtag 15.3./15.9.
--
-- Migration 130 hat den UPDATE-Zweig (Wareneingang) schon auf die Schulzeit gestellt; die
-- Vorgabe der Spalte war die letzte Stelle mit der anderen Regel. Littera trägt beim
-- Anlegen „das aktuelle Tagesdatum" ein — das des Arbeitsplatzes, also der Schule.
--
-- Der Ausdruck ist derselbe wie in pkg/schulzeit (SQLHeute); er steht hier ausgeschrieben,
-- weil SQL kein Go-Paket importiert. Der Listenimport (internal/service/import_dynamic.go)
-- schreibt seit derselben Änderung kein CURRENT_DATE mehr, sondern nimmt diese Vorgabe.
--
-- Bestehende Zeilen bleiben: Ob ein Zugang in jenen zwei Stunden entstand, steht nirgends.
-- Idempotent: SET DEFAULT lässt sich beliebig oft ausführen.

ALTER TABLE buecher_exemplare
    ALTER COLUMN erworben_am SET DEFAULT ((now() AT TIME ZONE 'Europe/Berlin')::date);
