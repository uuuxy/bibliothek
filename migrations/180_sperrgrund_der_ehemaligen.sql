-- =============================================================================
-- Migration 180: Ein Sperrgrund der Ehemaligen, der nicht veraltet, und eine Form der Anonymisierung
-- =============================================================================
-- Der Grund eines Ehemaligen nannte seinen Zustand („wegen offener Vorgänge“, „in der Karenzzeit
-- vor der Anonymisierung“) und stand nach der letzten Rückgabe falsch in der Akte. Jeder
-- Ehemalige trägt jetzt denselben Grund, im Wort der Oberfläche (repository.AbgaengerSperrgrund);
-- wer wieder aktiv ist und noch etwas offen hat, den Grund mit den offenen Vorgängen
-- (repository.AbgaengerSperrgrundOffen). Ein Grund von Hand trägt das Präfix nicht und bleibt.
--
-- Die Uhr der Karenz rechnet bei einer Zeile ohne abgaenger_seit mit aktualisiert_am, und diese
-- Anweisung verschiebt aktualisiert_am. Der alte Wert wird deshalb in abgaenger_seit festgehalten.
--
-- Ohne Messung: Den Schulserver erreicht die Entwicklung nicht, und beide Anweisungen tragen jede
-- Form, die das Programm je geschrieben hat. Die erste nimmt jeden Grund mit dem alten Präfix,
-- auch die Fassungen mit Klammern vor Migration 179, und unterscheidet nur nach ist_abgaenger;
-- die zweite trifft genau die Form, die die Anonymisierung im Abgleich schrieb.
UPDATE schueler
   SET block_reason = CASE WHEN ist_abgaenger
                           THEN 'Automatisierte Ehemaligen-Sperre'
                           ELSE 'Automatisierte Ehemaligen-Sperre wegen offener Vorgänge' END,
       abgaenger_seit = CASE WHEN ist_abgaenger THEN COALESCE(abgaenger_seit, aktualisiert_am)
                             ELSE abgaenger_seit END
 WHERE block_reason LIKE 'Automatisierte Abgänger-Sperre%';

-- Die Anonymisierung im Abgleich mit Karenzzeit 0 schrieb eine eigene Form („Abgänger
-- Anonymisiert-…“, Klasse ABG); sie schreibt jetzt dieselbe wie der nächtliche Lauf
-- (repository.sqlAnonymisierung).
UPDATE schueler
   SET vorname = left(md5(random()::text), 8),
       nachname = 'Anonym',
       klasse = '',
       block_reason = 'Anonymisiert (DSGVO)'
 WHERE anonymized_at IS NOT NULL
   AND vorname = 'Abgänger'
   AND nachname LIKE 'Anonymisiert-%';
