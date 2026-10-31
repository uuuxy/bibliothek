package lusd

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"bibliothek/internal/xlsxtest"
	"bibliothek/pkg/csvutil"
)

// Die Quelle: CSV oder Excel, Kopfzeile gesucht statt vorausgesetzt. Ob die Schule
// wirklich eine Excel-Datei bekommt, ist offen — beide müssen gehen.

func TestLeseDatei_XLSXMitTitelzeilenUndDatumszellen(t *testing.T) {
	geb := time.Date(2012, 3, 4, 0, 0, 0, 0, time.UTC)
	xlsx := xlsxtest.Baue(t, map[string][][]any{
		"Klassenliste": {
			{"Klassenliste 05G1 — Schuljahr 2026/27"}, // Titelzeile über der Kopfzeile
			{},
			{"Nachname", "Vorname", "Klasse", "Geburtsdatum"},
			{"Mustermann", "Max", "05G1", geb},            // echte Datumszelle (Serienzahl)
			{"Musterfrau", "Erika", "05G1", "05.06.2011"}, // Datum als Text
			{}, // Leerzeile am Ende
		},
	})
	datei, err := LeseDatei(xlsx)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if datei.Modus != ModusName || len(datei.Zeilen) != 2 {
		t.Fatalf("LeseDatei(xlsx) = Modus %v mit %d Zeilen, erwartet Name+Geburtsdatum mit 2 Zeilen", datei.Modus, len(datei.Zeilen))
	}
	if datei.Zeilen[0].Geburtsdatum == nil || !datei.Zeilen[0].Geburtsdatum.Equal(geb) {
		t.Errorf("Excel-Datumszelle falsch gelesen: %v", datei.Zeilen[0].Geburtsdatum)
	}
	if datei.Zeilen[1].Geburtsdatum == nil || datei.Zeilen[1].Geburtsdatum.Format("2006-01-02") != "2011-06-05" {
		t.Errorf("Text-Datum falsch gelesen: %v", datei.Zeilen[1].Geburtsdatum)
	}
	if datei.Zeilen[0].Nr != 4 {
		t.Errorf("Zeilennummer muss die Excel-Zeile sein (4), war %d", datei.Zeilen[0].Nr)
	}
}

func TestLeseDatei_XLSXNimmtDasBlattMitKopfzeile(t *testing.T) {
	// Das erste Blatt ist ein Deckblatt ohne Tabelle; die Daten stehen im zweiten.
	xlsx := xlsxtest.Baue(t, map[string][][]any{
		"Deckblatt": {{"Philipp-Reis-Schule"}, {"Export vom 01.08.2026"}},
		"Schueler":  {{"Vorname", "Nachname", "Klasse"}, {"Max", "Mustermann", "6a"}},
	})
	datei, err := LeseDatei(xlsx)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if len(datei.Zeilen) != 1 || datei.Zeilen[0].Klasse != "6a" || datei.Modus != ModusNurName {
		t.Fatalf("falsches Blatt gelesen: %+v (%v)", datei.Zeilen, datei.Modus)
	}
}

func TestLeseDatei_XLSXOhneKopfzeileMeldetPflichtspalte(t *testing.T) {
	xlsx := xlsxtest.Baue(t, map[string][][]any{"Tabelle1": {{"foo", "bar"}, {"1", "2"}}})
	_, err := LeseDatei(xlsx)
	var spalte *PflichtspalteFehler
	if !errors.As(err, &spalte) || spalte.Spalte != spalteVorname {
		t.Fatalf("LeseDatei = %v, erwartet PflichtspalteFehler für %q", err, spalteVorname)
	}
}

// Die Anleitung, wie aus der Datei eine lesbare wird, formuliert die Tür
// (api/leser/lusd_datei_ablehnung_pg_test.go).
func TestLeseDatei_AltesXlsWirdAbgewiesen(t *testing.T) {
	xls := append([]byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}, bytes.Repeat([]byte{0}, 64)...)
	_, err := LeseDatei(xls)
	if !errors.Is(err, ErrAltesExcel) {
		t.Fatalf("LeseDatei = %v, erwartet ErrAltesExcel", err)
	}
}

func TestLeseDatei_CSVMitTitelzeileUeberKopfzeile(t *testing.T) {
	csv := "Klassenliste 2026/27\n\nNachname;Vorname;Klasse;Geburtsdatum\nMustermann;Max;5a;1.2.2012\n"
	datei, err := LeseDatei([]byte(csv))
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if len(datei.Zeilen) != 1 || datei.Zeilen[0].Geburtsdatum == nil || datei.Zeilen[0].Nr != 4 {
		t.Fatalf("Zeile falsch: %+v", datei.Zeilen)
	}
}

// Excel speichert „CSV (Trennzeichen-getrennt)" unter Windows in Windows-1252: Das ü ist dort das
// eine Byte 0xFC, auch in der Kopfzeile. Ohne Umwandlung stünde „J�rgen" in der Vorschau, und die
// Datenbank lehnte die Übernahme ab.
func TestLeseDatei_CSVAusExcelInWindows1252(t *testing.T) {
	csv := "Nachname;Vorname;Klasse;Geburtsdatum;Stra\xdfe\r\nM\xfcller;J\xfcrgen;5a;1.2.2012;Gr\xfcner Weg\r\n"
	datei, err := LeseDatei([]byte(csv))
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if len(datei.Zeilen) != 1 {
		t.Fatalf("%d Zeilen, erwartet 1", len(datei.Zeilen))
	}
	z := datei.Zeilen[0]
	if z.Vorname != "Jürgen" || z.Nachname != "Müller" || z.Strasse != "Grüner Weg" {
		t.Errorf("Vorname %q, Nachname %q, Straße %q", z.Vorname, z.Nachname, z.Strasse)
	}
}

