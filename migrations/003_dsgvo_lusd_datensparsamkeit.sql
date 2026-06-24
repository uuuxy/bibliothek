-- ============================================================
-- Migration: DSGVO-Härtung LUSD-Import – Datensparsamkeit
-- Rechtsgrundlage: Art. 5 Abs. 1 lit. c DSGVO (Datensparsamkeit)
-- Datum: 2026-05-31
-- ============================================================

-- 1. Geburtsdatum als zusätzliches LUSD-Feld.
--    NULL für Altdatensätze, die vor dieser Migration importiert wurden.
ALTER TABLE schueler
  ADD COLUMN IF NOT EXISTS geburtsdatum DATE DEFAULT NULL;

-- 2. HINWEIS (Richtlinienänderung 2026-06):
--    Der frühere "Sicherheitsnachweis", der adress-/kontaktbezogene Spalten
--    (strasse, hausnummer, plz, ort, eltern_email …) per RAISE EXCEPTION
--    VERBOT, wurde bewusst ENTFERNT. Die Schulbibliothek benötigt Postanschrift
--    und Elternkontakt fachlich zwingend (u. a. für das Mahnwesen / Elternbriefe).
--    Diese Daten werden auf Grundlage der Aufgabenerfüllung der Schule verarbeitet;
--    die Spalten werden über Migration 009 angelegt und vom Adressfeature genutzt.
--
--    Der alte Wächter war zudem ein Betriebsrisiko: Da schema.sql diese Spalten
--    inzwischen anlegt, hätte die Exception jede Neuinstallation beim Start
--    abgebrochen. Datensparsamkeit bleibt Leitlinie (nur benötigte Felder), wird
--    aber nicht mehr per harter DB-Exception gegen das Adressfeature erzwungen.
