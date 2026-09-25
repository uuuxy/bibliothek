-- =============================================================================
-- Migration 147: Protokolleinträge zu gelöschten Lesern verlieren, was die Tilgung entfernt
-- =============================================================================
-- Die Tilgung (repository.spurTilgungen, Anweisung „audit_logs (LUSD-ID, Barcodes,
-- Sperrgrund)") nimmt beim Anonymisieren und beim endgültigen Löschen eines Lesers fünf
-- Schlüssel aus seinen Einträgen im Admin-Protokoll: lusd_id, barcode, aufgeloest_barcode,
-- grund und reason. Die Einträge selbst bleiben (Aktion, Zeit, Bearbeiter, die Kennung als
-- Pseudonym). lusd_id kam am 22.08.2026 dazu (6d01f27a), die Barcodes am 02.09.2026
-- (131a534a), grund und reason am 24.09.2026 (5b50202d).
--
-- Wer vorher endgültig gelöscht wurde, behielt diese Schlüssel, ebenso ein Leser, den ein
-- Skript direkt gelöscht hat (scripts/entferne_demo_daten.sql). Die nächtliche
-- Nachbereinigung erreicht solche Einträge nicht: Sie sucht über anonymized_at in
-- vorhandenen Zeilen, und die Zeile gibt es nicht mehr. Gemessen am Testserver am
-- 25.09.2026: 4 Einträge mit grund oder reason, davon 3 zu einer Kennung ohne Leser. Die
-- übrigen drei Schlüssel sind an solchen Einträgen nicht einzeln gemessen; die Anweisung
-- nimmt sie mit weg — dieselbe Regel, die beim Löschen heute gilt.
--
-- Neue Einträge dieser Art entstehen nicht mehr: Das endgültige Löschen tilgt vorher
-- (TilgeSchuelerSpuren), und das Zusammenführen hängt die Einträge auf den Ziel-Leser um.
--
-- Die Kennung wird über lower() verglichen: Ein Eintrag, der einen vorhandenen Leser in
-- Großbuchstaben nennt, bleibt unangetastet. Idempotent: Beim zweiten Lauf trägt kein
-- solcher Eintrag mehr einen der Schlüssel.

UPDATE audit_logs a
SET details = a.details - 'lusd_id' - 'barcode' - 'aufgeloest_barcode' - 'grund' - 'reason'
WHERE (a.details ? 'lusd_id' OR a.details ? 'barcode' OR a.details ? 'aufgeloest_barcode'
       OR a.details ? 'grund' OR a.details ? 'reason')
  AND a.details ? 'schueler_id'
  AND NOT EXISTS (SELECT 1 FROM leser l WHERE l.id::text = lower(a.details->>'schueler_id'));
