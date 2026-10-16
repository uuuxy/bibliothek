package ersatzwert

import "testing"

// Die Staffel aus der Arbeitshilfe, Jahr für Jahr, und die Beispiele, die dort wörtlich
// stehen. An ihnen zeigt sich, dass die Tabelle richtig gelesen ist.
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
			got := Berechne(f.verleihjahr, f.kaufpreis, f.neupreis, 0, PreisquelleListenpreis)
			if got.Betrag != f.wantBetrag {
				t.Errorf("Betrag = %.2f, erwartet %.2f", got.Betrag, f.wantBetrag)
			}
			if got.Prozent != f.wantProzent {
				t.Errorf("Prozent = %d, erwartet %d", got.Prozent, f.wantProzent)
			}
			if got.Basis != f.wantBasis {
				t.Errorf("Basis = %q, erwartet %q", got.Basis, f.wantBasis)
			}
		})
	}
}

// Kaufmännisch auf Cent: In einem Bescheid darf kein Betrag mit vier Nachkommastellen
// stehen, und 33,33 € muss 33,33 € bleiben.
func TestBetragAufCentGerundet(t *testing.T) {
	if got := Berechne(3, 0, 55.55, 0, PreisquelleListenpreis).Betrag; got != 33.33 {
		t.Errorf("60 %% von 55,55 = %.4f, erwartet 33.33", got)
	}
	if got := Berechne(2, 0, 12.345, 0, PreisquelleListenpreis).Betrag; got != 9.88 {
		t.Errorf("80 %% von 12,345 = %.4f, erwartet 9.88", got)
	}
}

// Das Verleihjahr ist die größere Zahl aus Historie und Alter im Bestand. Sonst bliebe ein
// Buch aus dem Altbestand im ersten Verleihjahr, und die Schule verlangte den vollen Kaufpreis
// für ein zehn Jahre altes Buch.
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
			t.Errorf("Verleihjahr(%d, %d) = %d, erwartet %d", f.ausleihen, f.bestandsalter, got, f.want)
		}
	}
}

// Der Abschlag für den Zustand, etwa 20 % für einen Wasserschaden. Seine Zahl steht in einem
// Bescheid an Eltern.
//
// Er wirkt nach der Staffel, nicht neben ihr und nicht statt ihr: erst der Zeitwert des Werks,
// dann der Abzug für dieses eine Stück.
func TestZustandAbschlagMindertDenZeitwert(t *testing.T) {
	faelle := []struct {
		name                string
		verleihjahr         int
		kaufpreis, neupreis float64
		abschlag            int
		wantBetrag          float64
		wantProzent         int
	}{
		// 60 % von 41,50 € sind 24,90 €, davon 20 % ab.
		{"3. Verleihjahr, 20 % Wasserschaden", 3, 20, 41.50, 20, 19.92, 60},
		{"derselbe Fall ohne Abschlag", 3, 20, 41.50, 0, 24.90, 60},
		// Der Abschlag wirkt auch im ersten Jahr: Ein neues Buch kann beschädigt sein.
		{"1. Verleihjahr, 50 % auf den Kaufpreis", 1, 20, 25, 50, 10, 100},
		// Er ersetzt die Staffel nicht: 10 % vom Neupreis, davon 20 % ab.
		{"6. Verleihjahr, Staffel und Abschlag", 6, 20, 30, 20, 2.40, 10},
		// Ein erfasster Totalschaden ergibt 0 €: ein Zustand, keine Forderung.
		{"100 % Abschlag ergibt 0 €", 3, 20, 41.50, 100, 0, 60},
		// Ohne Preis bleibt es bei 0; der Abschlag erfindet keinen Betrag.
		{"kein Preis, mit Abschlag", 3, 0, 0, 30, 0, 60},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			got := Berechne(f.verleihjahr, f.kaufpreis, f.neupreis, f.abschlag, PreisquelleListenpreis)
			if got.Betrag != f.wantBetrag {
				t.Errorf("Betrag = %.2f, erwartet %.2f — die Zahl steht in einem Bescheid",
					got.Betrag, f.wantBetrag)
			}
			// Der Abschlag verschiebt die Staffel nicht: Prozentsatz und Basispreis sind, was
			// die Herleitung nennt („60 % von 41,50 €, abzüglich 20 %").
			if got.Prozent != f.wantProzent {
				t.Errorf("Prozent = %d, erwartet %d — der Abschlag verschiebt die Staffel nicht",
					got.Prozent, f.wantProzent)
			}
			if got.ZustandAbschlag != f.abschlag {
				t.Errorf("ZustandAbschlag = %d, want %d — ohne dieses Feld kann die "+
					"Herleitung den Abzug nicht nennen", got.ZustandAbschlag, f.abschlag)
			}
		})
	}
}

