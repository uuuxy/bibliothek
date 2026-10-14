package service

import "log"

// meldeProtokollFehler schreibt ins Log, wenn ein Eintrag der Verwaltung
// (ProtokolliereAdminAktion: übergangene Sperre, Wareneingang) nicht geschrieben werden konnte.
// Diese Einträge stehen neben dem Vorgang und nicht in seiner Transaktion, deshalb bricht ihr
// Fehler den Vorgang nicht ab. Ausleihe und Rückgabe schreiben ihren Eintrag dagegen in der
// Transaktion der Buchung, und ein Fehler dort lässt die Buchung scheitern.
func meldeProtokollFehler(aktion string, err error) {
	if err != nil {
		log.Printf("audit: %s konnte nicht protokolliert werden: %v", aktion, err)
	}
}
