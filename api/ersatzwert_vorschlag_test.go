package api

import (
	"strings"
	"testing"

	"bibliothek/repository"
)

// Gate für die ZWEI Regeln des Betragsvorschlags (OFFEN.md 9.3 a).
//
// Der teure Fehler wäre, die Staffel der Arbeitshilfe auf den Bestand der Schülerbücherei
// anzuwenden: Sie gehört dem Schulträger, und dessen Benutzungsordnung verlangt „zuerst
// Ersatzbeschaffung, sonst Geld in Höhe des NEUWERTS" (mittel_konzept.md 1.2) — ohne
// Abschlag für das Alter. Ein zehn Jahre altes Büchereibuch mit 10 % anzusetzen, weil die
// Staffel das für Schulbücher sagt, verschenkt das Geld eines fremden Trägers.
//
// Deshalb steht hier für JEDE Regel ein Fall mit einem ALTEN Buch: Bei Lernmitteln muss
// der Abschlag greifen, bei Büchereibüchern darf er es nicht.
func TestErsatzwertVorschlagWaehltDieRichtigeRegel(t *testing.T) {
	faelle := []struct {
		name          string
		groessen      repository.ErsatzwertGroessen
		wantBetrag    float64
		wantImSatz    string
		wantNichtSatz string
	}{
		{
			name: "Lernmittel im 1. Verleihjahr — voller Kaufpreis",
			groessen: repository.ErsatzwertGroessen{
				Kaufpreis: 20.00, IstLernmittel: true,
				SchuljahreMitAusleihe: 1, SchuljahreImBestand: 0,
			},
			wantBetrag: 20.00,
			wantImSatz: "100 %",
		},
		{
			// Das Beispiel „Schüler C" aus der Arbeitshilfe: drittes Verleihjahr, 60 %.
			name: "Lernmittel im 3. Verleihjahr — 60 %",
			groessen: repository.ErsatzwertGroessen{
				Kaufpreis: 41.50, IstLernmittel: true,
				SchuljahreMitAusleihe: 3, SchuljahreImBestand: 2,
			},
			wantBetrag: 24.90,
			wantImSatz: "60 %",
		},
		{
			name: "Lernmittel im 9. Verleihjahr — ab dem 6. sind es 10 %",
			groessen: repository.ErsatzwertGroessen{
				Kaufpreis: 30.00, IstLernmittel: true,
				SchuljahreMitAusleihe: 2, SchuljahreImBestand: 8,
			},
			wantBetrag: 3.00,
			wantImSatz: "10 %",
		},
		{
			name: "Lernmittel ohne erfassten Preis — kein geratener Betrag",
			groessen: repository.ErsatzwertGroessen{
				Kaufpreis: 0, IstLernmittel: true,
				SchuljahreMitAusleihe: 3, SchuljahreImBestand: 2,
			},
			wantBetrag: 0,
			wantImSatz: "kein Preis hinterlegt",
		},
		{
			name: "Büchereibuch, neu — Neuwert",
			groessen: repository.ErsatzwertGroessen{
				Kaufpreis: 12.00, IstLernmittel: false,
				SchuljahreMitAusleihe: 1, SchuljahreImBestand: 0,
			},
			wantBetrag:    12.00,
			wantImSatz:    "ohne Abschlag",
			wantNichtSatz: "Verleihjahr",
		},
		{
			// DER Fall, für den dieses Gate da ist: Nach der Staffel wären es 10 % =
			// 1,20 €. Die Benutzungsordnung kennt diesen Abschlag nicht.
			name: "Büchereibuch, zehn Jahre alt — TROTZDEM voller Neuwert",
			groessen: repository.ErsatzwertGroessen{
				Kaufpreis: 12.00, IstLernmittel: false,
				SchuljahreMitAusleihe: 7, SchuljahreImBestand: 10,
			},
			wantBetrag:    12.00,
			wantImSatz:    "ohne Abschlag",
			wantNichtSatz: "%",
		},
		{
			name: "Büchereibuch ohne erfassten Preis",
			groessen: repository.ErsatzwertGroessen{
				Kaufpreis: 0, IstLernmittel: false,
				SchuljahreMitAusleihe: 2, SchuljahreImBestand: 3,
			},
			wantBetrag: 0,
			wantImSatz: "kein Preis hinterlegt",
		},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			got := ersatzwertVorschlagAus(f.groessen)

			if got.Betrag != f.wantBetrag {
				t.Errorf("Betrag = %.2f, want %.2f — die Zahl landet in einer Forderung "+
					"(Herleitung: %q)", got.Betrag, f.wantBetrag, got.Herleitung)
			}
			if !strings.Contains(got.Herleitung, f.wantImSatz) {
				t.Errorf("Herleitung = %q, erwartet darin %q", got.Herleitung, f.wantImSatz)
			}
			if f.wantNichtSatz != "" && strings.Contains(got.Herleitung, f.wantNichtSatz) {
				t.Errorf("Herleitung = %q — darf %q NICHT enthalten: Für den Bestand des "+
					"Schulträgers gilt die Staffel der Arbeitshilfe nicht",
					got.Herleitung, f.wantNichtSatz)
			}
			if got.IstLernmittel != f.groessen.IstLernmittel {
				t.Errorf("IstLernmittel = %v, want %v — der Dialog benennt die Regel danach",
					got.IstLernmittel, f.groessen.IstLernmittel)
			}
		})
	}
}
