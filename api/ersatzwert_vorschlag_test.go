package api

import (
	"strings"
	"testing"

	"bibliothek/pkg/ersatzwert"
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
		// wantOhnePreis: Dem Betrag liegt KEIN Preis zugrunde. Die Vorgabe ist bewusst
		// „Preis vorhanden" — ein neuer Fall, der ihn vergisst, wird laut statt still.
		wantOhnePreis bool
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
			wantBetrag:    0,
			wantImSatz:    "kein Preis hinterlegt",
			wantOhnePreis: true,
		},
		{
			name: "Büchereibuch, neu — Neuwert",
			groessen: repository.ErsatzwertGroessen{
				Kaufpreis: 12.00, IstLernmittel: false,
				SchuljahreMitAusleihe: 1, SchuljahreImBestand: 0,
			},
			wantBetrag:    12.00,
			wantImSatz:    "ohne Altersabschlag",
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
			wantImSatz:    "ohne Altersabschlag",
			wantNichtSatz: "%",
		},
		{
			name: "Büchereibuch ohne erfassten Preis",
			groessen: repository.ErsatzwertGroessen{
				Kaufpreis: 0, IstLernmittel: false,
				SchuljahreMitAusleihe: 2, SchuljahreImBestand: 3,
			},
			wantBetrag:    0,
			wantImSatz:    "kein Preis hinterlegt",
			wantOhnePreis: true,
		},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			got := ersatzwertVorschlagAus(f.groessen, ersatzwert.PreisquelleListenpreis)

			if got.Betrag != f.wantBetrag {
				t.Errorf("Betrag = %.2f, want %.2f — die Zahl landet in einer Forderung "+
					"(Herleitung: %q)", got.Betrag, f.wantBetrag, got.Herleitung)
			}
			if !strings.Contains(got.Herleitung, f.wantImSatz) {
				t.Errorf("Herleitung = %q, erwartet darin %q", got.Herleitung, f.wantImSatz)
			}
			// Das Feld, NICHT der Satz: Die Oberfläche entscheidet daran, ob sie „Ersatzwert
			// heute: 0,00 €" schreibt oder schweigt. Bis zum 17.09.2026 las sie stattdessen
			// den Anfang der Herleitung — ein deutscher Satz als Schnittstelle, den dieser
			// Test hier hätte umformulieren können, ohne dass vorn etwas rot geworden wäre.
			if got.Bekannt == f.wantOhnePreis {
				t.Errorf("Bekannt = %v bei Betrag %.2f (%q) — erwartet %v",
					got.Bekannt, got.Betrag, got.Herleitung, !f.wantOhnePreis)
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

// Die Herleitung nennt den Zustands-Abschlag — der Satz ist der Teil, den ein Mensch
// nachrechnet.
//
// Gate für die zweite Hälfte derselben Lücke: Dass die ZAHL den Abschlag enthält, prüft
// pkg/ersatzwert. Ob der SATZ ihn nennt, prüfte bis hierher niemand — und ein Betrag, der
// unerklärt um ein Fünftel kleiner ist als die Staffel, ist im Gespräch mit Eltern
// schlimmer als gar keine Begründung.
func TestErsatzwertVorschlagNenntDenZustandsAbschlag(t *testing.T) {
	// 60 % von 41,50 € = 24,90 €, abzüglich 20 % für den Zustand = 19,92 €.
	mitAbschlag := repository.ErsatzwertGroessen{
		Kaufpreis: 20.00, Listenpreis: 41.50, ZustandAbschlag: 20, IstLernmittel: true,
		SchuljahreMitAusleihe: 3, SchuljahreImBestand: 2,
	}
	got := ersatzwertVorschlagAus(mitAbschlag, ersatzwert.PreisquelleListenpreis)

	if got.Betrag != 19.92 {
		t.Errorf("Betrag = %.2f, want 19.92 (Herleitung: %q)", got.Betrag, got.Herleitung)
	}
	for _, teil := range []string{"3. Verleihjahr", "60 %", "41,50", "Listenpreis",
		"abzüglich 20 % für den Zustand"} {
		if !strings.Contains(got.Herleitung, teil) {
			t.Errorf("Herleitung = %q, erwartet darin %q", got.Herleitung, teil)
		}
	}

	// Ohne Abschlag darf der Satz ihn NICHT nennen: „abzüglich 0 %" liest sich wie ein
	// Fehler und lädt zur Rückfrage ein, die es nicht braucht.
	ohneAbschlag := mitAbschlag
	ohneAbschlag.ZustandAbschlag = 0
	ohne := ersatzwertVorschlagAus(ohneAbschlag, ersatzwert.PreisquelleListenpreis)
	if ohne.Betrag != 24.90 {
		t.Errorf("Betrag ohne Abschlag = %.2f, want 24.90", ohne.Betrag)
	}
	if strings.Contains(ohne.Herleitung, "abzüglich") {
		t.Errorf("Herleitung = %q — ohne Abschlag darf kein Abzug im Satz stehen",
			ohne.Herleitung)
	}
}

// DREI Türen, EIN Betrag — das Gate gegen die Ungleichheit, die es bis zum 17.09.2026 gab.
//
// Der Melde-Dialog wählte die Regel nach IstLernmittel, die beiden Bescheid-Wege wendeten
// die Staffel der Arbeitshilfe auf ALLES an. Für ein zehn Jahre altes Büchereibuch nannte
// der Dialog 14,00 € und der Brief 1,40 € — und der Brief war der falsche: Die Staffel
// steht in einer Arbeitshilfe für Lehrwerke der Lernmittelfreiheit, das Geld der Bücherei
// gehört dem Schulträger.
//
// Geprüft wird mit DENSELBEN Zahlen über alle drei Wege. Der Test vergleicht sie
// gegeneinander, nicht gegen eine abgetippte Erwartung: Wer die Rechnung ändert, ändert
// alle drei oder wird hier rot.
func TestDreiWegeNennenDenselbenBetrag(t *testing.T) {
	for _, istLernmittel := range []bool{true, false} {
		name := "Büchereibuch"
		if istLernmittel {
			name = "Lernmittel"
		}
		t.Run(name, func(t *testing.T) {
			// Zehn Jahre im Bestand, 20 % Wasserschaden, beide Preise erfasst.
			const kauf, liste = 8.00, 14.00
			const abschlag = 20
			const mitAusleihe, imBestand = 7, 10

			ausExemplar := ersatzwertVorschlagAus(repository.ErsatzwertGroessen{
				Kaufpreis: kauf, Listenpreis: liste, ZustandAbschlag: abschlag,
				SchuljahreMitAusleihe: mitAusleihe, SchuljahreImBestand: imBestand,
				IstLernmittel: istLernmittel,
			}, ersatzwert.PreisquelleListenpreis)

			ausForderung := bescheidVorschlagAus(repository.OffeneForderung{
				Kaufpreis: kauf, Listenpreis: liste, ZustandAbschlag: abschlag,
				SchuljahreMitAusleihe: mitAusleihe, SchuljahreImBestand: imBestand,
				IstLernmittel: istLernmittel,
			}, ersatzwert.PreisquelleListenpreis)

			ausAusleihe := bescheidVorschlagAusAusleihe(repository.UeberfaelligeAusleihe{
				Kaufpreis: kauf, Listenpreis: liste, ZustandAbschlag: abschlag,
				SchuljahreMitAusleihe: mitAusleihe, SchuljahreImBestand: imBestand,
				IstLernmittel: istLernmittel,
			}, ersatzwert.PreisquelleListenpreis)

			if ausForderung.Betrag != ausExemplar.Betrag || ausAusleihe.Betrag != ausExemplar.Betrag {
				t.Errorf("drei Wege, drei Beträge: Exemplar %.2f, Forderung %.2f, Ausleihe %.2f",
					ausExemplar.Betrag, ausForderung.Betrag, ausAusleihe.Betrag)
			}
			if ausForderung.Herleitung != ausExemplar.Herleitung ||
				ausAusleihe.Herleitung != ausExemplar.Herleitung {
				t.Errorf("drei Wege, drei Begründungen:\n  Exemplar:  %q\n  Forderung: %q\n  Ausleihe:  %q",
					ausExemplar.Herleitung, ausForderung.Herleitung, ausAusleihe.Herleitung)
			}

			// Und die Regel ist die richtige: Für das Büchereibuch darf das Alter nichts
			// abziehen (14 € minus 20 % Zustand), für das Lernmittel schon (10 % von 14 €,
			// davon 20 % ab).
			wantBetrag := 11.20
			if istLernmittel {
				wantBetrag = 1.12
			}
			if ausExemplar.Betrag != wantBetrag {
				t.Errorf("Betrag = %.2f, want %.2f (Herleitung: %q)",
					ausExemplar.Betrag, wantBetrag, ausExemplar.Herleitung)
			}
		})
	}
}
