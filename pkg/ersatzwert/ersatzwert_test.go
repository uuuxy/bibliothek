package ersatzwert

import "testing"

// Die Staffel aus der Arbeitshilfe, Jahr für Jahr — und die beiden Beispiele, die dort
// wörtlich stehen. Sie sind der Beweis, dass die Tabelle richtig gelesen ist.
func TestStaffelNachArbeitshilfe(t *testing.T) {
	faelle := []struct {
		name                string
		verleihjahr         int
		kaufpreis, neupreis float64
		wantBetrag          float64
		wantProzent         int
		wantBasis           Basis
	}{
		// „An Schüler A wurde das neue Lehrwerk X zum ersten Mal verliehen. Es wurde für
		// den Preis in Höhe von 20,-- € beschafft. … Als Schadenersatz wird der volle
		// Lehrwerkspreis zum Zeitpunkt des Kaufs berechnet, also 20,-- €."
		{"Beispiel A: 1. Verleih, voller Kaufpreis", 1, 20, 25, 20, 100, BasisKaufpreis},
		// „An Schüler B wurde ein Lehrwerk verliehen, das … zuvor erst einmal an einen
		// anderen Schüler verliehen wurde. … 80 % des Neupreises (zweites Verleihjahr)."
		{"Beispiel B: 2. Verleihjahr, 80 % vom Neupreis", 2, 20, 25, 20, 80, BasisNeupreis},
		// „Das Lehrwerk wird nach drei Schuljahren … zurückgegeben … 60 % des Neupreises
		// (drittes Verleihjahr)."
		{"Beispiel C: 3. Verleihjahr, 60 % vom Neupreis", 3, 20, 30, 18, 60, BasisNeupreis},
		{"4. Verleihjahr", 4, 20, 30, 12, 40, BasisNeupreis},
		{"5. Verleihjahr", 5, 20, 30, 6, 20, BasisNeupreis},
		{"6. Verleihjahr — ab hier 10 %", 6, 20, 30, 3, 10, BasisNeupreis},
		{"12. Verleihjahr — bleibt bei 10 %", 12, 20, 30, 3, 10, BasisNeupreis},
		// Ohne Neupreis: ab dem 2. Jahr ersatzweise der Kaufpreis, und das wird benannt.
		{"kein Neupreis hinterlegt", 3, 41.5, 0, 24.9, 60, BasisKaufpreisErsatzweise},
		// Kein Preis überhaupt: 0 € und der Mensch trägt ein.
		{"kein Preis bekannt", 2, 0, 0, 0, 80, BasisKaufpreisErsatzweise},
		// Eine 0 aus unvollständiger Historie darf nicht 0 € ergeben.
		{"Verleihjahr 0 gilt als erstes Jahr", 0, 20, 30, 20, 100, BasisKaufpreis},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			got := Rechne(f.verleihjahr, f.kaufpreis, f.neupreis, 0, PreisquelleListenpreis)
			if got.Betrag != f.wantBetrag {
				t.Errorf("Betrag = %.2f, want %.2f", got.Betrag, f.wantBetrag)
			}
			if got.Prozent != f.wantProzent {
				t.Errorf("Prozent = %d, want %d", got.Prozent, f.wantProzent)
			}
			if got.Basis != f.wantBasis {
				t.Errorf("Basis = %q, want %q", got.Basis, f.wantBasis)
			}
		})
	}
}

// Kaufmännisch auf Cent: In einem Bescheid darf kein Betrag mit vier Nachkommastellen
// stehen, und 33,33 € muss 33,33 € bleiben.
func TestBetragAufCentGerundet(t *testing.T) {
	if got := Rechne(3, 0, 55.55, 0, PreisquelleListenpreis).Betrag; got != 33.33 {
		t.Errorf("60 %% von 55,55 = %.4f, want 33.33", got)
	}
	if got := Rechne(2, 0, 12.345, 0, PreisquelleListenpreis).Betrag; got != 9.88 {
		t.Errorf("80 %% von 12,345 = %.4f, want 9.88", got)
	}
}

// Das Verleihjahr nimmt das Maximum aus Historie und Bestandsalter — sonst wäre ein
// Buch aus dem Altbestand für immer im ersten Verleihjahr und die Schule verlangte den
// vollen Kaufpreis für ein zehn Jahre altes Buch.
func TestVerleihjahrNimmtDasMaximum(t *testing.T) {
	faelle := []struct{ ausleihen, bestandsalter, want int }{
		{1, 0, 1},  // neues Buch, einmal verliehen
		{1, 6, 7},  // Altbestand ohne Historie: sieben Jahre im Haus
		{4, 1, 4},  // junges Buch, viermal verliehen
		{0, 0, 1},  // nichts bekannt → erstes Jahr
		{0, -1, 1}, // unplausibel → erstes Jahr
	}
	for _, f := range faelle {
		if got := Verleihjahr(f.ausleihen, f.bestandsalter); got != f.want {
			t.Errorf("Verleihjahr(%d, %d) = %d, want %d", f.ausleihen, f.bestandsalter, got, f.want)
		}
	}
}

