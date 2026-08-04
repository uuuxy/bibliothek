package littera

import (
	"io"
	"strconv"
	"strings"
	"time"
)

// datumsFormate sind die Schreibweisen, in denen mdb-export Access-Zeitstempel ausgibt.
//
// Access speichert Datum und Zeit gemeinsam; der Export schreibt US-Reihenfolge
// (Monat/Tag/Jahr) mit ZWEISTELLIGEM Jahr. Go legt bei "06" die Grenze bei 69: 69–99
// werden 19xx, 00–68 werden 20xx. Das passt zu diesem Bestand — Geburtsdaten wie
// "05/03/95" sind 1995, Ausleihen wie "08/03/01" sind 2001.
var datumsFormate = []string{
	"01/02/06 15:04:05",
	"01/02/06",
}

// DatumAus liest einen Littera-Zeitstempel. Zweiter Rückgabewert ist false bei leerem
// oder unlesbarem Wert — der Aufrufer entscheidet dann, ob das ein Fehler ist.
//
// Bewusst keine „Notfall-Null-Zeit": Ein Datum, das als 0001-01-01 durchrutscht, sieht
// in der Datenbank wie ein gültiger Wert aus und fällt erst auf, wenn jemand eine
// 2000 Jahre überfällige Mahnung in der Hand hält.
func DatumAus(roh string) (time.Time, bool) {
	roh = strings.TrimSpace(roh)
	if roh == "" {
		return time.Time{}, false
	}
	for _, f := range datumsFormate {
		if t, err := time.Parse(f, roh); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// GeburtsdatumAus liest ein Geburtsdatum und holt es zurück in die Vergangenheit.
//
// Die Jahrhundertgrenze bei „06" liegt in Go fest auf 69: 69–99 werden 19xx, 00–68
// werden 20xx. Für Ausleihdaten stimmt das, für Geburtsdaten nicht. Gemessen am
// Altbestand landen 69 Personen — Lehrkräfte der Jahrgänge 1946 bis 1968 — dadurch in
// den Jahren 2046 bis 2068.
//
// Ein Geburtsdatum in der Zukunft ist immer falsch, also wird ein Jahrhundert abgezogen.
// jetzt wird übergeben, damit der Test nicht von der Systemuhr abhängt.
func GeburtsdatumAus(roh string, jetzt time.Time) (time.Time, bool) {
	t, ok := DatumAus(roh)
	if !ok {
		return time.Time{}, false
	}
	if t.After(jetzt) {
		t = t.AddDate(-100, 0, 0)
	}
	return t, true
}

// Ausleihe ist eine Zeile aus der Littera-Tabelle `Verleih`.
type Ausleihe struct {
	ID         string
	ExemplarID string // FK auf Exemplar.Buchungsnummer
	// LeserID zeigt auf Leser.BUCHUNGSNUMMER, nicht auf die Lesernummer — siehe die
	// Warnung am Typ Leser. Über die Lesernummer träfen nur 792 von 15.615 Ausleihen.
	LeserID string

	AusgeliehenAm  time.Time
	Frist          time.Time // Rückgabedatum = die Frist, nicht die Rückgabe
	RueckgabeAm    time.Time // IstRückgabedatum — nur gesetzt, wenn zurückgegeben
	Zurueckgegeben bool
	Mahnungen      int
}

// Offen sagt, ob das Buch noch beim Entleiher ist.
func (a Ausleihe) Offen() bool { return !a.Zurueckgegeben }

// LeseAusleihen liest die Tabelle `Verleih`.
//
// Zwei Fallstricke stecken in den Spaltennamen:
//   - `Rückgabedatum` ist die FRIST, nicht die Rückgabe. Die tatsächliche Rückgabe
//     steht in `IstRückgabedatum`. Wer die beiden vertauscht, bucht jede Ausleihe als
//     sofort zurückgegeben — und der Bestand stünde als vollständig da, während
//     tausende Bücher bei Schülern liegen.
//   - `Zurückgegeben` ist ein eigenes Kennzeichen und nicht aus dem Vorhandensein von
//     `IstRückgabedatum` ableitbar; im Altbestand gibt es zurückgegebene Ausleihen
//     ohne eingetragenes Rückgabedatum.
func LeseAusleihen(r io.Reader) ([]Ausleihe, error) {
	zeilen, err := leseTabelle(r)
	if err != nil {
		return nil, err
	}

	ausleihen := make([]Ausleihe, 0, len(zeilen))
	for _, z := range zeilen {
		exemplar := strings.TrimSpace(z["Exemplar"])
		leser := strings.TrimSpace(z["Leser"])
		if exemplar == "" || leser == "" {
			continue // ohne Buch oder ohne Entleiher ist die Zeile nicht buchbar
		}

		a := Ausleihe{
			ID:             strings.TrimSpace(z["Buchungsnummer"]),
			ExemplarID:     exemplar,
			LeserID:        leser,
			Zurueckgegeben: wahrheitswert(z["Zurückgegeben"]),
			Mahnungen:      zahlAus(z["Mahnungen"]),
		}
		a.AusgeliehenAm, _ = DatumAus(z["Verleihdatum"])
		a.Frist, _ = DatumAus(z["Rückgabedatum"])
		if r, ok := DatumAus(z["IstRückgabedatum"]); ok {
			a.RueckgabeAm = r
		}
		ausleihen = append(ausleihen, a)
	}
	return ausleihen, nil
}

// wahrheitswert liest Litteras Boolean-Export. mdb-export schreibt 0/1, je nach
// Fassung auch "True"/"False".
func wahrheitswert(roh string) bool {
	switch strings.ToLower(strings.TrimSpace(roh)) {
	case "1", "true", "wahr", "-1": // -1 ist Access' interne Wahrheit
		return true
	}
	return false
}

func zahlAus(roh string) int {
	n, err := strconv.Atoi(strings.TrimSpace(roh))
	if err != nil {
		return 0
	}
	return n
}

// NurOffene liefert die Ausleihen, bei denen das Buch noch beim Entleiher ist.
//
// Das ist die Menge, die beim Import wirklich zählt: Ohne sie startet das System mit
// „alles verfügbar", obwohl tausende Bücher bei Schülern liegen — und jede Ausleihe
// eines längst vergebenen Exemplars scheitert dann an der Theke.
func NurOffene(ausleihen []Ausleihe) []Ausleihe {
	var offen []Ausleihe
	for _, a := range ausleihen {
		if a.Offen() {
			offen = append(offen, a)
		}
	}
	return offen
}

// OhneExemplar meldet Ausleihen, deren Exemplar es im Bestand nicht (mehr) gibt.
//
// Im Altbestand betrifft das genau EINE von 15.615 Zeilen — ein Datenfehler in Littera,
// kein Zuordnungsfehler. Der Import muss solche Zeilen ueberspringen UND melden: Sie
// blind zu schreiben scheitert am Fremdschluessel und reisst den ganzen Lauf mit; sie
// stillschweigend zu verwerfen verschweigt, dass ein Buch verliehen war.
func OhneExemplar(ausleihen []Ausleihe, bekannteExemplare map[string]bool) []Ausleihe {
	var verwaist []Ausleihe
	for _, a := range ausleihen {
		if !bekannteExemplare[a.ExemplarID] {
			verwaist = append(verwaist, a)
		}
	}
	return verwaist
}

// OhneFrist meldet Ausleihen ohne lesbare Frist.
//
// ausleihen.rueckgabe_frist ist NOT NULL. Solche Zeilen brauchen vor dem Schreiben eine
// Regel (z. B. Verleihdatum + Standardleihfrist) — sie stillschweigend auf „heute" zu
// setzen wuerde tausende Buecher als ueberfaellig ausweisen.
func OhneFrist(ausleihen []Ausleihe) []Ausleihe {
	var fehlend []Ausleihe
	for _, a := range ausleihen {
		if a.Frist.IsZero() {
			fehlend = append(fehlend, a)
		}
	}
	return fehlend
}
