-- =============================================================================
-- Migration 098: Jede Zeile des LMF-Plans gehört zu einem Plan
-- =============================================================================
-- Rasterdurchgang 05.09.2026 abends, Frage 1 („Konvention statt Regel"): Migration 097
-- hängte plan_id und position an lmf_termine, ließ beide aber nullbar — die Zugehörigkeit
-- war eine Verabredung zwischen Dateien, kein Constraint. Die Zeilen des Termin-Dialogs
-- vom Nachmittag (Migration 096, Weg seit 097 abgebaut) blieben damit als Waisen liegen:
-- Sie erscheinen weiter im Portal und im PDF und setzen über RueckgabeTerminFuerKlasse
-- weiter die Frist ihrer Klasse — der Planer zeigt sie aber nicht, und „Plan speichern"
-- oder „Plan verwerfen" fassen sie nicht an (beide arbeiten über plan_id). Eine Frist,
-- die niemand mehr sehen oder ändern kann, ist genau die stille Sorte Fehler.
--
-- Deshalb: Waisen entfernen (sie stammen ausschließlich aus dem einen Nachmittag) und die
-- Spalten NOT NULL setzen. Wer künftig eine Zeile ohne Plan schreiben will, scheitert an
-- der Datenbank statt an einer Verabredung.
DELETE FROM lmf_termine WHERE plan_id IS NULL;

ALTER TABLE lmf_termine
    ALTER COLUMN plan_id SET NOT NULL,
    ALTER COLUMN position SET NOT NULL;