// Unmögliche Abschläge werden gekappt, nicht abgelehnt. Der teure Fall ist der negative: Er
// erhöhte den Betrag, und ein Tippfehler löste eine Forderung über mehr als den Zeitwert aus.
func TestZustandAbschlagWirdGekappt(t *testing.T) {
	ohne := Berechne(3, 20, 41.50, 0, PreisquelleListenpreis)

	negativ := Berechne(3, 20, 41.50, -20, PreisquelleListenpreis)
	if negativ.Betrag != ohne.Betrag {
		t.Errorf("Betrag bei -20 %% = %.2f, want %.2f — ein negativer Abschlag darf die "+
			"Forderung nicht erhöhen", negativ.Betrag, ohne.Betrag)
	}
	if negativ.ZustandAbschlag != 0 {
		t.Errorf("ZustandAbschlag = %d, want 0 — die Herleitung darf keinen Abzug nennen, "+
			"den es nicht gibt", negativ.ZustandAbschlag)
	}

	ueber := Berechne(3, 20, 41.50, 140, PreisquelleListenpreis)
	if ueber.Betrag != 0 {
		t.Errorf("Betrag bei 140 %% = %.2f, erwartet 0", ueber.Betrag)
	}
	if ueber.ZustandAbschlag != 100 {
		t.Errorf("ZustandAbschlag = %d, want 100 — gekappt, und die Herleitung nennt den "+
			"gekappten Wert", ueber.ZustandAbschlag)
	}
}

// Gerundet wird einmal, am Ende. Würde der Zeitwert vor dem Abschlag auf Cent gerundet, käme
// bei 24,99 € im dritten Verleihjahr mit 20 % Abschlag 11,99 € heraus statt 12,00 €. Ein Cent
// in einem Bescheid ist eine falsche Zahl.
func TestZustandAbschlagRundetNurEinmal(t *testing.T) {
	if got := Berechne(3, 0, 24.99, 20, PreisquelleListenpreis).Betrag; got != 12.00 {
		t.Errorf("60 %% von 24,99 €, abzüglich 20 %% = %.4f, want 12.00 "+
			"(11,99 hieße: zweimal gerundet)", got)
	}
}

// Die Schule wählt, welcher Preis ab dem zweiten Verleihjahr gilt.
//
// Der Nullwert ist die Regel der Arbeitshilfe: der Fall einer Schule, die die Einstellung nie
// angefasst hat. Wäre es umgekehrt, rechnete jede bestehende Anlage mit dem alten
// Einkaufspreis statt mit dem heutigen Neupreis, ohne dass jemand etwas geändert hätte.
func TestPreisquelleWaehltDieGrundlage(t *testing.T) {
	// Drittes Verleihjahr: 60 %. Kaufpreis 20 €, Listenpreis 41,50 €.
	nachVorgabe := Berechne(3, 20, 41.50, 0, PreisquelleListenpreis)
	if nachVorgabe.Betrag != 24.90 || nachVorgabe.Basis != BasisNeupreis {
		t.Errorf("Vorgabe: Betrag %.2f, Basis %q — erwartet 24.90 auf dem Listenpreis",
			nachVorgabe.Betrag, nachVorgabe.Basis)
	}

	gewaehlt := Berechne(3, 20, 41.50, 0, PreisquelleKaufpreis)
	if gewaehlt.Betrag != 12.00 {
		t.Errorf("mit gewähltem Kaufpreis: Betrag %.2f, erwartet 12.00 (60 %% von 20 €)",
			gewaehlt.Betrag)
	}
	// Eine eigene Basis, nicht die ersatzweise: Die Herleitung unterscheidet „so eingestellt"
	// von „kein Listenpreis hinterlegt"; sonst behauptete der Bescheid eine Datenlage, die es
	// nicht gibt.
	if gewaehlt.Basis != BasisKaufpreisGewaehlt {
		t.Errorf("Basis = %q, erwartet %q", gewaehlt.Basis, BasisKaufpreisGewaehlt)
	}
	if nachVorgabe.BasisPreis != 41.50 || gewaehlt.BasisPreis != 20 {
		t.Errorf("BasisPreis: Vorgabe %.2f, gewählt %.2f — erwartet 41,50 und 20,00",
			nachVorgabe.BasisPreis, gewaehlt.BasisPreis)
	}

	// Der ungesetzte Typ ist die Vorgabe: der Fall einer Anlage ohne die Einstellung und der
	// eines Rufers, der die Frage nicht kennt.
	var ungesetzt Preisquelle
	if Berechne(3, 20, 41.50, 0, ungesetzt).Betrag != 24.90 {
		t.Error("eine ungesetzte Preisquelle muss die Regel der Arbeitshilfe ergeben")
	}
}

// Im ersten Verleihjahr ändert die Einstellung nichts: Dort gilt ohnehin der Kaufpreis. Sonst
// stünde im Bescheid „so eingestellt", wo die Arbeitshilfe es vorschreibt.
func TestPreisquelleAendertDasErsteJahrNicht(t *testing.T) {
	for _, quelle := range []Preisquelle{PreisquelleListenpreis, PreisquelleKaufpreis} {
		got := Berechne(1, 20, 41.50, 0, quelle)
		if got.Betrag != 20 || got.Basis != BasisKaufpreis {
			t.Errorf("Quelle %q: Betrag %.2f, Basis %q — erwartet 20,00 auf %q",
				quelle, got.Betrag, got.Basis, BasisKaufpreis)
		}
	}
}

