package api

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
)

// Die Quelle: CSV oder Excel, Kopfzeile gesucht statt vorausgesetzt. Ob die Schule
// wirklich eine Excel-Datei bekommt, ist offen — beide müssen gehen.

// baueXlsx erzeugt eine Arbeitsmappe im Test; rows sind die Zellen je Zeile, Zellen vom
// Typ time.Time werden als echte Datumszelle geschrieben (Excel-Serienzahl).
func baueXlsx(t *testing.T, blaetter map[string][][]any) []byte {
	t.Helper()
	f := excelize.NewFile()
	erstes := true
	for name, rows := range blaetter {
		if erstes {
			if err := f.SetSheetName("Sheet1", name); err != nil {
				t.Fatal(err)
			}
			erstes = false
		} else if _, err := f.NewSheet(name); err != nil {
			t.Fatal(err)
		}
		for r, row := range rows {
			for c, wert := range row {
				zelle, err := excelize.CoordinatesToCellName(c+1, r+1)
				if err != nil {
					t.Fatal(err)
				}
				if err := f.SetCellValue(name, zelle, wert); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestParseLusdDatei_XlsxMitTitelzeilenUndDatumszellen(t *testing.T) {
	geb := time.Date(2012, 3, 4, 0, 0, 0, 0, time.UTC)
	xlsx := baueXlsx(t, map[string][][]any{
		"Klassenliste": {
			{"Klassenliste 05G1 — Schuljahr 2026/27"}, // Titelzeile über der Kopfzeile
			{},
			{"Nachname", "Vorname", "Klasse", "Geburtsdatum"},
			{"Mustermann", "Max", "05G1", geb},            // echte Datumszelle (Serienzahl)
			{"Musterfrau", "Erika", "05G1", "05.06.2011"}, // Datum als Text
			{}, // Leerzeile am Ende
		},
	})
	datei, err := parseLusdDatei(xlsx)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if datei.Modus != lusdModusName || len(datei.Zeilen) != 2 {
		t.Fatalf("erwartet Name+Geburtsdatum mit 2 Zeilen, war %v / %d", datei.Modus, len(datei.Zeilen))
	}
	if datei.Zeilen[0].GebDatum == nil || !datei.Zeilen[0].GebDatum.Equal(geb) {
		t.Errorf("Excel-Datumszelle falsch gelesen: %v", datei.Zeilen[0].GebDatum)
	}
	if datei.Zeilen[1].GebDatum == nil || datei.Zeilen[1].GebDatum.Format("2006-01-02") != "2011-06-05" {
		t.Errorf("Text-Datum falsch gelesen: %v", datei.Zeilen[1].GebDatum)
	}
	if datei.Zeilen[0].LineNum != 4 {
		t.Errorf("Zeilennummer muss die Excel-Zeile sein (4), war %d", datei.Zeilen[0].LineNum)
	}
}

func TestParseLusdDatei_XlsxNimmtDasBlattMitKopfzeile(t *testing.T) {
	// Das erste Blatt ist ein Deckblatt ohne Tabelle; die Daten stehen im zweiten.
	xlsx := baueXlsx(t, map[string][][]any{
		"Deckblatt": {{"Philipp-Reis-Schule"}, {"Export vom 01.08.2026"}},
		"Schueler":  {{"Vorname", "Nachname", "Klasse"}, {"Max", "Mustermann", "6a"}},
	})
	datei, err := parseLusdDatei(xlsx)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if len(datei.Zeilen) != 1 || datei.Zeilen[0].Klasse != "6a" || datei.Modus != lusdModusNurName {
		t.Fatalf("falsches Blatt gelesen: %+v (%v)", datei.Zeilen, datei.Modus)
	}
}

func TestParseLusdDatei_XlsxOhneKopfzeileMeldetPflichtspalte(t *testing.T) {
	xlsx := baueXlsx(t, map[string][][]any{"Tabelle1": {{"foo", "bar"}, {"1", "2"}}})
	_, err := parseLusdDatei(xlsx)
	if err == nil || !strings.Contains(err.Error(), "Pflichtspalte") {
		t.Fatalf("erwartet Pflichtspalten-Meldung, war: %v", err)
	}
}

func TestParseLusdDatei_AltesXlsWirdMitAnleitungAbgewiesen(t *testing.T) {
	xls := append([]byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}, bytes.Repeat([]byte{0}, 64)...)
	_, err := parseLusdDatei(xls)
	if err == nil || !strings.Contains(err.Error(), ".xlsx") {
		t.Fatalf("erwartet Hinweis auf .xlsx/CSV, war: %v", err)
	}
}

func TestParseLusdDatei_CsvMitTitelzeileUeberKopfzeile(t *testing.T) {
	csv := "Klassenliste 2026/27\n\nNachname;Vorname;Klasse;Geburtsdatum\nMustermann;Max;5a;1.2.2012\n"
	datei, err := parseLusdDatei([]byte(csv))
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if len(datei.Zeilen) != 1 || datei.Zeilen[0].GebDatum == nil || datei.Zeilen[0].LineNum != 4 {
		t.Fatalf("Zeile falsch: %+v", datei.Zeilen)
	}
}

func TestParseLusdDatei_BinaermuellBleibtVerstaendlich(t *testing.T) {
	_, err := parseLusdDatei([]byte{0xff, 0xfe, 0x00, 0x9c, 0x01, 0x02, 0x03})
	if err == nil {
		t.Fatal("Binärmüll muss eine Meldung liefern, keinen Erfolg")
	}
}

// Eine Klassenliste mit LUSD-Feldkürzeln (26.08.2026, kein LUSD-Export): je Klasse ein Blatt, gleiche Kopfzeile im
// SLR_/KLA_-Kürzelstil, kein Geburtsdatum. Alle Blätter müssen ankommen — vorher nur
// das erste, und im Nur-Name-Modus hätten die übrigen Klassen als Abgänger gegolten.
func TestParseLusdDatei_XlsxLiestAlleBlaetterMitKopfzeile(t *testing.T) {
	kopf := []any{"SLR_Nachname", "SLR_Vorname", "SLR_Strasse", "SLR_PLZ", "SLR_ORT", "KLA_Klassennamen", "KLA_KlassenlehrerKuerzel"}
	xlsx := baueXlsx(t, map[string][][]any{
		"6F1": {kopf, {"Adler", "Ava", "Weg 1", "61381", "Ort", "06F1", "FLA"}, {"Beck", "Ben", "Weg 2", "61381", "Ort", "06F1", "FLA"}},
		"6F2": {kopf, {"Cato", "Cem", "Weg 3", "61381", "Ort", "06F2", "XY"}},
		// Deckblatt ohne Tabelle bleibt unsichtbar.
		"Hinweise": {{"Stand März 2026"}, {"Nur für den Dienstgebrauch"}},
		// Ein Blatt mit ANDERER Spaltenreihenfolge muss trotzdem richtig landen.
		"6F3": {{"KLA_Klassennamen", "SLR_Vorname", "SLR_Nachname"}, {"06F3", "Dana", "Dorn"}},
	})
	datei, err := parseLusdDatei(xlsx)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if datei.Modus != lusdModusNurName {
		t.Errorf("ohne ID und Geburtsdatum muss der Nur-Name-Modus gelten, war %v", datei.Modus)
	}
	if len(datei.Zeilen) != 4 {
		t.Fatalf("erwartet 4 Schüler aus drei Blättern, bekam %d: %+v", len(datei.Zeilen), datei.Zeilen)
	}
	klassen := map[string]string{}
	for _, z := range datei.Zeilen {
		klassen[z.Nachname] = z.Klasse
	}
	for name, klasse := range map[string]string{"Adler": "06F1", "Beck": "06F1", "Cato": "06F2", "Dorn": "06F3"} {
		if klassen[name] != klasse {
			t.Errorf("%s: Klasse %q erwartet, %q bekommen — Blatt verschluckt oder Spalten vertauscht", name, klasse, klassen[name])
		}
	}
	for _, z := range datei.Zeilen {
		if z.Nachname == "Dorn" && z.Vorname != "Dana" {
			t.Errorf("umsortiertes Blatt: Vorname %q statt Dana", z.Vorname)
		}
	}
}
