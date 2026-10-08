package inventur

import (
	"strconv"
	"testing"
)

// Die Stufe gilt nur, wo der Titel sie als Schulstufe nennt und einen einzigen Jahrgang meint.
// Der Vorschlag landet in der Jahrgangsspanne der Maske: Ein Band, ein Level oder eine bloße
// Zahl stünde dort als Jahrgang, mit dem Inventur und Portal rechnen.
func TestAutomatischeKategorisierung(t *testing.T) {
	tests := []struct {
		name       string
		titel      string
		untertitel string
		wantFach   string
		wantStufe  string
	}{
		{"Klasse nach dem Wort", "Mathematik für Gymnasien", "Klasse 7", "Mathematik", "7"},
		{"Jahrgangsstufe, groß geschrieben", "BIOLOGIE", "Jahrgangsstufe 10", "Biologie", "10"},
		{"Schuljahr als Ordnungszahl", "Deutschbuch", "7. Schuljahr", "Deutsch", "7"},
		{"Klasse als Ordnungszahl", "Geschichte entdecken", "10. Klasse", "Geschichte", "10"},
		{"Klassenstufe", "Chemie heute", "Klassenstufe 9", "Chemie", "9"},

		{"Band ist keine Klasse", "English G Access", "Band 2", "Englisch", ""},
		{"Level ist keine Klasse", "Algebra und mehr", "Level 9", "Mathematik", ""},
		{"Stufe allein ist keine Schulstufe", "Allgemeines Buch", "Stufe 8", "", ""},
		{"bloße Zahl am Lehrwerk", "Green Line 5", "", "", ""},
		{"bloße Zahl im Untertitel", "Découvertes", "für Französisch 6", "Französisch", ""},
		{"Ausgabe mit Zahl", "Geschichte entdecken", "Ausgabe 12 Hessen", "Geschichte", ""},
		{"Zahl im Romantitel", "Die 13½ Leben des Käpt'n Blaubär", "", "", ""},

		{"zwei Klassen mit Schrägstrich", "Natur und Technik", "Klasse 7/8", "Naturwissenschaften", ""},
		{"Klassen bis", "Erdkunde", "Klasse 7 bis 9", "Erdkunde", ""},
		{"Mehrzahl mit Spanne", "Erdkunde", "Klassen 7–10", "Erdkunde", ""},
		{"zwei Schuljahre", "Deutschbuch", "5./6. Schuljahr", "Deutsch", ""},
		{"Schuljahre bis", "Deutschbuch", "7. bis 10. Schuljahr", "Deutsch", ""},

		{"ab einer Klasse", "Wörterbuch", "ab Klasse 5", "", ""},
		{"ab einer Klasse, als Ordnungszahl", "Lernhilfe", "ab der 7. Klasse", "", ""},
		{"bis zu einer Klasse", "Erdkunde", "bis zur 10. Klasse", "Erdkunde", ""},

		{"Klasse der Grundschule", "Mathematik", "Klasse 4", "Mathematik", ""},
		{"Schuljahr als Zeitraum", "Chemie", "Schuljahr 2026/27", "Chemie", ""},
		{"Fach ohne Stufe", "Chemie Grundlagen", "", "Chemie", ""},
		{"nichts erkannt", "Ein spannender Roman", "Teil Drei", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFach, gotStufe := automatischeKategorisierung(tt.titel, tt.untertitel)
			if gotFach != tt.wantFach {
				t.Errorf("Fach = %q, erwartet %q", gotFach, tt.wantFach)
			}
			if gotStufe != tt.wantStufe {
				t.Errorf("Stufe = %q, erwartet %q", gotStufe, tt.wantStufe)
			}
		})
	}
}

// Welche Zahl als Jahrgang der Schule gilt, steht an zwei Stellen: am Vorschlag der
// ISBN-Abfrage (schulstufe) und an der Spalte „klasse" des Listenimports (parseKlassenStufe).
// Beide tragen den Wert in dieselbe Spanne ein und müssen dieselben Zahlen annehmen.
func TestSchulstufe_WieDerListenimport(t *testing.T) {
	for n := -1; n <= 20; n++ {
		zahl := strconv.Itoa(n)
		vorschlag := schulstufe(zahl) != ""
		liste := parseKlassenStufe(zahl) != 0
		if vorschlag != liste {
			t.Errorf("%d: ISBN-Abfrage nimmt an = %v, Listenimport nimmt an = %v", n, vorschlag, liste)
		}
	}
}
