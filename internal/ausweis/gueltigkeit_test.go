package ausweis

import "testing"

// Die Schreibweisen der Klassen stammen aus der Schule: "05F1" oder "5F1", "06G2" oder
// "6G2", "7H1", "8G2", "9R1", in der Oberstufe "E1", "E2" und "Q1" bis "Q4". Mit und ohne
// führende Null ergibt sich dasselbe; LUSD und Littera liefern die Klasse verschieden.
func TestGueltigBisJahr(t *testing.T) {
	const schuljahrEnde = 2027 // Schuljahr 2026/27

	faelle := []struct {
		klasse string
		will   int
		warum  string
	}{
		// Förderstufe: Zweig steht noch nicht fest, angesetzt wird der längste Weg
		// der Mittelstufe (10). Ein zu früher Ablauf machte den Ausweis eines
		// künftigen Realschülers vorzeitig ungültig.
		{"5F1", 2032, "Jahrgang 5 → Ende 10, also fünf Schuljahre"},
		{"05F1", 2032, "führende Null identisch"},
		{"6F3", 2031, "Jahrgang 6 → Ende 10"},
		{"06G2", 2031, "Förderstufe wird auch mit G-Zug geführt"},

		// Hauptschulzweig endet mit 9.
		{"7H1", 2029, "Jahrgang 7 → Ende 9"},
		{"8H2", 2028, "Jahrgang 8 → Ende 9"},
		{"9H1", 2027, "bereits im Abschlussjahrgang"},

		// Real- und Gymnasialzweig enden für den Ausweis beide mit 10; der Gymnasialzweig
		// bekommt für die Oberstufe einen neuen.
		{"7R1", 2030, "Jahrgang 7 → Ende 10"},
		{"9R1", 2028, "Jahrgang 9 → Ende 10"},
		{"7G1", 2030, "Gymnasialzweig: Ausweis bis Ende Mittelstufe, nicht bis 13"},
		{"8G2", 2029, "Jahrgang 8 → Ende 10"},
		{"10R2", 2027, "Abschlussjahrgang selbst"},
		{"10G2", 2027, "Gymnasialzweig im Abschlussjahrgang der Mittelstufe"},

		// Das freiwillige zehnte Jahr im Hauptschulzweig (Weg zum Mittleren Abschluss). Der
		// Zweig endet sonst mit 9; ein Zehntklässler darin ist trotzdem kein Datenfehler und
		// braucht in diesem Jahr einen neuen Ausweis.
		{"10H1", 2027, "10. Hauptschuljahr: gilt bis zum Ende dieses Jahrgangs"},
		{"11H1", 2027, "auch darüber hinaus wird kein Datum in der Vergangenheit erzeugt"},

		// Ohne erkennbaren Zweigbuchstaben gilt der längste Weg der Mittelstufe (10) —
		// dieselbe Regel wie für die Förderstufe. Der Ausweis eines Hauptschülers gilt
		// dann ein Jahr länger als nötig; das ist folgenlos, weil mit dem Abgang das
		// Konto gesperrt wird. Andersherum würde eine gültige Karte abgewiesen.
		{"10a", 2027, "Demo-/Kurzschema: Jahrgang 10 → Ende 10"},
		{"5a", 2032, "Jahrgang 5 ohne Zweig → längster Mittelstufenweg"},
		{"9d", 2028, "Jahrgang 9 ohne Zweig → Ende 10"},
		{"7X1", 2030, "unbekannter Zweigbuchstabe wie kein Zweigbuchstabe"},
		{"7", 2030, "Jahrgang ohne jeden Zusatz"},

		// Oberstufe (Hessen G9): E = Jahrgang 11, Q1/Q2 = 12, Q3/Q4 = 13,
		// Abitur am Ende von 13.
		{"E1", 2029, "Einführungsphase ist Jahrgang 11 → zwei Jahre bis 13"},
		{"E2", 2029, "E2 liegt im selben Jahrgang wie E1"},
		{"Q1", 2028, "Q1 liegt in Jahrgang 12"},
		{"Q2", 2028, "Q2 ebenfalls Jahrgang 12"},
		{"Q3", 2027, "Q3 liegt in Jahrgang 13"},
		{"Q4", 2027, "Q4 ebenfalls Jahrgang 13"},

		// Die Oberstufe in der Schreibweise dieser Schule: ET, 12T, 13T. 12T und 13T beginnen
		// mit einer Ziffer und kommen deshalb nicht über die Erkennung von E und Q. Gälte „T"
		// als unbekannter Zweig, liefe der Ausweis eines Schülers in 12T ein Jahr zu früh ab,
		// und das fiele erst an der Ausleihe auf.
		{"12T1", 2028, "Q-Phase dieser Schule: Jahrgang 12 → Ende 13"},
		{"13T1", 2027, "Jahrgang 13 ist der Abschlussjahrgang selbst"},
		{"12T", 2028, "auch ohne Zug"},
		{"ET", 2029, "Einführungsphase mit T — dieselbe Regel wie E1/E2"},

		// Schreibweise darf egal sein.
		{"7h1", 2029, "Kleinschreibung"},
		{"  8G2  ", 2029, "Leerzeichen"},
		{"e1", 2029, "Oberstufe kleingeschrieben"},
		{"12t1", 2028, "Q-Phase kleingeschrieben"},
	}

	for _, f := range faelle {
		jahr, ok := GueltigBisJahr(f.klasse, schuljahrEnde)
		if !ok {
			t.Errorf("%q: nicht erkannt, erwartet %d (%s)", f.klasse, f.will, f.warum)
			continue
		}
		if jahr != f.will {
			t.Errorf("%q: %d, erwartet %d (%s)", f.klasse, jahr, f.will, f.warum)
		}
	}
}

