package repository

import "strings"

// protokollSchluesselMitPersonenbezug sind die Schlüssel, die in einem Protokolleintrag neben
// der Kennung eines Lesers (details->>'schueler_id') einen Wert seiner Leserzeile oder Freitext
// über ihn tragen. Die Tilgung (spurTilgungen) nimmt sie beim Anonymisieren und beim
// endgültigen Löschen aus beiden Protokollen: dem Admin-Protokoll (audit_logs) und der
// Datensatz-Historie (audit_log). Der Eintrag selbst bleibt — Aktion, Zeit, Bearbeiter und
// die Kennung, die nach der Anonymisierung ein Pseudonym ist.
//
// Woher die Schlüssel kommen:
//   - lusd_id: die staatliche Schüler-ID (LUSD_ID_NACHGETRAGEN)
//   - barcode, aufgeloest_barcode: Ausweisnummern (SCHUELER_ZUSAMMENGEFUEHRT)
//   - grund, reason: der Sperrgrund (LESER_GESPERRT, LESER_ENTSPERRT; OVERRIDE_BLOCK bis
//     zum 24.09.2026) und der getippte Grund einer Stornierung (STORNIERUNG in audit_log)
//   - schuldner, beschreibung: Name und Freitext einer offenen Forderung, die mit ihrem
//     Titel gelöscht wurde (audit_books_forderung.go, inventur/db_books_delete_spur.go)
//   - betrifft: Name eines Lesers, dessen Vormerkung mit dem Titel fiel
//     (titel_loeschen_wartende.go)
//   - entleiher: Name aus der Spur einer laufenden Ausleihe, deren Titel gelöscht wurde
//   - vorname, nachname, email: Name und Adresse eines gelöschten Zugangskontos, dessen
//     Leserzeile stehen blieb (konto_loeschspur.go, seit dem 29.09.2026)
//   - ziel_id: die Kennung des Zugangskontos neben der des Lesers
//     (KOLLEGIUMSKONTO_NACHGETRAGEN). Das Konto trägt Name und Adresse; wie die Verknüpfung
//     in benutzer zeigt danach auch kein Eintrag mehr vom getilgten Leser auf ein Konto.
//
// Eine Liste für beide Tabellen. Bis zum 29.09.2026 hatte nur audit_logs eine solche
// Anweisung; Name und Freitext der Titel-Löschspur in audit_log überlebten die
// Anonymisierung bis zur Audit-Aufbewahrung (24 Monate).
var protokollSchluesselMitPersonenbezug = []string{
	"lusd_id", "barcode", "aufgeloest_barcode", "grund", "reason",
	"schuldner", "beschreibung", "betrifft", "entleiher",
	"vorname", "nachname", "email", "ziel_id",
}

// tilgePersonenbezugImProtokoll baut die Anweisung, die einer Protokolltabelle (audit_logs
// oder audit_log) diese Schlüssel nimmt. $1 = Leser-Kennungen als text[]. Die Schlüssel sind
// Konstanten dieser Datei, keine Eingabe.
func tilgePersonenbezugImProtokoll(tabelle string) string {
	schluessel := "ARRAY['" + strings.Join(protokollSchluesselMitPersonenbezug, "', '") + "']::text[]"
	return `UPDATE ` + tabelle + `
			SET details = details - ` + schluessel + `
			WHERE details ?| ` + schluessel + `
			  AND details->>'schueler_id' = ANY($1::text[])`
}

// kontoSchluesselMitPersonenbezug sind die Schlüssel, die in einem Eintrag über ein Zugangskonto
// (details->>'ziel_id': USER_CREATE, USER_UPDATE, SELBSTANMELDUNG) Name oder Adresse tragen.
// Die Tilgung nimmt sie aus den Einträgen der früheren Konten eines Lesers. Der Weg dorthin ist
// der Löscheintrag des Kontos, der den Leser nennt und dabei selbst Name und Adresse verliert
// (konto_loeschspur.go): Blieben sie hier stehen, führte von der Kennung des getilgten Lesers
// über die Kennung des Kontos ein Weg zu seinem Namen. Die Einträge eines bestehenden Kontos
// bleiben wie das Konto selbst; es gehört der Anlage, die Tilgung löst nur die Verknüpfung.
var kontoSchluesselMitPersonenbezug = []string{"vorname", "nachname", "email"}

// tilgePersonenbezugFruehererKonten baut die Anweisung dazu. $1 = Leser-Kennungen als text[].
// Die Schlüssel sind Konstanten dieser Datei, keine Eingabe.
func tilgePersonenbezugFruehererKonten() string {
	schluessel := "ARRAY['" + strings.Join(kontoSchluesselMitPersonenbezug, "', '") + "']::text[]"
	return `UPDATE audit_logs
			SET details = details - ` + schluessel + `
			WHERE details ?| ` + schluessel + `
			  AND details->>'ziel_id' IN (
				SELECT datensatz_id::text FROM audit_log
				WHERE tabelle = 'benutzer' AND aktion = 'DELETE'
				  AND details->>'schueler_id' = ANY($1::text[]))`
}
