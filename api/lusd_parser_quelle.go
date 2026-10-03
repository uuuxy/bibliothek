package api

import (
	"bibliothek/pkg/xlsxgrenze"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"bibliothek/pkg/closeutil"

	"github.com/xuri/excelize/v2"
)

// Die Quelle des LUSD-Imports: CSV oder Excel. Was die Schule wirklich bekommt, ist
// nicht verlässlich bekannt — der LANIS-Klassenlisten-Export ist eine Semikolon-CSV
// (`Nachname;Vorname;Klasse;…`, UTF-8 mit BOM), LUSD-Berichte kommen als .xlsx mit
// Titelzeilen ÜBER der Kopfzeile. Deshalb: beide Formate, Kopfzeile wird gesucht,
// nicht vorausgesetzt.

// maxKopfzeilenSuche begrenzt, wie viele Zeilen vor der Kopfzeile stehen dürfen
// (Titel, Schuljahr, Leerzeilen). LUSD-Berichte brauchen 1–3.
const maxKopfzeilenSuche = 10

var (
	xlsxSignatur = []byte("PK\x03\x04")
	xlsSignatur  = []byte{0xD0, 0xCF, 0x11, 0xE0}
)

// tabellenZeile ist eine Zeile der Quelle samt ihrer ECHTEN Zeilennummer in der Datei
// (CSV: Dateizeile inkl. übersprungener Leerzeilen, Excel: Blattzeile). Die Nummer steht
// in jeder Fehlermeldung — das Sekretariat muss die Zeile in seiner Datei finden können.
type tabellenZeile struct {
	nr     int
	zellen []string
}

// leseLusdTabelle liest die Datei als Zeilenraster — Excel (.xlsx, am Zip-Kopf erkannt,
// nicht am Dateinamen) oder CSV. Altes Binär-Excel (.xls) wird mit Anleitung abgewiesen.
func leseLusdTabelle(content []byte) ([]tabellenZeile, error) {
	switch {
	case bytes.HasPrefix(content, xlsxSignatur):
		return leseXlsxZeilen(content)
	case bytes.HasPrefix(content, xlsSignatur):
		return nil, fmt.Errorf("Die Datei ist ein altes Excel-Binärformat (.xls). Bitte in Excel „Speichern unter“ → .xlsx oder CSV wählen und erneut hochladen.") //nolint:staticcheck // ST1005: nutzer-sichtbarer Text
	default:
		return leseCsvZeilen(content)
	}
}

// leseCsvZeilen liest eine CSV mit automatischer Trennzeichen-Erkennung. Die Zeilen
// dürfen unterschiedlich lang sein (FieldsPerRecord -1): Titelzeilen über der
// Kopfzeile haben weniger Felder, und spaltenWert fängt kurze Zeilen ab.
func leseCsvZeilen(content []byte) ([]tabellenZeile, error) {
	contentStr := strings.TrimPrefix(string(content), "\uFEFF")
	delimiter := ','
	if strings.Count(contentStr, ";") > strings.Count(contentStr, ",") {
		delimiter = ';'
	}
	reader := csv.NewReader(strings.NewReader(contentStr))
	reader.Comma = delimiter
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1

	var rows []tabellenZeile
	for gelesen := 1; ; gelesen++ {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("zeile %d der CSV-Datei ist nicht lesbar: %w", gelesen, err)
		}
		// FieldPos liefert die Dateizeile — der Reader überspringt Leerzeilen still,
		// ein eigener Zähler läge danach daneben.
		nr, _ := reader.FieldPos(0)
		rows = append(rows, tabellenZeile{nr: nr, zellen: row})
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("fehler beim lesen der csv-kopfzeile: die Datei ist leer")
	}
	return rows, nil
}

