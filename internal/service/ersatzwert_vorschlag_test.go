package service

import (
	"strings"
	"testing"

	"bibliothek/pkg/betrag"
	"bibliothek/pkg/ersatzwert"
	"bibliothek/pkg/mitteltopf"
	"bibliothek/repository"
)

// Prüfung für die zwei Regeln des Betragsvorschlags.
//
// Der teure Fehler wäre, die Staffel der Arbeitshilfe auf den Bestand der Schülerbücherei
// anzuwenden: Sie gehört dem Schulträger, und dessen Benutzungsordnung verlangt „zuerst
// Ersatzbeschaffung, sonst Geld in Höhe des NEUWERTS" (mittel_konzept.md 1.2) — ohne
// Abschlag für das Alter. Ein zehn Jahre altes Büchereibuch mit 10 % anzusetzen, weil die
// Staffel das für Schulbücher sagt, verschenkt das Geld eines fremden Trägers.
//
// Deshalb steht hier für jede Regel ein Fall mit einem alten Buch: Bei Lernmitteln muss
// der Abschlag greifen, bei Büchereibüchern darf er es nicht.
func TestErsatzwertVorschlagWaehltDieRichtigeRegel(t *testing.T) {
	faelle := []struct {
		name          string
		groessen      repository.ErsatzwertGroessen
		wantBetrag    betrag.Cent
		wantImSatz    string
		wantNichtSatz string
		// wantOhnePreis: Dem Betrag liegt kein Preis zugrunde. Die Vorgabe ist bewusst
		// „Preis vorhanden" — ein neuer Fall, der ihn vergisst, wird laut statt still.
		wantOhnePreis bool
	}{
		{
			name: "Lernmittel im 1. Verleihjahr — voller Kaufpreis",
			groessen: repository.ErsatzwertGroessen{
				Kaufpreis: 2000, Topf: mitteltopf.Land,
				SchuljahreMitAusleihe: 1, SchuljahreImBestand: 0,
			},
			wantBetrag: 2000,
			wantImSatz: "100 %",
		},
		{
			// Das Beispiel „Schüler C" aus der Arbeitshilfe: drittes Verleihjahr, 60 %.
			name: "Lernmittel im 3. Verleihjahr — 60 %",
			groessen: repository.ErsatzwertGroessen{
				Kaufpreis: 4150, Topf: mitteltopf.Land,
				SchuljahreMitAusleihe: 3, SchuljahreImBestand: 2,
			},
			wantBetrag: 2490,
			wantImSatz: "60 %",
		},
		{
			name: "Lernmittel im 9. Verleihjahr — ab dem 6. sind es 10 %",
			groessen: repository.ErsatzwertGroessen{
				Kaufpreis: 3000, Topf: mitteltopf.Land,
				SchuljahreMitAusleihe: 2, SchuljahreImBestand: 8,
			},
			wantBetrag: 300,
			wantImSatz: "10 %",
		},
		{
			name: "Lernmittel ohne erfassten Preis — kein geratener Betrag",
			groessen: repository.ErsatzwertGroessen{
				Kaufpreis: 0, Topf: mitteltopf.Land,
				SchuljahreMitAusleihe: 3, SchuljahreImBestand: 2,
			},
			wantBetrag:    0,
			wantImSatz:    "kein Preis hinterlegt",
			wantOhnePreis: true,
		},
		{
			name: "Büchereibuch, neu — Neuwert",
			groessen: repository.ErsatzwertGroessen{
				Kaufpreis: 1200, Topf: mitteltopf.Schultraeger,
				SchuljahreMitAusleihe: 1, SchuljahreImBestand: 0,
			},
			wantBetrag:    1200,
			wantImSatz:    "ohne Altersabschlag",
			wantNichtSatz: "Verleihjahr",
		},
		{
			// der Fall, für den diese Prüfung da ist: Nach der Staffel wären es 10 % =
			// 1,20 €. Die Benutzungsordnung kennt diesen Abschlag nicht.
			name: "Büchereibuch, zehn Jahre alt — TROTZDEM voller Neuwert",
			groessen: repository.ErsatzwertGroessen{
				Kaufpreis: 1200, Topf: mitteltopf.Schultraeger,
				SchuljahreMitAusleihe: 7, SchuljahreImBestand: 10,
			},
			wantBetrag:    1200,
			wantImSatz:    "ohne Altersabschlag",
			wantNichtSatz: "%",
		},
		{
			name: "Büchereibuch ohne erfassten Preis",
			groessen: repository.ErsatzwertGroessen{
				Kaufpreis: 0, Topf: mitteltopf.Schultraeger,
				SchuljahreMitAusleihe: 2, SchuljahreImBestand: 3,
			},
			wantBetrag:    0,
			wantImSatz:    "kein Preis hinterlegt",
			wantOhnePreis: true,
		},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			got := ErsatzwertVorschlagAus(f.groessen, ersatzwert.PreisquelleListenpreis)

			if got.Betrag != f.wantBetrag {
				t.Errorf("Betrag = %s, want %s — die Zahl landet in einer Forderung "+
					"(Herleitung: %q)", got.Betrag.Text(), f.wantBetrag.Text(), got.Herleitung)
			}
			if !strings.Contains(got.Herleitung, f.wantImSatz) {
				t.Errorf("Herleitung = %q, erwartet darin %q", got.Herleitung, f.wantImSatz)
			}
			// Das Feld, nicht der Satz: Die Oberfläche entscheidet daran, ob sie „Ersatzwert
			// heute: 0,00 €" schreibt oder schweigt. Läse sie den Anfang der Herleitung, wäre
			// ein deutscher Satz die Schnittstelle: Ihn könnte dieser Test umformulieren, ohne
			// dass vorn etwas rot wird.
			if got.Bekannt == f.wantOhnePreis {
				t.Errorf("Bekannt = %v bei Betrag %s (%q) — erwartet %v",
					got.Bekannt, got.Betrag.Text(), got.Herleitung, !f.wantOhnePreis)
			}
			if f.wantNichtSatz != "" && strings.Contains(got.Herleitung, f.wantNichtSatz) {
				t.Errorf("Herleitung = %q — darf %q NICHT enthalten: Für den Bestand des "+
					"Schulträgers gilt die Staffel der Arbeitshilfe nicht",
					got.Herleitung, f.wantNichtSatz)
			}
			if got.Topf != f.groessen.Topf {
				t.Errorf("Topf = %q, erwartet %q — der Dialog benennt die Regel danach",
					got.Topf, f.groessen.Topf)
			}
		})
	}
}

