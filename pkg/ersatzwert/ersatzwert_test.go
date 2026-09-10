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
			got := Rechne(f.verleihjahr, f.kaufpreis, f.neupreis)
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
	if got := Rechne(3, 0, 55.55).Betrag; got != 33.33 {
		t.Errorf("60 %% von 55,55 = %.4f, want 33.33", got)
	}
	if got := Rechne(2, 0, 12.345).Betrag; got != 9.88 {
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