// Was sich nicht ableiten lässt, wird nicht geraten. Ein erfundenes Datum auf einem Ausweis
// fiele erst auf, wenn die Karte an der Ausleihe abgewiesen wird. Der Druckdialog fragt in
// diesen Fällen nach.
func TestUnbekannteKlassenWerdenNichtGeraten(t *testing.T) {
	unbekannt := []string{
		"",           // leer
		"   ",        // nur Leerzeichen
		"Vorkurs",    // kein Jahrgang
		"0F1",        // Jahrgang 0
		"14G1",       // Jahrgang jenseits der Schulzeit
		"Klasse 7H1", // Text vor dem Jahrgang
	}
	for _, k := range unbekannt {
		if jahr, ok := GueltigBisJahr(k, 2027); ok {
			t.Errorf("%q wurde als %d erkannt — unbekannte Klassen dürfen nichts liefern", k, jahr)
		}
	}
}

// Der Jahrgang, mit dem der Ausweis abläuft, ist die Regel; das Kalenderjahr ist die
// Umrechnung darauf. Beides ist getrennt geprüft, damit ein Fehler in der Rechnung mit Jahren
// nicht wie ein Fehler der Regel aussieht.
func TestAblaufJahrgang(t *testing.T) {
	faelle := []struct {
		klasse          string
		ablauf, aktuell int
	}{
		{"5F1", AbschlussMittelstufe, 5},
		{"7H1", AbschlussHauptschule, 7},
		{"7R1", AbschlussMittelstufe, 7},
		{"7G1", AbschlussMittelstufe, 7},
		{"E1", AbschlussOberstufe, 11},
		{"Q1", AbschlussOberstufe, 12},
		{"Q4", AbschlussOberstufe, 13},
	}
	for _, f := range faelle {
		ablauf, aktuell, ok := AblaufJahrgang(f.klasse)
		if !ok {
			t.Errorf("%q: nicht erkannt", f.klasse)
			continue
		}
		if ablauf != f.ablauf || aktuell != f.aktuell {
			t.Errorf("%q: Ablauf %d / aktuell %d, erwartet %d / %d",
				f.klasse, ablauf, aktuell, f.ablauf, f.aktuell)
		}
	}
}

// Das Schuljahr ist ein Parameter und kein Zustand des Pakets: Derselbe Schüler bekommt im
// nächsten Schuljahr ein um eins höheres Ablaufjahr, ohne dass die Regel angefasst wird.
func TestSchuljahrVerschiebtMit(t *testing.T) {
	for _, ende := range []int{2027, 2028, 2035} {
		jahr, ok := GueltigBisJahr("7H1", ende)
		if !ok {
			t.Fatalf("7H1 in %d nicht erkannt", ende)
		}
		if will := ende + 2; jahr != will {
			t.Errorf("Schuljahresende %d: %d, erwartet %d", ende, jahr, will)
		}
	}
}
