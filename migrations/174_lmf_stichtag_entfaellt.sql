-- =============================================================================
-- Migration 174: Die Einstellung lmf_stichtag entfällt
-- =============================================================================
-- Der letzte Rückgabetag der Schulbücher kommt aus den Sommerferien: bis Jahrgang 10 der
-- Donnerstag vor den Ferien, in der Oberstufe der 31. Juli (repository/lmf_stichtag.go). Seitdem
-- liest und schreibt keine Stelle den Wert „MM-TT" der Einstellung lmf_stichtag.
--
-- Gemessen am Testserver am 15.11.2026 (lesend, beim Eintrag des Verdachts in OFFEN 5.181): Die
-- Einstellung steht auf 07-31, der Vorgabe. Die Fristen offener Ausleihen fasst die Migration
-- nicht an.
--
-- Nicht umkehrbar: Der Wert steht danach nur noch in der Sicherung, die update.sh vor dem
-- Einspielen anlegt.
--
-- Idempotent: Beim zweiten Lauf fehlt die Zeile schon.
-- =============================================================================

DELETE FROM system_einstellungen WHERE schluessel = 'lmf_stichtag';
