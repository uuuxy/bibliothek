package api

import (
	"strings"
	"testing"
)

// Der LUSD-Export der Schule hat keine Schüler-ID — und bekommt auch keine. Die Datei
// selbst entscheidet den Modus: ID-Spalte mit Werten → ID-Modus; sonst Namensmodus,
// und dann ist das Geburtsdatum in jeder Zeile Pflicht.

func TestParseLusdDatei_OhneIDSpalteIstNamensmodus(t *testing.T) {
	csv := "vorname;nachname;klasse;geburtsdatum\nMax;Mustermann;05a;01.02.2012\nErika;Musterfrau;6b;2011-03-04\n"
	datei, err := parseLusdDatei([]byte(csv))
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if datei.Modus != lusdModusName || datei.Modus.String() != "name_geburtsdatum" {
		t.Fatalf("erwartet Namensmodus, war %v", datei.Modus)
	}
	if len(datei.Zeilen) != 2 || datei.Zeilen[0].GebDatum == nil || datei.Zeilen[1].GebDatum == nil {
		t.Fatalf("Zeilen/Geburtsdaten falsch: %+v", datei.Zeilen)
	}
	if datei.Zeilen[0].LusdID != "" {
		t.Errorf("ohne ID-Spalte darf keine LUSD-ID stehen, war %q", datei.Zeilen[0].LusdID)
	}
}

func TestParseLusdDatei_LeereIDSpalteIstNamensmodus(t *testing.T) {
	// LUSD exportiert die Spalte mitunter leer — eine ID-Spalte ohne IDs ist keine.
	csv := "lusd_id,vorname,nachname,klasse,geburtsdatum\n,Max,Mustermann,5a,01.02.2012\n,Erika,Musterfrau,6b,02.03.2011\n"
	datei, err := parseLusdDatei([]byte(csv))
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if datei.Modus != lusdModusName {
		t.Fatalf("erwartet Namensmodus bei leerer ID-Spalte, war %v", datei.Modus)
	}
}

func TestParseLusdDatei_EineIDReichtFuerIDModus(t *testing.T) {
	// Gemischt: sobald eine echte ID drinsteht, gilt der ID-Modus; ID-lose Zeilen
	// bleiben erhalten und werden in der Klassifizierung als übersprungen gezählt.
	csv := "lusd_id,vorname,nachname,klasse,geburtsdatum\nL1,Max,Mustermann,5a,01.02.2012\n,Erika,Musterfrau,6b,02.03.2011\n"
	datei, err := parseLusdDatei([]byte(csv))
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if datei.Modus != lusdModusID || len(datei.Zeilen) != 2 {
		t.Fatalf("erwartet ID-Modus mit 2 Zeilen, war %v / %d", datei.Modus, len(datei.Zeilen))
	}
}

func TestParseLusdDatei_LanisKlassenlisteIstNurNameModus(t *testing.T) {
	// Echter Export aus dem Schulportal (LANIS): Semikolon, UTF-8-BOM, Kursspalten, weder
	// ID noch Geburtsdatum. Genau so liegt er im Sekretariat vor.
	csv := "\uFEFFNachname;Vorname;Klasse;BKU;Spanisch\nMustermann;Max;05G1;x;\nMusterfrau;Erika;05G1;;x\n"
	datei, err := parseLusdDatei([]byte(csv))
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if datei.Modus != lusdModusNurName || datei.Modus.String() != "name" {
		t.Fatalf("erwartet Nur-Name-Modus, war %v", datei.Modus)
	}
	if len(datei.Zeilen) != 2 || datei.Zeilen[0].Nachname != "Mustermann" || datei.Zeilen[0].Klasse != "05G1" {
		t.Fatalf("Zeilen falsch: %+v", datei.Zeilen)
	}
}

