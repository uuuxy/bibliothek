package katalog

import (
	"fmt"
	"net/http"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/pdf"
	"bibliothek/repository"
)

// Das Abgangsbuch: zwei Türen auf dieselbe Abfrage, eine für den Bildschirm (JSON) und eine
// für das Blatt (PDF). Den Zeitraum bestimmt eine Stelle, bestandsbuchZeitraum: Belegte die
// Oberfläche ihn selbst vor, gäbe es zwei Auslegungen von „laufendes Halbjahr", und der
// Ausdruck deckte einen anderen Zeitraum ab als die Liste, aus der er entstand.

// abgangsbuchAntwort ist das Abgangsbuch, wie Bildschirm und Blatt es lesen: fertig in
// Abschnitte geteilt, mit den Überschriften des Servers. Die Oberfläche gruppiert nicht selbst,
// sonst stünden auf dem Blatt und auf dem Bildschirm zwei Wörter für denselben Topf.
type abgangsbuchAntwort struct {
	Von           time.Time                                `json:"von"`
	Bis           time.Time                                `json:"bis"`
	Abschnitte    []topfAbschnitt[repository.AbgangsZeile] `json:"abschnitte"`
	Gesamt        int                                      `json:"gesamt"`
	OhneZeitpunkt int                                      `json:"ohne_zeitpunkt"`
	// AusKatalogGeloescht zählt die im Zeitraum körperlich gelöschten Exemplare. Sie stehen
	// in keinem Abschnitt, denn mit der Zeile ist auch ihr Titel gegangen
	// (repository/abgangsbuch.go).
	AusKatalogGeloescht int `json:"aus_katalog_geloescht"`
}

func abgangsbuchAntwortAus(buch repository.Abgangsbuch) abgangsbuchAntwort {
	return abgangsbuchAntwort{
		Von:           buch.Von,
		Bis:           buch.Bis,
		Abschnitte:    abschnitteAus(buch.Zeilen, func(z repository.AbgangsZeile) string { return z.Topf }),
		Gesamt:        len(buch.Zeilen),
		OhneZeitpunkt: buch.OhneZeitpunkt,

		AusKatalogGeloescht: buch.AusKatalogGeloescht,
	}
}

// abgangsbuchBlatt füllt die Eingabe des Blatts aus der Antwort für den Bildschirm: Das Blatt
// trägt dieselben Abschnitte, Überschriften und Zahlen wie die Liste, aus der es entsteht.
func abgangsbuchBlatt(antwort abgangsbuchAntwort) pdf.Abgangsbuch {
	blatt := pdf.Abgangsbuch{
		Von:                 antwort.Von,
		Bis:                 antwort.Bis,
		Abschnitte:          make([]pdf.AbgangsAbschnitt, 0, len(antwort.Abschnitte)),
		Gesamt:              antwort.Gesamt,
		OhneZeitpunkt:       antwort.OhneZeitpunkt,
		AusKatalogGeloescht: antwort.AusKatalogGeloescht,
	}
	for _, abschnitt := range antwort.Abschnitte {
		zeilen := make([]pdf.AbgangsZeile, 0, len(abschnitt.Zeilen))
		for _, z := range abschnitt.Zeilen {
			zeilen = append(zeilen, pdf.AbgangsZeile{
				Datum: z.Datum, Barcode: z.Barcode, Titel: z.Titel, Signatur: z.Signatur, GrundText: z.GrundText,
			})
		}
		blatt.Abschnitte = append(blatt.Abschnitte, pdf.AbgangsAbschnitt{Titel: abschnitt.Titel, Zeilen: zeilen})
	}
	return blatt
}

// Abgangsbuch liefert die Abgänge eines Zeitraums für den Bildschirm.
// GET /api/bestand/abgangsbuch?von=JJJJ-MM-TT&bis=JJJJ-MM-TT
func (t *Tueren) Abgangsbuch() http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		von, bis, err := bestandsbuchZeitraum(r)
		if err != nil {
			return apierrors.BadRequest(err.Error(), err)
		}
		buch, err := repository.LadeAbgangsbuch(r.Context(), t.DB.Pool, von, bis)
		if err != nil {
			return apierrors.Internal("Abgangsbuch konnte nicht gelesen werden", err)
		}
		tuer.AntworteJSON(w, http.StatusOK, abgangsbuchAntwortAus(buch))
		return nil
	})
}

// AbgangsbuchPDF druckt dasselbe als Blatt zum Abheften.
// GET /api/bestand/abgangsbuch/pdf?von=JJJJ-MM-TT&bis=JJJJ-MM-TT
func (t *Tueren) AbgangsbuchPDF() http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		von, bis, err := bestandsbuchZeitraum(r)
		if err != nil {
			return apierrors.BadRequest(err.Error(), err)
		}
		ctx := r.Context()
		buch, err := repository.LadeAbgangsbuch(ctx, t.DB.Pool, von, bis)
		if err != nil {
			return apierrors.Internal("Abgangsbuch konnte nicht gelesen werden", err)
		}

		einst, err := repository.NewEinstellungsAbfragen(t.DB.Pool).Lade(ctx)
		if err != nil {
			return apierrors.Internal(tuer.MeldungEinstellungenUnlesbar, err)
		}
		schule := tuer.SchuleAus(einst)

		blatt, err := pdf.ErzeugeAbgangsbuch(abgangsbuchBlatt(abgangsbuchAntwortAus(buch)), schule)
		if err != nil {
			return apierrors.Internal("Abgangsbuch konnte nicht gedruckt werden", err)
		}
		w.Header().Set(tuer.HeaderContentType, tuer.ContentTypePDF)
		w.Header().Set(tuer.HeaderContentDisposition, fmt.Sprintf(`inline; filename="Abgangsbuch_%s_bis_%s.pdf"`,
			buch.Von.Format(tuer.DatumsformatISO), buch.Bis.Format(tuer.DatumsformatISO)))
		w.Header().Set(tuer.HeaderContentLength, fmt.Sprint(len(blatt)))
		if _, err := w.Write(blatt); err != nil {
			return nil // Die Verbindung ist weg, der Browser hat abgebrochen.
		}
		return nil
	})
}
