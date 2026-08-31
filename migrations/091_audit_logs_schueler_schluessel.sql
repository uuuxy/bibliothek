-- 091: Ein Schlüssel für den Schüler-Verweis in audit_logs.details — `schueler_id`.
--
-- DELETE_STUDENT, RESTORE_STUDENT und PURGE_STUDENT schrieben denselben Wert als
-- `student_id`, während LUSD_ID_NACHGETRAGEN `schueler_id` schreibt. Jede Abfrage über
-- details->>'schueler_id' — die Art.-15-Auskunft ebenso wie die DSGVO-Tilgung
-- (repository.SpurTilgungen) — sah die student_id-Einträge deshalb nicht: zwei Namen
-- für denselben Wert, zwei Wahrheitsquellen (Fund 31.08.2026).
--
-- Diese Migration schlüsselt die Altzeilen um; die drei Schreibstellen schreiben seit
-- demselben Commit `schueler_id`, und api/dsgvo_paar_vollstaendigkeit_test.go friert
-- das Vokabular ein. Inhalt und Zeitstempel der Einträge bleiben unangetastet —
-- umbenannt wird nur der Schlüssel, nicht die Aussage (Rechenschaftspflicht).
UPDATE audit_logs
SET details = (details - 'student_id') || jsonb_build_object('schueler_id', details->'student_id')
WHERE details ? 'student_id';
