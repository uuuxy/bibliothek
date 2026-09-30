-- =============================================================================
-- Migration 153: Sonderkonten sind eine Art des Lesers (docs/OFFEN.md 5.18)
-- =============================================================================
-- Entschieden am 30.09.2026: Praktikum, Sekretariat, U-plus und Fachbereich stehen als Art
-- neben Schüler, Lehrkraft und LiV. Littera führt diese Konten als eigene Lesergruppen; die
-- Übernahme brachte sie seit dem 28.09.2026 als „Lehrkraft" mit, und was sie wirklich sind,
-- stand nur im Protokoll des Laufs. Das Programm unterscheidet an jeder Stelle nur Schüler
-- und Nicht-Schüler (art = 'schueler'); für die vier gilt deshalb alles, was für das
-- Kollegium gilt: keine Frist, keine Mahnung, keine Forderung.
--
-- Geplant war am 29.09.2026 eine eigene Gruppe neben der Art, in der Tabelle lesergruppen.
-- Zwei Felder für dieselbe Frage hätten Paare wie „LiV + Sekretariat" erlaubt. Die Tabelle
-- lesergruppen (Migration 009) hat nie ein Schreibweg gefüllt und kein Go-Code gelesen;
-- gemessen am 29.09.2026 am Testserver: leer. Sie fällt weg.
--
-- Ohne Messung: Die Migration ändert keine Zeile. Die Sonderkonten einer früheren Übernahme
-- bleiben „lehrkraft", bis jemand die Art in der Akte ändert.
-- =============================================================================

ALTER TABLE leser DROP CONSTRAINT IF EXISTS chk_leser_art;
ALTER TABLE leser
    ADD CONSTRAINT chk_leser_art CHECK (art IN ('schueler', 'lehrkraft', 'liv', 'praktikum',
                                                'sekretariat', 'uplus', 'fachbereich'));

COMMENT ON COLUMN leser.art IS
    'Art des Lesers: schueler | lehrkraft | liv | praktikum | sekretariat | uplus | fachbereich. '
    'Entscheidet NICHT über Rechte — nur darüber, wer vom LUSD-Abgleich und vom Löschjob '
    'erfasst wird (art = ''schueler''). Praktikum und Fachbereich bekommen kein Zugangskonto '
    '(repository.ArtOhneKonto).';

DROP TABLE IF EXISTS lesergruppen;
