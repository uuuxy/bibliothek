package schaden

import (
	"testing"

	"bibliothek/internal/service"
	"bibliothek/pkg/ersatzwert"
	"bibliothek/pkg/mitteltopf"
	"bibliothek/repository"
)

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
	for _, topf := range []string{mitteltopf.Land, mitteltopf.Schultraeger} {
		name := "Büchereibuch"
		if topf == mitteltopf.Land {
			name = "Buch des Landes"
		}
		t.Run(name, func(t *testing.T) {
			// Zehn Jahre im Bestand, 20 % Wasserschaden, beide Preise erfasst.
			const kauf, liste = 8.00, 14.00
			const abschlag = 20
			const mitAusleihe, imBestand = 7, 10

			ausExemplar := service.ErsatzwertVorschlagAus(repository.ErsatzwertGroessen{
				Kaufpreis: kauf, Listenpreis: liste, ZustandAbschlag: abschlag,
				SchuljahreMitAusleihe: mitAusleihe, SchuljahreImBestand: imBestand,
				Topf: topf,
			}, ersatzwert.PreisquelleListenpreis)

			ausForderung := bescheidVorschlagAus(repository.OffeneForderung{
				Kaufpreis: kauf, Listenpreis: liste, ZustandAbschlag: abschlag,
				SchuljahreMitAusleihe: mitAusleihe, SchuljahreImBestand: imBestand,
				Topf: topf,
			}, ersatzwert.PreisquelleListenpreis)

			ausAusleihe := bescheidVorschlagAusAusleihe(repository.UeberfaelligeAusleihe{
				Kaufpreis: kauf, Listenpreis: liste, ZustandAbschlag: abschlag,
				SchuljahreMitAusleihe: mitAusleihe, SchuljahreImBestand: imBestand,
				Topf: topf,
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
			// abziehen (14 € minus 20 % Zustand), für das Buch des Landes schon (10 % von
			// 14 €, davon 20 % ab).
			wantBetrag := 11.20
			if topf == mitteltopf.Land {
				wantBetrag = 1.12
			}
			if ausExemplar.Betrag != wantBetrag {
				t.Errorf("Betrag = %.2f, want %.2f (Herleitung: %q)",
					ausExemplar.Betrag, wantBetrag, ausExemplar.Herleitung)
			}
		})
	}
}
