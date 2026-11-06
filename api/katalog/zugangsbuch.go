package katalog

import (
	"net/http"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/internal/bestandsbuch"
	"bibliothek/pdf"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
)

// Das Zugangsbuch: die Gegenrichtung zum Abgangsbuch, mit demselben Aufbau und demselben
// Zeitraum. Die Unterschiede stehen in repository/zugangsbuch.go.

// zugangsbuchAntwort ist das Zugangsbuch, fertig in Abschnitte geteilt.
type zugangsbuchAntwort struct {
	Von        time.Time                            `json:"von"`
	Bis        time.Time                            `json:"bis"`
	Abschnitte []topfAbschnitt[bestandsbuch.Zugang] `json:"abschnitte"`
	Gesamt     int                                  `json:"gesamt"`
}

func zugangsbuchAntwortAus(buch repository.Zugangsbuch) zugangsbuchAntwort {
	return zugangsbuchAntwort{
		Von: buch.Von,
		Bis: buch.Bis,
		Abschnitte: abschnitteAus(bestandsbuch.Zugaenge(buch.Zeilen),
			func(z bestandsbuch.Zugang) string { return z.Topf }, func(z bestandsbuch.Zugang) int { return z.Anzahl }),
		Gesamt: len(buch.Zeilen),
	}
}

// zugangsbuchBlatt füllt die Eingabe des Blatts aus der Antwort für den Bildschirm: Das Blatt
// trägt dieselben Abschnitte, Überschriften und Zahlen wie die Liste, aus der es entsteht.
func zugangsbuchBlatt(antwort zugangsbuchAntwort) pdf.Zugangsbuch {
	blatt := pdf.Zugangsbuch{
		Von:        antwort.Von,
		Bis:        antwort.Bis,
		Abschnitte: make([]pdf.ZugangsAbschnitt, 0, len(antwort.Abschnitte)),
		Gesamt:     antwort.Gesamt,
	}
	for _, abschnitt := range antwort.Abschnitte {
		// Der Abschnitt ohne Topf trägt die Exemplare ohne hinterlegte Bestellung.
		if abschnitt.Topf == "" && len(abschnitt.Zeilen) > 0 {
			blatt.OhneZuordnung = true
		}
		zeilen := make([]pdf.ZugangsZeile, 0, len(abschnitt.Zeilen))
		for _, z := range abschnitt.Zeilen {
			zeilen = append(zeilen, pdf.ZugangsZeile{Datum: z.Datum, Titel: z.Titel, Anzahl: z.Anzahl,
				Barcodes: z.Barcodes, Lieferant: z.Lieferant})
		}
		blatt.Abschnitte = append(blatt.Abschnitte,
			pdf.ZugangsAbschnitt{Titel: abschnitt.Titel, Exemplare: abschnitt.Exemplare, Zeilen: zeilen})
	}
	return blatt
}

// Zugangsbuch liefert die Zugänge eines Zeitraums für den Bildschirm.
// GET /api/bestand/zugangsbuch?von=JJJJ-MM-TT&bis=JJJJ-MM-TT
func (t *Tueren) Zugangsbuch() http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		von, bis, err := bestandsbuchZeitraum(r, schulzeit.VorigesHalbjahr)
		if err != nil {
			return err
		}
		buch, err := repository.LadeZugangsbuch(r.Context(), t.DB.Pool, von, bis)
		if err != nil {
			return apierrors.Internal("Zugangsbuch konnte nicht gelesen werden", err)
		}
		tuer.AntworteJSON(w, http.StatusOK, zugangsbuchAntwortAus(buch))
		return nil
	})
}

// ZugangsbuchPDF druckt dasselbe als Blatt zum Abheften.
// GET /api/bestand/zugangsbuch/pdf?von=JJJJ-MM-TT&bis=JJJJ-MM-TT
func (t *Tueren) ZugangsbuchPDF() http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		von, bis, err := bestandsbuchZeitraum(r, schulzeit.VorigesHalbjahr)
		if err != nil {
			return err
		}
		buch, err := repository.LadeZugangsbuch(r.Context(), t.DB.Pool, von, bis)
		if err != nil {
			return apierrors.Internal("Zugangsbuch konnte nicht gelesen werden", err)
		}
		return t.antworteBestandsbuchBlatt(r.Context(), w, "Zugangsbuch", buch.Von, buch.Bis,
			func(schule pdf.SchuleInfo) ([]byte, error) {
				return pdf.ErzeugeZugangsbuch(zugangsbuchBlatt(zugangsbuchAntwortAus(buch)), schule)
			})
	})
}
