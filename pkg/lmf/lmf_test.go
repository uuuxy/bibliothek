package lmf

import (
	"os"
	"strings"
	"testing"
)

func TestHatKennung(t *testing.T) {
	faelle := []struct {
		wert string
		want bool
	}{
		{"lmf-Deutsch 5", true},
		{"LMF-Deutsch 5", true},
		{"LMF - Deutsch 5", true}, // Leerzeichen um den Bindestrich
		{"LMF Deutsch 5", true},
		{"  lmf-Mathe", true}, // Leerraum davor
		{"LMF Bio 7 / Nat", true},
		{"Der kleine Hobbit", false},
		{"LMFP-Roman", false},      // kein Trenner nach lmf
		{"lmfao Witzebuch", false}, // kein Trenner nach lmf
		{"", false},
		{"lmf", false}, // Kürzel allein, ohne Trenner und Rest
		{"BIB Rom", false},
	}
	for _, f := range faelle {
		if got := HatKennung(f.wert); got != f.want {
			t.Errorf("HatKennung(%q) = %v, want %v", f.wert, got, f.want)
		}
	}
}

// Der Vermerk steht als eigenes Wort da, allein oder neben einem zweiten Ort.
func TestHatVermerk(t *testing.T) {
	faelle := map[string]bool{
		"LMF":               true,
		"lmf":               true,
		"LMF/Bibliothek":    true,
		"Bibliothek/LMF":    true,
		"Buch LMF Ma 6/Gri": true,
		"LMF Bio 7":         true,
		"Filmfest":          false,
		"Elmf":              false,
		"LMFP-Roman":        false,
		"Schulseelsorge":    false,
		"":                  false,
	}
	for wert, erwartet := range faelle {
		if got := HatVermerk(wert); got != erwartet {
			t.Errorf("HatVermerk(%q) = %v, erwartet %v", wert, got, erwartet)
		}
	}
}

// Migration 093 hat den Altbestand mit demselben Muster befüllt, das HatKennung liest. Laufen
// die zwei auseinander, erkennt der Import ein anderes Buch als Lernmittel als die Migration,
// und ein Schulbuch stünde im öffentlichen Katalog.
func TestHatKennung_MusterStehtInMigration093(t *testing.T) {
	sql, err := os.ReadFile("../../migrations/093_lernmittel_feld.sql")
	if err != nil {
		t.Fatalf("Migration lesen: %v", err)
	}
	const muster = `~ '^lmf[ -]'`
	if n := strings.Count(string(sql), muster); n != 2 {
		t.Errorf("Migration 093 enthält das Muster %s %d-mal, erwartet 2 (Titel und Signatur)", muster, n)
	}
	if got := kennung.String(); got != `(?i)^lmf[ -]` {
		t.Errorf("Go-Muster = %q — die Migration kennt nur ^lmf[ -] (auf LOWER(btrim()))", got)
	}
}

func TestZerlege(t *testing.T) {
	faelle := []struct {
		signatur string
		fach     string
		von, bis int
		ok       bool
	}{
		// Formen aus dem Katalog der Schule
		{"LMF Bio 7", FachBiologie, 7, 7, true},
		{"LMF Ma 5", FachMathematik, 5, 5, true},
		{"LMF PoWi 9", FachPoWi, 9, 9, true},
		{"LMF Eng 12/", FachEnglisch, 12, 12, true},
		{"LMF Ges 12/13", FachGeschichte, 12, 13, true},
		{"LMF Deu 1213", FachDeutsch, 12, 13, true},
		{"LMF ErdAt", FachErdkunde, 0, 0, true},
		{"LMF SpoSII", FachSport, 0, 0, true},
		{"LMF Spo SI", FachSport, 0, 0, true},
		{"LMF Ma Fo", FachMathematik, 0, 0, true},
		{"LMF Pusch", "", 0, 0, true}, // unbekanntes Kürzel: Lernmittel ja, Fach offen
		{"lmf eng 9", FachEnglisch, 9, 9, true},
		// Kürzel des Titels hinter dem Schrägstrich
		{"LMF Deu 7 / Bie", FachDeutsch, 7, 7, true},
		// Früherer Signatur-Vorschlag der Maske
		{"LMF M", FachMathematik, 0, 0, true},
		{"LMF-Ma 5", FachMathematik, 5, 5, true},
		{"LMF - Deutsch 5", FachDeutsch, 5, 5, true},
		// Keine Lernmittelsignatur
		{"Bio 7", "", 0, 0, false},
		{"JF", "", 0, 0, false},
		{"", "", 0, 0, false},
		// Zahlen außerhalb von 5 bis 13 zählen nicht: In „LMF Arb 1" ist die 1 der Band.
		{"LMF Ma 99", FachMathematik, 0, 0, true},
		{"LMF Arb 1", FachArbeitslehre, 0, 0, true},
	}
	for _, f := range faelle {
		z, ok := Zerlege(f.signatur)
		if ok != f.ok || z.Fach != f.fach || z.JahrgangVon != f.von || z.JahrgangBis != f.bis {
			t.Errorf("Zerlege(%q) = {%q %d %d} ok=%v, want {%q %d %d} ok=%v",
				f.signatur, z.Fach, z.JahrgangVon, z.JahrgangBis, ok, f.fach, f.von, f.bis, f.ok)
		}
	}
}

