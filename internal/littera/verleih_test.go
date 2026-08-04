package littera

import (
	"strings"
	"testing"
	"time"
)

// Kopfzeile und Beispielsatz stammen aus dem echten Export (littera_sav.mdb).
const verleihCSV = `Buchungsdatum,Buchungsnummer,Exemplar,Leser,Verleihdatum,Rückgabedatum,Versäumnisdatum,IstRückgabedatum,Zurückgegeben,Verlängerung,Mahnungen
"08/03/01 11:41:58",16,4483,1676,"08/03/01 00:00:00","07/31/11 00:00:00","07/31/11 00:00:00",,0,0,0
"09/01/09 09:00:00",17,4484,1677,"09/01/09 00:00:00","09/15/09 00:00:00",,"09/10/09 00:00:00",1,0,2
"09/01/09 09:00:00",18,4485,1678,"09/01/09 00:00:00",,,,0,0,0
"09/01/09 09:00:00",19,,1679,"09/01/09 00:00:00","09/15/09 00:00:00",,,0,0,0
`

func TestLeseAusleihen(t *testing.T) {
	ausleihen, err := LeseAusleihen(strings.NewReader(verleihCSV))
	if err != nil {
		t.Fatalf("Ausleihen lesen: %v", err)
	}

	// Die Zeile ohne Exemplar faellt raus — ohne Buch ist nichts zu buchen.
	if len(ausleihen) != 3 {
		t.Fatalf("erwartet 3 Ausleihen, waren %d", len(ausleihen))
	}

	offen := ausleihen[0]
	if !offen.Offen() {
		t.Error("erste Zeile ist nicht zurueckgegeben, muss offen sein")
	}
	// DER Fallstrick: Rueckgabedatum ist die FRIST, nicht die Rueckgabe.
	if offen.Frist.Format("2006-01-02") != "2011-07-31" {
		t.Errorf("Frist falsch gelesen: %v", offen.Frist)
	}
	if !offen.RueckgabeAm.IsZero() {
		t.Errorf("offene Ausleihe darf kein Rueckgabedatum tragen: %v", offen.RueckgabeAm)
	}
	if offen.AusgeliehenAm.Format("2006-01-02") != "2001-08-03" {
		t.Errorf("Verleihdatum falsch (zweistelliges Jahr!): %v", offen.AusgeliehenAm)
	}

	zurueck := ausleihen[1]
	if zurueck.Offen() {
		t.Error("zweite Zeile ist zurueckgegeben")
	}
	if zurueck.RueckgabeAm.Format("2006-01-02") != "2009-09-10" {
		t.Errorf("IstRueckgabedatum falsch: %v", zurueck.RueckgabeAm)
	}
	if zurueck.Mahnungen != 2 {
		t.Errorf("Mahnungen falsch: %d", zurueck.Mahnungen)
	}

	// Ohne Frist: wird gemeldet, nicht stillschweigend gefuellt.
	fehlend := OhneFrist(ausleihen)
	if len(fehlend) != 1 || fehlend[0].ID != "18" {
		t.Errorf("Ausleihe ohne Frist nicht erkannt: %+v", fehlend)
	}
}

func TestNurOffene(t *testing.T) {
	ausleihen := []Ausleihe{
		{ID: "1", Zurueckgegeben: false},
		{ID: "2", Zurueckgegeben: true},
		{ID: "3", Zurueckgegeben: false},
	}
	offen := NurOffene(ausleihen)
	if len(offen) != 2 {
		t.Fatalf("erwartet 2 offene, waren %d", len(offen))
	}
}

// TestZurueckgegebenIstEigenesKennzeichen haelt fest, dass der Zustand NICHT aus dem
// Rueckgabedatum abgeleitet werden darf: Im Altbestand gibt es zurueckgegebene
// Ausleihen ohne eingetragenes IstRueckgabedatum. Wer daraus schliesst, fuehrt sie
// als offen — und das Buch waere in der neuen Anwendung dauerhaft blockiert.
func TestZurueckgegebenIstEigenesKennzeichen(t *testing.T) {
	const csv = `Buchungsnummer,Exemplar,Leser,Verleihdatum,Rückgabedatum,IstRückgabedatum,Zurückgegeben
20,100,200,"09/01/09 00:00:00","09/15/09 00:00:00",,1
`
	ausleihen, err := LeseAusleihen(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("lesen: %v", err)
	}
	if ausleihen[0].Offen() {
		t.Error("Zurueckgegeben=1 ohne Rueckgabedatum muss trotzdem als zurueckgegeben gelten")
	}
	if len(NurOffene(ausleihen)) != 0 {
		t.Error("diese Ausleihe darf nicht in der offenen Menge stehen")
	}
}

func TestDatumAus(t *testing.T) {
	faelle := map[string]string{
		"08/03/01 11:41:58": "2001-08-03", // zweistelliges Jahr < 69 -> 20xx
		"05/03/95 00:00:00": "1995-05-03", // >= 69 -> 19xx
		"12/24/10":          "2010-12-24", // auch ohne Uhrzeit
	}
	for roh, erwartet := range faelle {
		d, ok := DatumAus(roh)
		if !ok {
			t.Errorf("DatumAus(%q) nicht gelesen", roh)
			continue
		}
		if d.Format("2006-01-02") != erwartet {
			t.Errorf("DatumAus(%q) = %s, erwartet %s", roh, d.Format("2006-01-02"), erwartet)
		}
	}

	// Leer und Unsinn liefern KEINE Null-Zeit als gueltigen Wert: Ein 0001-01-01 saehe
	// in der Datenbank echt aus und faellt erst als 2000 Jahre alte Mahnung auf.
	for _, roh := range []string{"", "  ", "kein Datum", "31/31/31"} {
		if d, ok := DatumAus(roh); ok {
			t.Errorf("DatumAus(%q) haette scheitern muessen, lieferte %v", roh, d)
		}
	}
}

func TestWahrheitswert(t *testing.T) {
	for _, wahr := range []string{"1", "True", "true", "-1", "WAHR"} {
		if !wahrheitswert(wahr) {
			t.Errorf("%q muss wahr sein", wahr)
		}
	}
	for _, falsch := range []string{"0", "False", "", "nein"} {
		if wahrheitswert(falsch) {
			t.Errorf("%q muss falsch sein", falsch)
		}
	}
}

func TestAusleiheOffenIstGegenteilVonZurueckgegeben(t *testing.T) {
	if (Ausleihe{Zurueckgegeben: true}).Offen() {
		t.Error("zurueckgegeben darf nicht offen sein")
	}
	if !(Ausleihe{Zurueckgegeben: false}).Offen() {
		t.Error("nicht zurueckgegeben muss offen sein")
	}
	_ = time.Time{}
}
