package api

import (
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

// leseXlsxZeilen liefert die Zeilen des ERSTEN Blatts, das eine Kopfzeile trägt — ein
// Bericht kann mehrere Blätter haben (z. B. je Schulzweig), und das aktive ist nicht
// zwingend das richtige. Rohwerte (RawCellValue): Datumszellen kommen als Excel-Serien-
// zahl und werden in parseLUSDGebDatum zurückgerechnet — die formatierte Ansicht hinge
// sonst an der Zellenformatierung des Exportierenden.
func leseXlsxZeilen(content []byte) ([]tabellenZeile, error) {
	f, err := excelize.OpenReader(bytes.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("Excel-Datei konnte nicht geöffnet werden: %w", err)
	}
	defer closeutil.LogClose(f, "lusd xlsx")

	var ersteZeilen []tabellenZeile
	for _, blatt := range f.GetSheetList() {
		raw, err := f.GetRows(blatt, excelize.Options{RawCellValue: true})
		if err != nil {
			return nil, fmt.Errorf("Excel-Blatt %q konnte nicht gelesen werden: %w", blatt, err)
		}
		if len(raw) == 0 {
			continue
		}
		rows := make([]tabellenZeile, len(raw))
		for i, r := range raw {
			rows[i] = tabellenZeile{nr: i + 1, zellen: r}
		}
		if ersteZeilen == nil {
			ersteZeilen = rows
		}
		if idx, _, err := findeKopfzeile(rows); err == nil && idx >= 0 {
			return rows, nil
		}
	}
	if ersteZeilen == nil {
		return nil, fmt.Errorf("fehler beim lesen der csv-kopfzeile: die Excel-Datei enthält kein Blatt mit Daten")
	}
	return ersteZeilen, nil // die Kopfzeilen-Meldung formuliert findeKopfzeile am ersten Blatt
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