// Der Zustands-Abschlag (Migration 127, Anforderungsliste Nr. 2: „20 % durch
// Wasserschaden"). Bis hierher rief JEDER Test oben ihn mit 0 — der Zweig war ungeprüft,
// obwohl seine Zahl in einem Bescheid an Eltern landet.
//
// Die Regel, die hier festgenagelt wird: Der Abschlag wirkt NACH der Staffel, nicht
// neben ihr und nicht statt ihr. Erst der Zeitwert des Werks, dann der Abzug für dieses
// eine Stück.
func TestZustandAbschlagMindertDenZeitwert(t *testing.T) {
	faelle := []struct {
		name                string
		verleihjahr         int
		kaufpreis, neupreis float64
		abschlag            int
		wantBetrag          float64
		wantProzent         int
	}{
		// Der Fall aus dem Bauplan: 60 % von 41,50 € = 24,90 €, davon 20 % ab.
		{"3. Verleihjahr, 20 % Wasserschaden", 3, 20, 41.50, 20, 19.92, 60},
		{"derselbe Fall ohne Abschlag", 3, 20, 41.50, 0, 24.90, 60},
		// Der Abschlag wirkt auch im ersten Jahr — ein neues Buch kann beschädigt sein.
		{"1. Verleihjahr, 50 % auf den Kaufpreis", 1, 20, 25, 50, 10, 100},
		// … und er ersetzt die Staffel nicht: 10 % vom Neupreis, davon 20 % ab.
		{"6. Verleihjahr, Staffel UND Abschlag", 6, 20, 30, 20, 2.40, 10},
		// Totalschaden erfasst: 0 €. Das ist ein Zustand, keine Forderung.
		{"100 % Abschlag ergibt 0 €", 3, 20, 41.50, 100, 0, 60},
		// Ohne Preis bleibt es bei 0 — der Abschlag erfindet keinen Betrag.
		{"kein Preis, mit Abschlag", 3, 0, 0, 30, 0, 60},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			got := Rechne(f.verleihjahr, f.kaufpreis, f.neupreis, f.abschlag, PreisquelleListenpreis)
			if got.Betrag != f.wantBetrag {
				t.Errorf("Betrag = %.2f, want %.2f — die Zahl steht in einem Bescheid",
					got.Betrag, f.wantBetrag)
			}
			// Der Abschlag darf die Staffel nicht verschieben: Prozentsatz und Basispreis
			// sind das, was die Herleitung nennt („60 % von 41,50 €, abzüglich 20 %").
			if got.Prozent != f.wantProzent {
				t.Errorf("Prozent = %d, want %d — der Abschlag verschiebt die Staffel nicht",
					got.Prozent, f.wantProzent)
			}
			if got.ZustandAbschlag != f.abschlag {
				t.Errorf("ZustandAbschlag = %d, want %d — ohne dieses Feld kann die "+
					"Herleitung den Abzug nicht nennen", got.ZustandAbschlag, f.abschlag)
			}
		})
	}
}

// Unmögliche Abschläge werden gekappt, nicht abgelehnt. Der teure Fall ist der negative:
// Er würde den Betrag ERHÖHEN — eine Forderung über mehr als den Zeitwert, ausgelöst von
// einem Tippfehler.
func TestZustandAbschlagWirdGekappt(t *testing.T) {
	ohne := Rechne(3, 20, 41.50, 0, PreisquelleListenpreis)

	negativ := Rechne(3, 20, 41.50, -20, PreisquelleListenpreis)
	if negativ.Betrag != ohne.Betrag {
		t.Errorf("Betrag bei -20 %% = %.2f, want %.2f — ein negativer Abschlag darf die "+
			"Forderung nicht erhöhen", negativ.Betrag, ohne.Betrag)
	}
	if negativ.ZustandAbschlag != 0 {
		t.Errorf("ZustandAbschlag = %d, want 0 — die Herleitung darf keinen Abzug nennen, "+
			"den es nicht gibt", negativ.ZustandAbschlag)
	}

	ueber := Rechne(3, 20, 41.50, 140, PreisquelleListenpreis)
	if ueber.Betrag != 0 {
		t.Errorf("Betrag bei 140 %% = %.2f, want 0", ueber.Betrag)
	}
	if ueber.ZustandAbschlag != 100 {
		t.Errorf("ZustandAbschlag = %d, want 100 — gekappt, und die Herleitung nennt den "+
			"gekappten Wert", ueber.ZustandAbschlag)
	}
}

