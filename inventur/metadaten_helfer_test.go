package inventur

import (
	"strconv"
	"testing"
)

// Die Stufe gilt nur, wo der Titel sie als Schulstufe nennt: einen Jahrgang oder zwei
// verbundene. Der Vorschlag landet in der Jahrgangsspanne der Maske: Ein Band, ein Level oder
// eine bloße Zahl stünde dort als Jahrgang, mit dem Inventur und Portal rechnen. Die Formen
// stammen aus Titeln der DNB.
func TestAutomatischeKategorisierung(t *testing.T) {
	tests := []struct {
		name       string
		titel      string
		untertitel string
		wantFach   string
		wantVon    int
		wantBis    int
	}{
		{"Klasse nach dem Wort", "Mathematik für Gymnasien", "Klasse 7", "Mathematik", 7, 7},
		{"Jahrgangsstufe, groß geschrieben", "BIOLOGIE", "Jahrgangsstufe 10", "Biologie", 10, 10},
		{"Schuljahr als Ordnungszahl", "Deutschbuch", "7. Schuljahr", "Deutsch", 7, 7},
		{"Klasse als Ordnungszahl", "Geschichte entdecken", "10. Klasse", "Geschichte", 10, 10},
		{"Klassenstufe", "Chemie heute", "Klassenstufe 9", "Chemie", 9, 9},
		{"Jahreszahl vor dem Jahrgang", "Deutschbuch", "Ausgabe 2027 - 5. Schuljahr", "Deutsch", 5, 5},
		{"die erste Angabe gilt", "Chemie", "Klasse 8 (G8) bzw. Klassen 8/9 (G9)", "Chemie", 8, 8},

		{"zwei Klassen mit Schrägstrich", "Natur und Technik", "Klasse 7/8", "Naturwissenschaften", 7, 8},
		{"Klasse bis", "Erdkunde", "Klasse 7 bis 9", "Erdkunde", 7, 9},
		{"Klasse mit Punkt und Strich", "Lernhilfe", "Klasse 7.-8. Schuljahr", "", 7, 8},
		{"Mehrzahl mit Halbgeviertstrich", "Erdkunde", "Klassen 7–10", "Erdkunde", 7, 10},
		{"Mehrzahl mit Bindestrich", "Erdkunde", "Klassen 5-10", "Erdkunde", 5, 10},
		{"Mehrzahl mit und", "Chemie", "Klassenstufen 5 und 6", "Chemie", 5, 6},
		{"Mehrzahl, Strich mit Leerzeichen", "Lernhilfe", "Klassen 8 - 10", "", 8, 10},
		{"Oberstufe", "Mathematik", "Jahrgangsstufen 11-13", "Mathematik", 11, 13},
		{"zwei Schuljahre", "Deutschbuch", "5./6. Schuljahr", "Deutsch", 5, 6},
		{"Schuljahre bis", "Deutschbuch", "7. bis 10. Schuljahr", "Deutsch", 7, 10},
		{"Schuljahre mit Strich", "Deutschbuch", "5.-10. Schuljahr", "Deutsch", 5, 10},
		{"Schuljahre mit u.", "Deutschbuch", "9. u. 10. Schuljahr", "Deutsch", 9, 10},
		{"Jahreszahl vor der Spanne", "Chemie", "Ausgabe ab 2026 - 9./10. Schuljahr", "Chemie", 9, 10},
		{"Jahreszahl mit Schrägstrich vor der Spanne", "Chemie", "Berlin/Brandenburg 2015/7./8. Schuljahr", "Chemie", 7, 8},

		{"Band ist keine Klasse", "English G Access", "Band 2", "Englisch", 0, 0},
		{"Level ist keine Klasse", "Algebra und mehr", "Level 9", "Mathematik", 0, 0},
		{"Stufe allein ist keine Schulstufe", "Allgemeines Buch", "Stufe 8", "", 0, 0},
		{"bloße Zahl am Lehrwerk", "Green Line 5", "", "", 0, 0},
		{"bloße Zahl im Untertitel", "Découvertes", "für Französisch 6", "Französisch", 0, 0},
		{"Ausgabe mit Zahl", "Geschichte entdecken", "Ausgabe 12 Hessen", "Geschichte", 0, 0},
		{"Zahl im Romantitel", "Die 13½ Leben des Käpt'n Blaubär", "", "", 0, 0},

		{"Band vor dem Jahrgang", "Lernhilfe", "Band 5 - 9. Schuljahr", "", 0, 0},
		{"Einzahl, Strich mit Leerzeichen", "Erdkunde", "Klasse 7 - 8. Auflage", "Erdkunde", 0, 0},
		{"Schuljahre, Strich mit Leerzeichen", "Deutschbuch", "7. - 10. Schuljahr", "Deutsch", 0, 0},
		{"drei Klassen", "Erdkunde", "Klasse 5/6/7", "Erdkunde", 0, 0},
		{"drei Schuljahre", "Deutschbuch", "5./6./7. Schuljahr", "Deutsch", 0, 0},
		{"Aufzählung nach der Mehrzahl", "Chemie", "Inhalte der Klassenstufen 8, 9 und 10", "Chemie", 0, 0},
		{"Mehrzahl mit einer Zahl", "Lernhilfe", "Klassen 7", "", 0, 0},
		{"zwei Angaben mit oder", "Mathematik", "Klassen 10-12 oder 11-13", "Mathematik", 0, 0},
		{"zwei Angaben mit bzw.", "Biologie", "Klassen 11-12 bzw. 11-13", "Biologie", 0, 0},
		{"Spanne beginnt in der Grundschule", "Mathematik", "Klassen 4-6", "Mathematik", 0, 0},
		{"Halbjahr hinter der Stufe", "Mathematik", "Jahrgangsstufe 12/2", "Mathematik", 0, 0},
		{"fallende Folge", "Erdkunde", "Klasse 9/7", "Erdkunde", 0, 0},

		{"ab einer Klasse", "Wörterbuch", "ab Klasse 5", "", 0, 0},
		{"ab einer Klasse, als Ordnungszahl", "Lernhilfe", "ab der 7. Klasse", "", 0, 0},
		{"bis zu einer Klasse", "Erdkunde", "bis zur 10. Klasse", "Erdkunde", 0, 0},
		{"ab zwei Klassen", "Wörterbuch", "ab Klasse 5/6", "", 0, 0},

		{"Klasse der Grundschule", "Mathematik", "Klasse 4", "Mathematik", 0, 0},
		{"Schuljahr als Zeitraum", "Chemie", "Schuljahr 2026/27", "Chemie", 0, 0},
		{"Fach ohne Stufe", "Chemie Grundlagen", "", "Chemie", 0, 0},
		{"nichts erkannt", "Ein spannender Roman", "Teil Drei", "", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFach, gotVon, gotBis := automatischeKategorisierung(tt.titel, tt.untertitel)
			if gotFach != tt.wantFach {
				t.Errorf("Fach = %q, erwartet %q", gotFach, tt.wantFach)
			}
			if gotVon != tt.wantVon || gotBis != tt.wantBis {
				t.Errorf("Stufe = %d bis %d, erwartet %d bis %d", gotVon, gotBis, tt.wantVon, tt.wantBis)
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
		vorschlag := schulstufe(zahl) != 0
		liste := parseKlassenStufe(zahl) != 0
		if vorschlag != liste {
			t.Errorf("%d: ISBN-Abfrage nimmt an = %v, Listenimport nimmt an = %v", n, vorschlag, liste)
		}
	}
}