// Die Herleitung nennt den Zustands-Abschlag — der Satz ist der Teil, den ein Mensch
// nachrechnet.
//
// Prüfung für die zweite Hälfte derselben Lücke: Dass die Zahl den Abschlag enthält, prüft
// pkg/ersatzwert, hier geht es um den Satz. Ein Betrag, der unerklärt um ein Fünftel kleiner
// ist als die Staffel, ist im Gespräch mit Eltern schlimmer als gar keine Begründung.
func TestErsatzwertVorschlagNenntDenZustandsAbschlag(t *testing.T) {
	// 60 % von 41,50 € = 24,90 €, abzüglich 20 % für den Zustand = 19,92 €.
	mitAbschlag := repository.ErsatzwertGroessen{
		Kaufpreis: 2000, Listenpreis: 4150, ZustandAbschlag: 20, Topf: mitteltopf.Land,
		SchuljahreMitAusleihe: 3, SchuljahreImBestand: 2,
	}
	got := ErsatzwertVorschlagAus(mitAbschlag, ersatzwert.PreisquelleListenpreis)

	if got.Betrag != 1992 {
		t.Errorf("Betrag = %s, erwartet 19,92 (Herleitung: %q)", got.Betrag.Text(), got.Herleitung)
	}
	for _, teil := range []string{"3. Verleihjahr", "60 %", "41,50", "Listenpreis",
		"abzüglich 20 % für den Zustand"} {
		if !strings.Contains(got.Herleitung, teil) {
			t.Errorf("Herleitung = %q, erwartet darin %q", got.Herleitung, teil)
		}
	}

	// Ohne Abschlag darf der Satz ihn nicht nennen: „abzüglich 0 %" liest sich wie ein
	// Fehler und lädt zur Rückfrage ein, die es nicht braucht.
	ohneAbschlag := mitAbschlag
	ohneAbschlag.ZustandAbschlag = 0
	ohne := ErsatzwertVorschlagAus(ohneAbschlag, ersatzwert.PreisquelleListenpreis)
	if ohne.Betrag != 2490 {
		t.Errorf("Betrag ohne Abschlag = %s, erwartet 24,90", ohne.Betrag.Text())
	}
	if strings.Contains(ohne.Herleitung, "abzüglich") {
		t.Errorf("Herleitung = %q — ohne Abschlag darf kein Abzug im Satz stehen",
			ohne.Herleitung)
	}
}