// Ohne Kaufpreis kann die Einstellung nicht greifen; dann zählt der Listenpreis, und die Basis
// sagt es. Eine 0 in einer Forderung wäre schlimmer als eine benannte Abweichung von der
// Einstellung.
func TestPreisquelleKaufpreisOhneKaufpreis(t *testing.T) {
	got := Berechne(3, 0, 41.50, 0, PreisquelleKaufpreis)
	if got.Betrag != 24.90 || got.Basis != BasisNeupreis {
		t.Errorf("Betrag %.2f, Basis %q — erwartet 24,90 auf dem Listenpreis",
			got.Betrag, got.Basis)
	}
}

// Die Regel für den Bestand des Schulträgers (Schülerbücherei): der Neuwert.
//
// Der teure Fehler wäre hier der Kaufpreis: Die Benutzungsordnung verlangt „Geld in Höhe des
// Neuwerts", und der Neuwert eines 2015 für 8 € gekauften Romans ist der Preis, den ein Ersatz
// heute kostet. Mit den 8 € fehlten 6 €, und das Geld gehört einem fremden Träger.
func TestNeuwertRegelNimmtDenListenpreis(t *testing.T) {
	faelle := []struct {
		name                   string
		kaufpreis, listenpreis float64
		abschlag               int
		quelle                 Preisquelle
		wantBetrag             float64
		wantBasis              Basis
	}{
		{"Listenpreis erfasst — er gilt", 8, 14, 0, PreisquelleListenpreis, 14, BasisNeupreis},
		{"kein Listenpreis — ersatzweise der Kaufpreis, benannt", 8, 0, 0, PreisquelleListenpreis, 8, BasisKaufpreisErsatzweise},
		{"Zustandsabschlag zählt auch hier", 8, 14, 25, PreisquelleListenpreis, 10.50, BasisNeupreis},
		{"100 % Abschlag ergibt 0 €", 8, 14, 100, PreisquelleListenpreis, 0, BasisNeupreis},
		{"Einstellung immer-Kaufpreis gilt auch hier", 8, 14, 0, PreisquelleKaufpreis, 8, BasisKaufpreisGewaehlt},
		{"Einstellung ohne Kaufpreis — dann der Listenpreis", 0, 14, 0, PreisquelleKaufpreis, 14, BasisNeupreis},
		{"kein Preis bekannt", 0, 0, 0, PreisquelleListenpreis, 0, BasisKaufpreisErsatzweise},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			got := BerechneNeuwert(f.kaufpreis, f.listenpreis, f.abschlag, f.quelle)
			if got.Betrag != f.wantBetrag {
				t.Errorf("Betrag = %.2f, erwartet %.2f", got.Betrag, f.wantBetrag)
			}
			if got.Basis != f.wantBasis {
				t.Errorf("Basis = %q, erwartet %q", got.Basis, f.wantBasis)
			}
			// Verleihjahr 0 ist das Kennzeichen dieser Regel: Daran unterscheidet die
			// Herleitung sie von der Staffel.
			if got.Verleihjahr != 0 {
				t.Errorf("Verleihjahr = %d, erwartet 0 — diese Rechnung kennt keines", got.Verleihjahr)
			}
			if got.Prozent != 100 {
				t.Errorf("Prozent = %d, erwartet 100 — kein Altersabschlag", got.Prozent)
			}
		})
	}
}

// Das Alter zählt hier nicht; darin liegt der Unterschied zur Staffel. Ein zehn Jahre alter
// Roman kostet in der Ersatzbeschaffung so viel wie ein neuer, und die Funktion hat deshalb
// kein Verleihjahr als Parameter. Dieselben Zahlen ergeben nach der Staffel 10 %, nach dem
// Neuwert 100 %.
func TestNeuwertRegelKenntKeinenAltersabschlag(t *testing.T) {
	nachStaffel := Berechne(9, 8, 14, 0, PreisquelleListenpreis)
	if nachStaffel.Betrag != 1.40 {
		t.Fatalf("Staffel im 9. Verleihjahr: %.2f, erwartet 1.40 (10 %% von 14 €)", nachStaffel.Betrag)
	}
	nachNeuwert := BerechneNeuwert(8, 14, 0, PreisquelleListenpreis)
	if nachNeuwert.Betrag != 14.00 {
		t.Errorf("Neuwert-Regel: %.2f, erwartet 14.00 — das Alter darf hier nichts abziehen; "+
			"1,40 € hieße, die Staffel des Landes gilt auch für den Bestand des Schulträgers",
			nachNeuwert.Betrag)
	}
}