func TestParseLusdDatei_NurNameLegtGleicheNamenNichtZusammen(t *testing.T) {
	// Zwei Zeilen gleichen Namens sind im Nur-Name-Modus zwei Menschen, die sich nicht
	// auseinanderhalten lassen — beide bleiben erhalten (die Klassifizierung meldet sie).
	csv := "Nachname;Vorname;Klasse\nMustermann;Max;5a\nMustermann;Max;7b\n"
	datei, err := parseLusdDatei([]byte(csv))
	if err != nil {
		t.Fatal(err)
	}
	if len(datei.Zeilen) != 2 || datei.DublettenInDatei != 0 {
		t.Fatalf("Nur-Name darf nicht zusammenlegen: %d Zeilen, %d Dubletten", len(datei.Zeilen), datei.DublettenInDatei)
	}
}

func TestParseLusdDatei_NamensmodusLeeresGeburtsdatumBrichtHartAb(t *testing.T) {
	// Eine Zeile ohne lesbares Datum hat im Namensmodus keinen Schlüssel. Still
	// überspringen hieße: dieser Schüler würde jedes Jahr neu angelegt oder nie
	// gefunden. Deshalb harter Abbruch mit Zeile — OHNE Namen: Die Meldung landet über
	// SendHTTPError im Server-Log (Prüfung 22.08.2026); die Zeilennummer reicht zum Finden.
	for _, wert := range []string{"", "Unsinn", "31.02.2012"} {
		csv := "vorname,nachname,klasse,geburtsdatum\nMax,Mustermann,5a,01.02.2012\nErika,Musterfrau,6b," + wert + "\n"
		_, err := parseLusdDatei([]byte(csv))
		if err == nil {
			t.Fatalf("Geburtsdatum %q: erwartet Abbruch", wert)
		}
		if !strings.Contains(err.Error(), "zeile 3") || strings.Contains(err.Error(), "Musterfrau") {
			t.Errorf("Geburtsdatum %q: Meldung muss die Zeile nennen und den Namen NICHT (Log), war: %v", wert, err)
		}
	}
}

func TestParseLusdDatei_NamensmodusDublettenLetzteGewinnt(t *testing.T) {
	// Dieselbe Person zweimal (LUSD führt Schulformwechsler doppelt): die spätere
	// Zeile gewinnt an der ersten Position, die Dublette wird gezählt — nicht verschwiegen.
	csv := "vorname,nachname,klasse,geburtsdatum\nMax,Mustermann,5a,01.02.2012\nAnna,Beispiel,5b,03.04.2012\nmax,MUSTERMANN,6a,01.02.2012\n"
	datei, err := parseLusdDatei([]byte(csv))
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if len(datei.Zeilen) != 2 || datei.DublettenInDatei != 1 {
		t.Fatalf("erwartet 2 Zeilen und 1 Dublette, waren %d / %d", len(datei.Zeilen), datei.DublettenInDatei)
	}
	if datei.Zeilen[0].Klasse != "6a" || datei.Zeilen[1].Nachname != "Beispiel" {
		t.Errorf("letzte Zeile muss an erster Position gewinnen: %+v", datei.Zeilen)
	}
}

func TestParseLusdDatei_IDModusDublettenGezaehlt(t *testing.T) {
	csv := "lusd_id,vorname,nachname,klasse\nL1,Max,Mustermann,5a\nL1,Max,Mustermann,6a\n"
	datei, err := parseLusdDatei([]byte(csv))
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if datei.Modus != lusdModusID || len(datei.Zeilen) != 1 || datei.DublettenInDatei != 1 || datei.Zeilen[0].Klasse != "6a" {
		t.Fatalf("ID-Dedupe falsch: %+v (Dubletten %d)", datei.Zeilen, datei.DublettenInDatei)
	}
}

func TestKlassenNormkey(t *testing.T) {
	faelle := map[string]string{
		"05A": "5a", " 5a ": "5a", "5 a": "5a", "Q2": "q2", "007b": "7b", "E 1": "e1", "ABG": "abg", "10a": "10a",
	}
	for in, soll := range faelle {
		if got := klassenNormkey(in); got != soll {
			t.Errorf("klassenNormkey(%q) = %q, erwartet %q", in, got, soll)
		}
	}
	if !klassenGleich("05A", "5a") || klassenGleich("5a", "5b") {
		t.Error("klassenGleich entscheidet falsch")
	}
}