// „LMF Deutsch 5" (Klartext statt Kürzel) ergibt dasselbe Fach wie „LMF Deu 5"; sonst hinge
// das Fach an der Schreibweise der Signatur.
func TestZerlege_KlartextFachUeberStichwort(t *testing.T) {
	a, _ := Zerlege("LMF Deutsch 5")
	b, _ := Zerlege("LMF Deu 5")
	if a != b || a.Fach != FachDeutsch {
		t.Errorf("Klartext %+v, Kürzel %+v — erwartet gleich mit Fach %q", a, b, FachDeutsch)
	}
}

func TestFachAusText(t *testing.T) {
	faelle := []struct{ text, want string }{
		{"Mathematik für Gymnasien Klasse 7", FachMathematik},
		{"Algebra und mehr", FachMathematik},
		{"English G Access Band 2", FachEnglisch},
		{"BIOLOGIE Jahrgangsstufe 10", FachBiologie},
		{"Découvertes für Französisch 6", FachFranzoesisch},
		{"Chemie Grundlagen", FachChemie},
		{"Geschichte entdecken", FachGeschichte},
		{"Natur und Technik 5/6", FachNaWi},
		{"Diercke Erdkunde 7", FachErdkunde},
		{"Ein spannender Roman", ""},
		{"", ""},
	}
	for _, f := range faelle {
		if got := FachAusText(f.text); got != f.want {
			t.Errorf("FachAusText(%q) = %q, want %q", f.text, got, f.want)
		}
	}
}

// Bei zwei Fächern im Text entscheidet die Reihenfolge der Liste, in jedem Lauf gleich. Eine
// Map lieferte mal das eine, mal das andere.
func TestFachAusText_Deterministisch(t *testing.T) {
	erstes := FachAusText("Mathematik und Physik für die Oberstufe")
	for i := 0; i < 200; i++ {
		if got := FachAusText("Mathematik und Physik für die Oberstufe"); got != erstes {
			t.Fatalf("Lauf %d: %q, vorher %q", i, got, erstes)
		}
	}
	if erstes != FachMathematik {
		t.Errorf("erwartet %q (steht in der Liste vor Physik), war %q", FachMathematik, erstes)
	}
}

func TestFachAusSchlagworten(t *testing.T) {
	faelle := []struct {
		sw   []string
		want string
	}{
		{[]string{"Schulbuch", "Naturwissenschaften", "Physik"}, FachPhysik},
		{[]string{"Deutschunterricht", "Interpretation"}, FachDeutsch},
		{[]string{"Deutsch für Ausländer"}, ""}, // exakt, nicht enthalten
		{[]string{"Deutsch", "Englisch"}, ""},   // zwei Fächer: offen lassen
		{[]string{"Politik", "Politik und Wirtschaft"}, FachPoWi},
		{[]string{"Unterrichtsmaterialien"}, ""},
		{nil, ""},
	}
	for _, f := range faelle {
		if got := FachAusSchlagworten(f.sw); got != f.want {
			t.Errorf("FachAusSchlagworten(%v) = %q, want %q", f.sw, got, f.want)
		}
	}
}

func TestJahrgangAusZielgruppe(t *testing.T) {
	faelle := []struct {
		z        string
		von, bis int
	}{
		{"Sekundarstufe 1", 5, 10},
		{"Sekundarstufe 2", 11, 13},
		{"Sekundarstufe 1 u. 2", 5, 13},
		{"Gymnasiale Eingangsstufe", 5, 6},
		{"Förderstufe", 5, 6},
		{"Lehrer", 0, 0},
		{"Referendare", 0, 0},
		{"", 0, 0},
	}
	for _, f := range faelle {
		von, bis := jahrgangAusZielgruppe(f.z)
		if von != f.von || bis != f.bis {
			t.Errorf("jahrgangAusZielgruppe(%q) = %d–%d, want %d–%d", f.z, von, bis, f.von, f.bis)
		}
	}
}

// Ein Titel trägt in Littera bis zu fünf Zielgruppen; die Spanne kommt aus allen, nicht aus der
// letzten. Stünde „Lehrer" zuletzt, bliebe sie sonst leer.
func TestJahrgangAusZielgruppen(t *testing.T) {
	faelle := []struct {
		z        []string
		von, bis int
	}{
		{[]string{"Sekundarstufe 2", "Lehrer"}, 11, 13},
		{[]string{"Lehrer", "Sekundarstufe 2"}, 11, 13},
		{[]string{"Sekundarstufe 1", "Sekundarstufe 2"}, 5, 13},
		{[]string{"Förderstufe", "Sekundarstufe 1"}, 5, 10},
		{[]string{"Lehrer", "Referendare", "DAZ-Schüler"}, 0, 0},
		{nil, 0, 0},
	}
	for _, f := range faelle {
		von, bis := JahrgangAusZielgruppen(f.z)
		if von != f.von || bis != f.bis {
			t.Errorf("JahrgangAusZielgruppen(%q) = %d–%d, want %d–%d", f.z, von, bis, f.von, f.bis)
		}
	}
}

func TestFachExakt(t *testing.T) {
	faelle := map[string]string{
		"Mathe": FachMathematik, "mathematik": FachMathematik, "Deutsch": FachDeutsch,
		"Politik und Wirtschaft": FachPoWi, "Geographie": FachErdkunde, " Chemie ": FachChemie,
		// Litteras Kategorie enthält auch Standorttexte; sie sind keine Fächer.
		"Buch Pg/Kaf 078829": "", "Buch Che 11/Che 175 Exemplare": "", "Buch Deu 6/Cha 126 Exemplare 1. Auflage": "",
		"10 R / Ausgabe 2010": "", "": "",
	}
	for text, want := range faelle {
		if got := FachExakt(text); got != want {
			t.Errorf("FachExakt(%q) = %q, want %q", text, got, want)
		}
	}
}