// leseXlsxZeilen liefert die Zeilen aller Blätter, die eine Kopfzeile tragen, als eine
// Tabelle im Spaltenbild des ersten Blatts mit Kopfzeile. Eine Klassenliste hat je Klasse ein
// Blatt; wer nur das erste läse, machte im Nur-Name-Modus aus den übrigen Klassen Abgänger.
//
// Blätter ohne Kopfzeile (Deckblatt) werden übersprungen. Spätere Blätter dürfen die Spalten
// in anderer Reihenfolge tragen und werden umsortiert. Gelesen werden Rohwerte: Datumszellen
// kommen als Excel-Serienzahl und werden in parseLUSDDatum zurückgerechnet.
func leseXlsxZeilen(content []byte) ([]tabellenZeile, error) {
	f, err := excelize.OpenReader(bytes.NewReader(content), xlsxgrenze.Optionen())
	if err != nil {
		return nil, fmt.Errorf("Excel-Datei konnte nicht geöffnet werden: %w", err)
	}
	defer closeutil.LogClose(f, "lusd xlsx")

	var ersteZeilen []tabellenZeile
	var gesamt xlsxTabelle
	for _, blatt := range f.GetSheetList() {
		rows, err := leseXlsxBlatt(f, blatt)
		if err != nil {
			return nil, err
		}
		if len(rows) == 0 {
			continue
		}
		if ersteZeilen == nil {
			ersteZeilen = rows
		}
		idx, kopf, err := findeKopfzeile(rows)
		if err != nil || idx < 0 {
			continue
		}
		gesamt.nimm(rows, idx, kopf)
	}
	if gesamt.zeilen != nil {
		return gesamt.zeilen, nil
	}
	if ersteZeilen == nil {
		return nil, fmt.Errorf("fehler beim lesen der csv-kopfzeile: die Excel-Datei enthält kein Blatt mit Daten")
	}
	return ersteZeilen, nil // die Kopfzeilen-Meldung formuliert findeKopfzeile am ersten Blatt
}

// leseXlsxBlatt liest ein Blatt als Rohwerte und nummeriert seine Zeilen ab 1.
func leseXlsxBlatt(f *excelize.File, blatt string) ([]tabellenZeile, error) {
	raw, err := f.GetRows(blatt, excelize.Options{RawCellValue: true})
	if err != nil {
		return nil, fmt.Errorf("Excel-Blatt %q konnte nicht gelesen werden: %w", blatt, err)
	}
	rows := make([]tabellenZeile, len(raw))
	for i, r := range raw {
		rows[i] = tabellenZeile{nr: i + 1, zellen: r}
	}
	return rows, nil
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
func (t *xlsxTabelle) nimm(rows []tabellenZeile, idx int, kopf map[string]int) {
	if t.zeilen == nil {
		t.zeilen, t.kopf, t.breite = rows, kopf, len(rows[idx].zellen)
		return
	}
	for _, z := range rows[idx+1:] {
		t.zeilen = append(t.zeilen, tabellenZeile{nr: z.nr, zellen: umsortiert(z.zellen, kopf, t.kopf, t.breite)})
	}
}

// umsortiert legt die Zellen eines späteren Blatts in das Spaltenbild des ersten:
// Für jede erkannte Spalte des ersten Blatts steht der Wert an dessen Index. Spalten,
// die der Parser nicht kennt, fallen weg — er liest ohnehin nur über die Kopfzeile.
func umsortiert(zellen []string, kopf, ersterKopf map[string]int, breite int) []string {
	out := make([]string, breite)
	for col, zielIdx := range ersterKopf {
		if quellIdx, ok := kopf[col]; ok && quellIdx < len(zellen) && zielIdx < breite {
			out[zielIdx] = zellen[quellIdx]
		}
	}
	return out
}

// findeKopfzeile sucht in den ersten Zeilen die Kopfzeile: die erste, die Vor- UND
// Nachname-Spalte trägt. Findet sich keine, wird die Pflichtspalten-Prüfung auf die
// erste Zeile losgelassen — so bleibt die Meldung konkret („Pflichtspalte 'vorname'
// fehlt …") statt eines vagen „Kopfzeile nicht gefunden".
func findeKopfzeile(rows []tabellenZeile) (int, map[string]int, error) {
	for i := 0; i < len(rows) && i < maxKopfzeilenSuche; i++ {
		if !siehtAusWieKopfzeile(rows[i].zellen) {
			continue
		}
		headerMap, err := lusdHeaderMap(rows[i].zellen)
		if err != nil {
			return i, nil, err
		}
		return i, headerMap, nil
	}
	_, err := lusdHeaderMap(rows[0].zellen)
	if err == nil {
		err = fmt.Errorf("Pflichtspalte '%s' fehlt in der CSV-Kopfzeile — ist das die richtige LUSD-Exportdatei?", lusdColVorname) //nolint:staticcheck // ST1005: nutzer-sichtbarer Text
	}
	return -1, nil, err
}

func siehtAusWieKopfzeile(row []string) bool {
	hatVorname, hatNachname := false, false
	for _, h := range row {
		switch lusdHeaderLookup[normalizeHeader(h)] {
		case lusdColVorname:
			hatVorname = true
		case lusdColNachname:
			hatNachname = true
		}
	}
	return hatVorname && hatNachname
}
