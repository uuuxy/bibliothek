package littera

import (
	"io"
	"strings"
)

// Fremdnummern sind Barcodes, die NICHT von Littera vergeben wurden.
//
// Der Schülerausweis der Schule trägt die Nummer seines Herstellers — gemessen an einer
// echten Karte `B97601826457`, während darunter nur `[0395] 37` aufgedruckt steht.
// Littera merkt sich diese Zuordnung in zwei eigenen Tabellen, und nur dort:
//
//	FremdLeserNummer (FremdNummer Text(255), Leser Long)          → Ausweis → Leser
//	FremdBarcode     (Buchungsnummer Long, Exemplarnummer Text(50)) → Exemplar → Barcode
//
// Ohne sie wäre der Import blind für die Etiketten, die tatsächlich gescannt werden. Im
// Backup von 2010 sind beide Tabellen leer (die Karte gilt bis 2031, sie ist jünger als
// die Datei); in einer laufenden Installation sind sie die maßgebliche Quelle.
//
// Deshalb sind die beiden Dateien beim Einlesen optional — aber wenn sie da sind, gewinnen
// sie gegen die Littera-eigenen Nummern.

// LeseAusweisnummern liest `FremdLeserNummer` als Leser-Buchungsnummer → Ausweisnummer.
//
// Die Tabelle ist von Haus aus in der anderen Richtung geführt (eine Person kann mehrere
// Karten haben, etwa nach einem Verlust). Für den Import zählt die umgekehrte Richtung:
// Welche Nummer schreiben wir dieser Person auf den Ausweis? Bei mehreren gewinnt die
// zuletzt gelesene Zeile — Littera hängt neue Karten hinten an — und alle Kandidaten
// stehen in `mehrfach`, damit der Bericht sie nennen kann.
func LeseAusweisnummern(r io.Reader) (nummern map[string]string, mehrfach []string, err error) {
	zeilen, err := leseTabelle(r)
	if err != nil {
		return nil, nil, err
	}
	nummern = make(map[string]string, len(zeilen))
	for _, z := range zeilen {
		leser := strings.TrimSpace(z["Leser"])
		nummer := strings.TrimSpace(z["FremdNummer"])
		if leser == "" || nummer == "" {
			continue
		}
		if _, schonDa := nummern[leser]; schonDa {
			mehrfach = append(mehrfach, leser)
		}
		nummern[leser] = nummer
	}
	return nummern, mehrfach, nil
}

// LeseFremdbarcodes liest `FremdBarcode` als Exemplar-Buchungsnummer → Barcode.
//
// Betrifft Exemplare, die ein Ersatzetikett aus fremder Quelle bekommen haben (etwa
// verlagseigene Aufkleber). Steht hier etwas, klebt genau das am Buch — und nicht die
// Littera-Exemplarnummer.
func LeseFremdbarcodes(r io.Reader) (map[string]string, error) {
	zeilen, err := leseTabelle(r)
	if err != nil {
		return nil, err
	}
	barcodes := make(map[string]string, len(zeilen))
	for _, z := range zeilen {
		exemplar := strings.TrimSpace(z["Buchungsnummer"])
		barcode := strings.TrimSpace(z["Exemplarnummer"])
		if exemplar == "" || barcode == "" {
			continue
		}
		barcodes[exemplar] = barcode
	}
	return barcodes, nil
}
