-- Migration 126: Die Auflage ist ein eigenes Feld am Titel.
--
-- Ein Schulbuch wird nachbestellt und es gibt es nur noch in der nächsten Auflage. Die
-- ISBN ist am Titel eindeutig, also entsteht zwangsläufig eine zweite Titelzeile — richtig
-- so, es sind zwei verschiedene Bücher. Ohne dieses Feld stehen sie in jeder Liste als
-- zwei gleich aussehende Zeilen „Lambacher Schweizer 7", und wer eine davon an eine Klasse
-- ausgibt, weiß nicht, welche (docs/OFFEN.md 4.18, Stufe 1).
--
-- Die Angabe konnte bisher nur als Freitext-Schlüssel in erweiterte_eigenschaften stehen:
-- Der Listenimport legt unbekannte Spalten dort ab, geschrieben hat sie kein Code. Der
-- Wert zieht deshalb in die Spalte um UND der Schlüssel fällt — zwei Orte für dieselbe
-- Angabe wären die zweite Wahrheit, und beim nächsten Import gewönne mal der eine, mal
-- der andere.
--
-- 50 Zeichen reichen für die Auflagenbezeichnung („4., überarbeitete Auflage 2023"). Ein
-- längerer Altwert wird beim Umzug gekappt statt die Migration scheitern zu lassen; er
-- stünde sonst weiter unerreichbar im JSON.
ALTER TABLE buecher_titel ADD COLUMN IF NOT EXISTS auflage VARCHAR(50);

UPDATE buecher_titel
SET auflage = left(btrim(erweiterte_eigenschaften ->> 'auflage'), 50)
WHERE auflage IS NULL
  AND btrim(COALESCE(erweiterte_eigenschaften ->> 'auflage', '')) <> '';

UPDATE buecher_titel
SET erweiterte_eigenschaften = erweiterte_eigenschaften - 'auflage'
WHERE erweiterte_eigenschaften ? 'auflage';
