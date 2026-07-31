package inventur

import "testing"

// Die Vorlagen stammen aus echten DNB-Antworten zu Titeln aus dem Bestand dieser
// Schule — nicht aus der MARC-Spezifikation. Genau diese Zeichenketten haben die
// beiden Regeln (nur Euro; DM-Sätze gar nicht) überhaupt erst begründet.
func TestPreisAusVerfuegbarkeit(t *testing.T) {
	faelle := []struct {
		name    string
		eingabe string
		erwarte float64
	}{
		{"Euro mit Punkt (ISBN 9783124912008)", "Pp. : EUR 27.00, sfr 46.90", 27.00},
		{"Euro mit Komma", "kart. : EUR 24,95", 24.95},
		{"Euro ohne Einband-Vorspann", "EUR 12.50", 12.50},

		// Der Weltatlas der Schule: Der Satz kennt nur D-Mark. Ein Preis von damals ist
		// kein Preis von heute — und ein Buch, das nur in DM verzeichnet ist, wird ohnehin
		// nicht mehr nachbestellt.
		{"nur D-Mark (ISBN 9783124801005)", "Pp. : DM 35.20", 0},

		// Der gefährlichste Fall: Der Satz nennt einen Euro-Betrag, aber es ist die
		// Umrechnung eines DM-Preises aus der Umstellungszeit. Derselbe Titel kostet laut
		// neuerem Satz 27.00 — 21.47 sähe plausibel aus und wäre falsch. Ein plausibel
		// aussehender falscher Preis ist schlimmer als gar keiner.
		{"Euro als DM-Umrechnung", "Pp. : DM 42.00, EUR 21.47", 0},

		{"leer", "", 0},
		{"ohne Betrag", "Pp.", 0},
		{"fremde Waehrung", "Pp. : sfr 46.90", 0},
		{"Null-Betrag", "EUR 0.00", 0},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			if got := preisAusVerfuegbarkeit(f.eingabe); got != f.erwarte {
				t.Errorf("preisAusVerfuegbarkeit(%q) = %v, erwartet %v", f.eingabe, got, f.erwarte)
			}
		})
	}
}

// Die DNB liefert zu einer ISBN mehrere Sätze, die neueren zuerst. Der erste brauchbare
// Preis muss gewinnen — sonst überschreibt ein alter Satz den aktuellen.
func TestVerarbeiteISBNNimmtDenErstenBrauchbarenPreis(t *testing.T) {
	var b marcBibDaten
	b.verarbeiteISBN([]marcSubfield{
		{Code: "a", Value: "9783124912008"},
		{Code: "c", Value: "Pp. : EUR 27.00, sfr 46.90"},
	})
	b.verarbeiteISBN([]marcSubfield{
		{Code: "a", Value: "3124912004"},
		{Code: "c", Value: "Pp. : DM 42.00, EUR 21.47"},
	})

	if b.preis != 27.00 {
		t.Errorf("preis = %v, erwartet 27.00 (der aeltere Satz darf den neueren nicht ueberschreiben)", b.preis)
	}
}