// Excel für Mac schreibt „CSV“ in Macintosh, „CSV (Macintosh)“ trennt zudem nur mit CR.
func TestLeseDatei_CSVAusExcelFuerMac(t *testing.T) {
	for name, trenner := range map[string]string{"CSV": "\r\n", "CSV (Macintosh)": "\r"} {
		csv := "Nachname;Vorname;Klasse;Geburtsdatum;Stra\xa7e" + trenner + "M\x9fller;J\x9frgen;5a;1.2.2012;Gr\x9fner Weg" + trenner
		datei, err := LeseDatei([]byte(csv))
		if err != nil {
			t.Fatalf("%s: unerwarteter Fehler: %v", name, err)
		}
		if len(datei.Zeilen) != 1 {
			t.Fatalf("%s: %d Zeilen, erwartet 1", name, len(datei.Zeilen))
		}
		z := datei.Zeilen[0]
		if z.Vorname != "Jürgen" || z.Nachname != "Müller" || z.Strasse != "Grüner Weg" {
			t.Errorf("%s: Vorname %q, Nachname %q, Straße %q", name, z.Vorname, z.Nachname, z.Strasse)
		}
	}
}

func TestLeseDatei_UnklareKodierungWirdGemeldet(t *testing.T) {
	_, err := LeseDatei([]byte("Nachname;Vorname;Klasse;Geburtsdatum\r\nKre\x9aimir;Ana;5a;1.2.2012\r\n"))
	if !errors.Is(err, csvutil.ErrKodierung) {
		t.Errorf("Fehler = %v, erwartet ErrKodierung", err)
	}
}

func TestLeseDatei_BinaermuellBleibtVerstaendlich(t *testing.T) {
	_, err := LeseDatei([]byte{0xff, 0xfe, 0x00, 0x9c, 0x01, 0x02, 0x03})
	if err == nil {
		t.Fatal("Binärmüll muss eine Meldung liefern, keinen Erfolg")
	}
}

// Eine Klassenliste mit den Feldkürzeln der LUSD (kein Export der LUSD): je Klasse ein
// Blatt, gleiche Kopfzeile im SLR_/KLA_-Kürzelstil, kein Geburtsdatum. Alle Blätter müssen
// ankommen; käme nur das erste an, gälten im Nur-Name-Modus die übrigen Klassen als Abgänger.
func TestLeseDatei_XLSXLiestAlleBlaetterMitKopfzeile(t *testing.T) {
	kopf := []any{"SLR_Nachname", "SLR_Vorname", "SLR_Strasse", "SLR_PLZ", "SLR_ORT", "KLA_Klassennamen", "KLA_KlassenlehrerKuerzel"}
	xlsx := xlsxtest.Baue(t, map[string][][]any{
		"6F1": {kopf, {"Adler", "Ava", "Weg 1", "61381", "Ort", "06F1", "FLA"}, {"Beck", "Ben", "Weg 2", "61381", "Ort", "06F1", "FLA"}},
		"6F2": {kopf, {"Cato", "Cem", "Weg 3", "61381", "Ort", "06F2", "XY"}},
		// Deckblatt ohne Tabelle bleibt unsichtbar.
		"Hinweise": {{"Stand März 2026"}, {"Nur für den Dienstgebrauch"}},
		// Ein Blatt mit anderer Spaltenreihenfolge muss trotzdem richtig landen.
		"6F3": {{"KLA_Klassennamen", "SLR_Vorname", "SLR_Nachname"}, {"06F3", "Dana", "Dorn"}},
	})
	datei, err := LeseDatei(xlsx)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if datei.Modus != ModusNurName {
		t.Errorf("ohne ID und Geburtsdatum muss der Nur-Name-Modus gelten, war %v", datei.Modus)
	}
	if len(datei.Zeilen) != 4 {
		t.Fatalf("len(datei.Zeilen) = %d, erwartet 4 Schüler aus drei Blättern: %+v", len(datei.Zeilen), datei.Zeilen)
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

// Steht über der Kopfzeile jedes Blatts eine Titelzeile, gibt trotzdem die Kopfzeile die
// Breite der Tabelle vor. An der Titelzeile abgelesen, bliebe von den Zeilen des zweiten
// Blatts nur eine Spalte übrig.
func TestLeseDatei_XLSXTitelzeileUeberJedemBlatt(t *testing.T) {
	xlsx := xlsxtest.Baue(t, map[string][][]any{
		"6F1": {{"Klasse 6F1"}, {"Nachname", "Vorname", "Klasse"}, {"Adler", "Ava", "06F1"}},
		"6F2": {{"Klasse 6F2"}, {"Klasse", "Vorname", "Nachname"}, {"06F2", "Cem", "Cato"}},
	})
	datei, err := LeseDatei(xlsx)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	gelesen := map[string]string{}
	for _, z := range datei.Zeilen {
		gelesen[z.Nachname] = z.Vorname + " " + z.Klasse
	}
	if len(datei.Zeilen) != 2 || gelesen["Adler"] != "Ava 06F1" || gelesen["Cato"] != "Cem 06F2" {
		t.Errorf("gelesen = %v, erwartet Adler (Ava 06F1) und Cato (Cem 06F2)", gelesen)
	}
}