// Gerundet wird EINMAL, am Ende. Würde der Zeitwert vor dem Abschlag auf Cent gerundet,
// käme bei 24,99 € im dritten Verleihjahr mit 20 % Abschlag 11,99 € heraus statt 12,00 €.
// Ein Cent in einem Bescheid ist kein Rundungsfehler, sondern eine falsche Zahl.
func TestZustandAbschlagRundetNurEinmal(t *testing.T) {
	if got := Rechne(3, 0, 24.99, 20, PreisquelleListenpreis).Betrag; got != 12.00 {
		t.Errorf("60 %% von 24,99 €, abzüglich 20 %% = %.4f, want 12.00 "+
			"(11,99 hieße: zweimal gerundet)", got)
	}
}

// Die wählbare Berechnungsgrundlage (Anforderungsliste Nr. 3, OFFEN.md 9.8 Stufe 4).
//
// Der Nullwert MUSS die Regel der Arbeitshilfe sein — der Fall einer Schule, die die
// Einstellung nie angefasst hat. Wäre es umgekehrt, rechnete jede bestehende Anlage nach
// dem ersten Update mit dem alten Einkaufspreis statt mit dem heutigen Neupreis, ohne
// dass jemand etwas geändert hätte.
func TestPreisquelleWaehltDieGrundlage(t *testing.T) {
	// Drittes Verleihjahr: 60 %. Kaufpreis 20 €, Listenpreis 41,50 €.
	nachVorgabe := Rechne(3, 20, 41.50, 0, PreisquelleListenpreis)
	if nachVorgabe.Betrag != 24.90 || nachVorgabe.Basis != BasisNeupreis {
		t.Errorf("Vorgabe: Betrag %.2f, Basis %q — erwartet 24.90 auf dem Listenpreis",
			nachVorgabe.Betrag, nachVorgabe.Basis)
	}

	gewaehlt := Rechne(3, 20, 41.50, 0, PreisquelleKaufpreis)
	if gewaehlt.Betrag != 12.00 {
		t.Errorf("mit gewähltem Kaufpreis: Betrag %.2f, erwartet 12.00 (60 %% von 20 €)",
			gewaehlt.Betrag)
	}
	// Eine EIGENE Basis, nicht die ersatzweise: Die Herleitung muss „so eingestellt" von
	// „kein Listenpreis hinterlegt" unterscheiden können — sonst behauptet der Bescheid
	// eine Datenlage, die es nicht gibt.
	if gewaehlt.Basis != BasisKaufpreisGewaehlt {
		t.Errorf("Basis = %q, erwartet %q", gewaehlt.Basis, BasisKaufpreisGewaehlt)
	}
	if nachVorgabe.BasisPreis != 41.50 || gewaehlt.BasisPreis != 20 {
		t.Errorf("BasisPreis: Vorgabe %.2f, gewählt %.2f — erwartet 41,50 und 20,00",
			nachVorgabe.BasisPreis, gewaehlt.BasisPreis)
	}

	// Der ungesetzte Typ ist die Vorgabe: Das ist der Fall einer Anlage ohne die
	// Einstellung und der eines Aufrufers, der die Frage nicht kennt.
	var ungesetzt Preisquelle
	if Rechne(3, 20, 41.50, 0, ungesetzt).Betrag != 24.90 {
		t.Error("eine ungesetzte Preisquelle muss die Regel der Arbeitshilfe ergeben")
	}
}

// Im ERSTEN Verleihjahr ändert die Einstellung nichts: Dort gilt ohnehin der Kaufpreis.
// Sonst stünde im Bescheid „so eingestellt", wo die Arbeitshilfe es ohnehin vorschreibt.
func TestPreisquelleAendertDasErsteJahrNicht(t *testing.T) {
	for _, quelle := range []Preisquelle{PreisquelleListenpreis, PreisquelleKaufpreis} {
		got := Rechne(1, 20, 41.50, 0, quelle)
		if got.Betrag != 20 || got.Basis != BasisKaufpreis {
			t.Errorf("Quelle %q: Betrag %.2f, Basis %q — erwartet 20,00 auf %q",
				quelle, got.Betrag, got.Basis, BasisKaufpreis)
		}
	}
}

// Ohne Kaufpreis kann die Einstellung nicht greifen — dann zählt der Listenpreis, und
// die Basis sagt es. Eine 0 in einer Forderung wäre schlimmer als eine benannte
// Abweichung von der Einstellung.
func TestPreisquelleKaufpreisOhneKaufpreis(t *testing.T) {
	got := Rechne(3, 0, 41.50, 0, PreisquelleKaufpreis)
	if got.Betrag != 24.90 || got.Basis != BasisNeupreis {
		t.Errorf("Betrag %.2f, Basis %q — erwartet 24,90 auf dem Listenpreis",
			got.Betrag, got.Basis)
	}
}
