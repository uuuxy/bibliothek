package ersatzwert

import (
	"testing"

	"bibliothek/pkg/betrag"
)

// Die Staffel aus der Arbeitshilfe, Jahr für Jahr, und die Beispiele, die dort wörtlich
// stehen. An ihnen zeigt sich, dass die Tabelle richtig gelesen ist.
func TestStaffelNachArbeitshilfe(t *testing.T) {
	faelle := []struct {
		name                string
		verleihjahr         int
		kaufpreis, neupreis betrag.Cent
		wantBetrag          betrag.Cent
		wantProzent         int
		wantBasis           Basis
	}{
		// „An Schüler A wurde das neue Lehrwerk X zum ersten Mal verliehen. Es wurde für
		// den Preis in Höhe von 20,-- € beschafft. … Als Schadenersatz wird der volle
		// Lehrwerkspreis zum Zeitpunkt des Kaufs berechnet, also 20,-- €."
		{"Beispiel A: 1. Verleih, voller Kaufpreis", 1, 2000, 2500, 2000, 100, BasisKaufpreis},
		// „An Schüler B wurde ein Lehrwerk verliehen, das … zuvor erst einmal an einen
		// anderen Schüler verliehen wurde. … 80 % des Neupreises (zweites Verleihjahr)."
		{"Beispiel B: 2. Verleihjahr, 80 % vom Neupreis", 2, 2000, 2500, 2000, 80, BasisNeupreis},
		// „Das Lehrwerk wird nach drei Schuljahren … zurückgegeben … 60 % des Neupreises
		// (drittes Verleihjahr)."
		{"Beispiel C: 3. Verleihjahr, 60 % vom Neupreis", 3, 2000, 3000, 1800, 60, BasisNeupreis},
		{"4. Verleihjahr", 4, 2000, 3000, 1200, 40, BasisNeupreis},
		{"5. Verleihjahr", 5, 2000, 3000, 600, 20, BasisNeupreis},
		{"6. Verleihjahr — ab hier 10 %", 6, 2000, 3000, 300, 10, BasisNeupreis},
		{"12. Verleihjahr — bleibt bei 10 %", 12, 2000, 3000, 300, 10, BasisNeupreis},
		// Ohne Neupreis: ab dem 2. Jahr ersatzweise der Kaufpreis, und das wird benannt.
		{"kein Neupreis hinterlegt", 3, 4150, 0, 2490, 60, BasisKaufpreisErsatzweise},
		// Kein Preis überhaupt: 0 € und der Mensch trägt ein.
		{"kein Preis bekannt", 2, 0, 0, 0, 80, BasisKaufpreisErsatzweise},
		// Eine 0 aus unvollständiger Historie darf nicht 0 € ergeben.
		{"Verleihjahr 0 gilt als erstes Jahr", 0, 2000, 3000, 2000, 100, BasisKaufpreis},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			got := Berechne(f.verleihjahr, f.kaufpreis, f.neupreis, 0, PreisquelleListenpreis)
			if got.Betrag != f.wantBetrag {
				t.Errorf("Betrag = %s, erwartet %s", got.Betrag.Text(), f.wantBetrag.Text())
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

// Kaufmännisch auf Cent: 60 % von 55,55 € sind 33,33 €, und 33,33 € muss 33,33 € bleiben.
func TestBetragAufCentGerundet(t *testing.T) {
	if got := Berechne(3, 0, 5555, 0, PreisquelleListenpreis).Betrag; got != 3333 {
		t.Errorf("60 %% von 55,55 = %s, erwartet 33,33", got.Text())
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
		kaufpreis, neupreis betrag.Cent
		abschlag            int
		wantBetrag          betrag.Cent
		wantProzent         int
	}{
		// 60 % von 41,50 € sind 24,90 €, davon 20 % ab.
		{"3. Verleihjahr, 20 % Wasserschaden", 3, 2000, 4150, 20, 1992, 60},
		{"derselbe Fall ohne Abschlag", 3, 2000, 4150, 0, 2490, 60},
		// Der Abschlag wirkt auch im ersten Jahr: Ein neues Buch kann beschädigt sein.
		{"1. Verleihjahr, 50 % auf den Kaufpreis", 1, 2000, 2500, 50, 1000, 100},
		// Er ersetzt die Staffel nicht: 10 % vom Neupreis, davon 20 % ab.
		{"6. Verleihjahr, Staffel und Abschlag", 6, 2000, 3000, 20, 240, 10},
		// Ein erfasster Totalschaden ergibt 0 €: ein Zustand, keine Forderung.
		{"100 % Abschlag ergibt 0 €", 3, 2000, 4150, 100, 0, 60},
		// Ohne Preis bleibt es bei 0; der Abschlag erfindet keinen Betrag.
		{"kein Preis, mit Abschlag", 3, 0, 0, 30, 0, 60},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			got := Berechne(f.verleihjahr, f.kaufpreis, f.neupreis, f.abschlag, PreisquelleListenpreis)
			if got.Betrag != f.wantBetrag {
				t.Errorf("Betrag = %s, erwartet %s — die Zahl steht in einem Bescheid",
					got.Betrag.Text(), f.wantBetrag.Text())
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
	ohne := Berechne(3, 2000, 4150, 0, PreisquelleListenpreis)

	negativ := Berechne(3, 2000, 4150, -20, PreisquelleListenpreis)
	if negativ.Betrag != ohne.Betrag {
		t.Errorf("Betrag bei -20 %% = %s, want %s — ein negativer Abschlag darf die "+
			"Forderung nicht erhöhen", negativ.Betrag.Text(), ohne.Betrag.Text())
	}
	if negativ.ZustandAbschlag != 0 {
		t.Errorf("ZustandAbschlag = %d, want 0 — die Herleitung darf keinen Abzug nennen, "+
			"den es nicht gibt", negativ.ZustandAbschlag)
	}

	ueber := Berechne(3, 2000, 4150, 140, PreisquelleListenpreis)
	if ueber.Betrag != 0 {
		t.Errorf("Betrag bei 140 %% = %s, erwartet 0", ueber.Betrag.Text())
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
	if got := Berechne(3, 0, 2499, 20, PreisquelleListenpreis).Betrag; got != 1200 {
		t.Errorf("60 %% von 24,99 €, abzüglich 20 %% = %s, want 12,00 "+
			"(11,99 hieße: zweimal gerundet)", got.Text())
	}
}

// Kaufmännisch gerundet wird der Betrag, der auf dem Papier steht, nicht eine Kommazahl des
// Rechners. In ihr liegt 10 % von 10,05 € knapp unter 1,005, und der Bescheid nennte 1,00 €
// statt 1,01 €. Die Probe ist dieselbe Rechnung in ganzen Cent.
func TestBetragRundetDenDezimalwert(t *testing.T) {
	if got := Berechne(6, 0, 1005, 0, PreisquelleListenpreis).Betrag; got != 101 {
		t.Errorf("10 %% von 10,05 € = %s, erwartet 1,01", got.Text())
	}
	if got := BerechneNeuwert(0, 35, 10, PreisquelleListenpreis).Betrag; got != 32 {
		t.Errorf("0,35 € abzüglich 10 %% = %s, erwartet 0,32", got.Text())
	}
	for cent := 1; cent <= 5000; cent++ {
		preis := betrag.Cent(cent)
		for _, abschlag := range []int{0, 5, 10, 20, 25, 42, 50} {
			for jahr := 1; jahr <= 6; jahr++ {
				got := Berechne(jahr, 0, preis, abschlag, PreisquelleListenpreis)
				want := betrag.Cent((cent*got.Prozent*(100-abschlag) + 5000) / 10000)
				if got.Betrag != want {
					t.Fatalf("%d. Verleihjahr, %s, Abschlag %d %%: %s, erwartet %s",
						jahr, preis.Euro(), abschlag, got.Betrag.Text(), want.Text())
				}
			}
			got := BerechneNeuwert(0, preis, abschlag, PreisquelleListenpreis)
			if want := betrag.Cent((cent*(100-abschlag) + 50) / 100); got.Betrag != want {
				t.Fatalf("Neuwert %s, Abschlag %d %%: %s, erwartet %s",
					preis.Euro(), abschlag, got.Betrag.Text(), want.Text())
			}
		}
	}
}

// Die Schule wählt, welcher Preis ab dem zweiten Verleihjahr gilt.
//
// Der Nullwert ist die Regel der Arbeitshilfe: der Fall einer Schule, die die Einstellung nie
// angefasst hat. Wäre es umgekehrt, rechnete jede bestehende Anlage mit dem alten
// Einkaufspreis statt mit dem heutigen Neupreis, ohne dass jemand etwas geändert hätte.
func TestPreisquelleWaehltDieGrundlage(t *testing.T) {
	// Drittes Verleihjahr: 60 %. Kaufpreis 20 €, Listenpreis 41,50 €.
	nachVorgabe := Berechne(3, 2000, 4150, 0, PreisquelleListenpreis)
	if nachVorgabe.Betrag != 2490 || nachVorgabe.Basis != BasisNeupreis {
		t.Errorf("Vorgabe: Betrag %s, Basis %q — erwartet 24,90 auf dem Listenpreis",
			nachVorgabe.Betrag.Text(), nachVorgabe.Basis)
	}

	gewaehlt := Berechne(3, 2000, 4150, 0, PreisquelleKaufpreis)
	if gewaehlt.Betrag != 1200 {
		t.Errorf("mit gewähltem Kaufpreis: Betrag %s, erwartet 12,00 (60 %% von 20 €)",
			gewaehlt.Betrag.Text())
	}
	// Eine eigene Basis, nicht die ersatzweise: Die Herleitung unterscheidet „so eingestellt"
	// von „kein Listenpreis hinterlegt"; sonst behauptete der Bescheid eine Datenlage, die es
	// nicht gibt.
	if gewaehlt.Basis != BasisKaufpreisGewaehlt {
		t.Errorf("Basis = %q, erwartet %q", gewaehlt.Basis, BasisKaufpreisGewaehlt)
	}
	if nachVorgabe.BasisPreis != 4150 || gewaehlt.BasisPreis != 2000 {
		t.Errorf("BasisPreis: Vorgabe %s, gewählt %s — erwartet 41,50 und 20,00",
			nachVorgabe.BasisPreis.Text(), gewaehlt.BasisPreis.Text())
	}

	// Der ungesetzte Typ ist die Vorgabe: der Fall einer Anlage ohne die Einstellung und der
	// eines Rufers, der die Frage nicht kennt.
	var ungesetzt Preisquelle
	if Berechne(3, 2000, 4150, 0, ungesetzt).Betrag != 2490 {
		t.Error("eine ungesetzte Preisquelle muss die Regel der Arbeitshilfe ergeben")
	}
}

// Im ersten Verleihjahr ändert die Einstellung nichts: Dort gilt ohnehin der Kaufpreis. Sonst
// stünde im Bescheid „so eingestellt", wo die Arbeitshilfe es vorschreibt.
func TestPreisquelleAendertDasErsteJahrNicht(t *testing.T) {
	for _, quelle := range []Preisquelle{PreisquelleListenpreis, PreisquelleKaufpreis} {
		got := Berechne(1, 2000, 4150, 0, quelle)
		if got.Betrag != 2000 || got.Basis != BasisKaufpreis {
			t.Errorf("Quelle %q: Betrag %s, Basis %q — erwartet 20,00 auf %q",
				quelle, got.Betrag.Text(), got.Basis, BasisKaufpreis)
		}
	}
}

// Ohne Kaufpreis kann die Einstellung nicht greifen; dann zählt der Listenpreis, und die Basis
// sagt es. Eine 0 in einer Forderung wäre schlimmer als eine benannte Abweichung von der
// Einstellung.
func TestPreisquelleKaufpreisOhneKaufpreis(t *testing.T) {
	got := Berechne(3, 0, 4150, 0, PreisquelleKaufpreis)
	if got.Betrag != 2490 || got.Basis != BasisNeupreis {
		t.Errorf("Betrag %s, Basis %q — erwartet 24,90 auf dem Listenpreis",
			got.Betrag.Text(), got.Basis)
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
		kaufpreis, listenpreis betrag.Cent
		abschlag               int
		quelle                 Preisquelle
		wantBetrag             betrag.Cent
		wantBasis              Basis
	}{
		{"Listenpreis erfasst — er gilt", 800, 1400, 0, PreisquelleListenpreis, 1400, BasisNeupreis},
		{"kein Listenpreis — ersatzweise der Kaufpreis, benannt", 800, 0, 0, PreisquelleListenpreis, 800, BasisKaufpreisErsatzweise},
		{"Zustandsabschlag zählt auch hier", 800, 1400, 25, PreisquelleListenpreis, 1050, BasisNeupreis},
		{"100 % Abschlag ergibt 0 €", 800, 1400, 100, PreisquelleListenpreis, 0, BasisNeupreis},
		{"Einstellung immer-Kaufpreis gilt auch hier", 800, 1400, 0, PreisquelleKaufpreis, 800, BasisKaufpreisGewaehlt},
		{"Einstellung ohne Kaufpreis — dann der Listenpreis", 0, 1400, 0, PreisquelleKaufpreis, 1400, BasisNeupreis},
		{"kein Preis bekannt", 0, 0, 0, PreisquelleListenpreis, 0, BasisKaufpreisErsatzweise},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			got := BerechneNeuwert(f.kaufpreis, f.listenpreis, f.abschlag, f.quelle)
			if got.Betrag != f.wantBetrag {
				t.Errorf("Betrag = %s, erwartet %s", got.Betrag.Text(), f.wantBetrag.Text())
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
	nachStaffel := Berechne(9, 800, 1400, 0, PreisquelleListenpreis)
	if nachStaffel.Betrag != 140 {
		t.Fatalf("Staffel im 9. Verleihjahr: %s, erwartet 1,40 (10 %% von 14 €)", nachStaffel.Betrag.Text())
	}
	nachNeuwert := BerechneNeuwert(800, 1400, 0, PreisquelleListenpreis)
	if nachNeuwert.Betrag != 1400 {
		t.Errorf("Neuwert-Regel: %s, erwartet 14,00 — das Alter darf hier nichts abziehen; "+
			"1,40 € hieße, die Staffel des Landes gilt auch für den Bestand des Schulträgers",
			nachNeuwert.Betrag.Text())
	}
}
