package lusd

import (
	"bibliothek/pkg/xlsxgrenze"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"github.com/xuri/excelize/v2"
)

// Die Quelle des LUSD-Imports: CSV oder Excel. Was die Schule wirklich bekommt, ist
// nicht verlässlich bekannt — der LANIS-Klassenlisten-Export ist eine Semikolon-CSV
// (`Nachname;Vorname;Klasse;…`, UTF-8 mit BOM), LUSD-Berichte kommen als .xlsx mit
// Titelzeilen über der Kopfzeile. Deshalb: beide Formate, Kopfzeile wird gesucht,
// nicht vorausgesetzt.

// maxKopfzeilenSuche begrenzt, wie viele Zeilen vor der Kopfzeile stehen dürfen
// (Titel, Schuljahr, Leerzeilen). LUSD-Berichte brauchen 1–3.
const maxKopfzeilenSuche = 10

var (
	xlsxSignatur = []byte("PK\x03\x04")
	xlsSignatur  = []byte{0xD0, 0xCF, 0x11, 0xE0}
)

// tabellenZeile ist eine Zeile der Quelle samt ihrer echten Zeilennummer in der Datei
// (CSV: Dateizeile inkl. übersprungener Leerzeilen, Excel: Blattzeile). Die Nummer steht
// in jeder Fehlermeldung — das Sekretariat muss die Zeile in seiner Datei finden können.
type tabellenZeile struct {
	nr     int
	zellen []string
}

// leseTabelle liest die Datei als Zeilenraster — Excel (.xlsx, am Zip-Kopf erkannt,
// nicht am Dateinamen) oder CSV. Altes Binär-Excel (.xls) wird mit Anleitung abgewiesen.
func leseTabelle(inhalt []byte) ([]tabellenZeile, error) {
	switch {
	case bytes.HasPrefix(inhalt, xlsxSignatur):
		return leseXLSXZeilen(inhalt)
	case bytes.HasPrefix(inhalt, xlsSignatur):
		return nil, fmt.Errorf("Die Datei ist ein altes Excel-Binärformat (.xls). Bitte in Excel „Speichern unter“ → .xlsx oder CSV wählen und erneut hochladen.") //nolint:staticcheck // ST1005: nutzer-sichtbarer Text
	default:
		return leseCSVZeilen(inhalt)
	}
}

// leseCSVZeilen liest eine CSV mit automatischer Trennzeichen-Erkennung. Die Zeilen
// dürfen unterschiedlich lang sein (FieldsPerRecord -1): Titelzeilen über der
// Kopfzeile haben weniger Felder, und spaltenWert fängt kurze Zeilen ab.
func leseCSVZeilen(inhalt []byte) ([]tabellenZeile, error) {
	text := strings.TrimPrefix(string(inhalt), "\uFEFF")
	trenner := ','
	if strings.Count(text, ";") > strings.Count(text, ",") {
		trenner = ';'
	}
	csvLeser := csv.NewReader(strings.NewReader(text))
	csvLeser.Comma = trenner
	csvLeser.LazyQuotes = true
	csvLeser.FieldsPerRecord = -1

	var tabelle []tabellenZeile
	for gelesen := 1; ; gelesen++ {
		zellen, err := csvLeser.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("zeile %d der CSV-Datei ist nicht lesbar: %w", gelesen, err)
		}
		// FieldPos liefert die Dateizeile — der Reader überspringt Leerzeilen still,
		// ein eigener Zähler läge danach daneben.
		nr, _ := csvLeser.FieldPos(0)
		tabelle = append(tabelle, tabellenZeile{nr: nr, zellen: zellen})
	}
	if len(tabelle) == 0 {
		return nil, fmt.Errorf("fehler beim lesen der csv-kopfzeile: die Datei ist leer")
	}
	return tabelle, nil
}

// leseXLSXZeilen liefert die Zeilen aller Blätter, die eine Kopfzeile tragen, als eine
// Tabelle im Spaltenbild des ersten Blatts mit Kopfzeile. Eine Klassenliste hat je Klasse ein
// Blatt; wer nur das erste läse, machte im Nur-Name-Modus aus den übrigen Klassen Abgänger.
//
// Blätter ohne Kopfzeile (Deckblatt) werden übersprungen. Spätere Blätter dürfen die Spalten
// in anderer Reihenfolge tragen und werden umsortiert. Gelesen werden Rohwerte: Datumszellen
// kommen als Excel-Serienzahl und werden in leseDatum zurückgerechnet.
func leseXLSXZeilen(inhalt []byte) ([]tabellenZeile, error) {
	// Nur das Lesen der Blätter liegt hinter der Schranke; die Kopfzeilen-Suche darunter
	// arbeitet auf fertigen Zeichenketten und darf keinen eigenen Fehler an sie verlieren.
	blaetter, err := xlsxgrenze.LeseMappe(bytes.NewReader(inhalt), func(f *excelize.File) ([][]tabellenZeile, error) {
		var alle [][]tabellenZeile
		for _, blatt := range f.GetSheetList() {
			tabelle, err := leseXLSXBlatt(f, blatt)
			if err != nil {
				return nil, err
			}
			alle = append(alle, tabelle)
		}
		return alle, nil
	})
	if err != nil {
		if xlsxgrenze.IstUnlesbar(err) {
			return nil, fmt.Errorf("Excel-Datei konnte nicht geöffnet werden: %w", err)
		}
		return nil, err
	}

	return fuegeBlaetterZusammen(blaetter)
}

// fuegeBlaetterZusammen sammelt aus den Blättern die mit Kopfzeile zu einer Tabelle; ohne
// ein solches Blatt kommen die Zeilen des ersten nicht leeren zurück, damit die
// Kopfzeilen-Meldung dort entsteht.
func fuegeBlaetterZusammen(blaetter [][]tabellenZeile) ([]tabellenZeile, error) {
	var ersteZeilen []tabellenZeile
	var gesamt xlsxTabelle
	for _, tabelle := range blaetter {
		if len(tabelle) == 0 {
			continue
		}
		if ersteZeilen == nil {
			ersteZeilen = tabelle
		}
		kopfNr, kopf, err := findeKopfzeile(tabelle)
		if err != nil || kopfNr < 0 {
			continue
		}
		gesamt.nimm(tabelle, kopfNr, kopf)
	}
	if gesamt.zeilen != nil {
		return gesamt.zeilen, nil
	}
	if ersteZeilen == nil {
		return nil, fmt.Errorf("fehler beim lesen der csv-kopfzeile: die Excel-Datei enthält kein Blatt mit Daten")
	}
	return ersteZeilen, nil // die Kopfzeilen-Meldung formuliert findeKopfzeile am ersten Blatt
}

// leseXLSXBlatt liest ein Blatt als Rohwerte und nummeriert seine Zeilen ab 1.
func leseXLSXBlatt(f *excelize.File, blatt string) ([]tabellenZeile, error) {
	roh, err := f.GetRows(blatt, excelize.Options{RawCellValue: true})
	if err != nil {
		return nil, fmt.Errorf("Excel-Blatt %q konnte nicht gelesen werden: %w", blatt, err)
	}
	tabelle := make([]tabellenZeile, len(roh))
	for i, zellen := range roh {
		tabelle[i] = tabellenZeile{nr: i + 1, zellen: zellen}
	}
	return tabelle, nil
}

// xlsxTabelle fügt die Blätter mit Kopfzeile zu einer Tabelle zusammen. Das erste gibt das
// Spaltenbild vor, an dem der Parser seine Indizes abliest.
type xlsxTabelle struct {
	zeilen []tabellenZeile
	kopf   map[string]int
	breite int
}

// nimm übernimmt das erste Blatt ganz und von jedem weiteren die Zeilen hinter der Kopfzeile,
// umsortiert auf das Spaltenbild des ersten.
func (t *xlsxTabelle) nimm(tabelle []tabellenZeile, kopfNr int, kopf map[string]int) {
	if t.zeilen == nil {
		t.zeilen, t.kopf, t.breite = tabelle, kopf, len(tabelle[kopfNr].zellen)
		return
	}
	for _, z := range tabelle[kopfNr+1:] {
		t.zeilen = append(t.zeilen, tabellenZeile{nr: z.nr, zellen: umsortiert(z.zellen, kopf, t.kopf, t.breite)})
	}
}

// umsortiert legt die Zellen eines späteren Blatts in das Spaltenbild des ersten:
// Für jede erkannte Spalte des ersten Blatts steht der Wert an dessen Index. Spalten,
// die der Parser nicht kennt, fallen weg — er liest ohnehin nur über die Kopfzeile.
func umsortiert(zellen []string, kopf, ersterKopf map[string]int, breite int) []string {
	ergebnis := make([]string, breite)
	for spalte, ziel := range ersterKopf {
		if quelle, ok := kopf[spalte]; ok && quelle < len(zellen) && ziel < breite {
			ergebnis[ziel] = zellen[quelle]
		}
	}
	return ergebnis
}

// findeKopfzeile sucht in den ersten Zeilen die Kopfzeile: die erste, die Vor- und
// Nachname-Spalte trägt. Findet sich keine, wird die Pflichtspalten-Prüfung auf die
// erste Zeile losgelassen — so bleibt die Meldung konkret („Pflichtspalte 'vorname'
// fehlt …") statt eines vagen „Kopfzeile nicht gefunden".
func findeKopfzeile(tabelle []tabellenZeile) (int, map[string]int, error) {
	for i := 0; i < len(tabelle) && i < maxKopfzeilenSuche; i++ {
		if !istKopfzeile(tabelle[i].zellen) {
			continue
		}
		spalten, err := ordneSpaltenZu(tabelle[i].zellen)
		if err != nil {
			return i, nil, err
		}
		return i, spalten, nil
	}
	_, err := ordneSpaltenZu(tabelle[0].zellen)
	if err == nil {
		err = fmt.Errorf("Pflichtspalte '%s' fehlt in der CSV-Kopfzeile — ist das die richtige LUSD-Exportdatei?", spalteVorname) //nolint:staticcheck // ST1005: nutzer-sichtbarer Text
	}
	return -1, nil, err
}

func istKopfzeile(zellen []string) bool {
	hatVorname, hatNachname := false, false
	for _, zelle := range zellen {
		switch spalteZuKopf[normiereKopf(zelle)] {
		case spalteVorname:
			hatVorname = true
		case spalteNachname:
			hatNachname = true
		}
	}
	return hatVorname && hatNachname
}
