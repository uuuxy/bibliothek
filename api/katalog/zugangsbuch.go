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

// Das Zugangsbuch: die Gegenrichtung zum Abgangsbuch, mit demselben Aufbau und demselben
// Zeitraum. Die Unterschiede stehen in repository/zugangsbuch.go.

// zugangsbuchAntwort ist das Zugangsbuch, fertig in Abschnitte geteilt.
type zugangsbuchAntwort struct {
	Von        time.Time                                `json:"von"`
	Bis        time.Time                                `json:"bis"`
	Abschnitte []topfAbschnitt[repository.ZugangsZeile] `json:"abschnitte"`
	Gesamt     int                                      `json:"gesamt"`
}

func zugangsbuchAntwortAus(buch repository.Zugangsbuch) zugangsbuchAntwort {
	return zugangsbuchAntwort{
		Von:        buch.Von,
		Bis:        buch.Bis,
		Abschnitte: abschnitteAus(buch.Zeilen, func(z repository.ZugangsZeile) string { return z.Topf }),
		Gesamt:     len(buch.Zeilen),
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
			zeilen = append(zeilen, pdf.ZugangsZeile{Datum: z.Datum, Barcode: z.Barcode, Titel: z.Titel, Lieferant: z.Lieferant})
		}
		blatt.Abschnitte = append(blatt.Abschnitte, pdf.ZugangsAbschnitt{Titel: abschnitt.Titel, Zeilen: zeilen})
	}
	return blatt
}

// Zugangsbuch liefert die Zugänge eines Zeitraums für den Bildschirm.
// GET /api/bestand/zugangsbuch?von=JJJJ-MM-TT&bis=JJJJ-MM-TT
func (t *Tueren) Zugangsbuch() http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		von, bis, err := bestandsbuchZeitraum(r)
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
		von, bis, err := bestandsbuchZeitraum(r)
		if err != nil {
			return err
		}
		ctx := r.Context()
		buch, err := repository.LadeZugangsbuch(ctx, t.DB.Pool, von, bis)
		if err != nil {
			return apierrors.Internal("Zugangsbuch konnte nicht gelesen werden", err)
		}

		einst, err := repository.NewEinstellungsAbfragen(t.DB.Pool).Lade(ctx)
		if err != nil {
			return apierrors.Internal(tuer.MeldungEinstellungenUnlesbar, err)
		}
		schule := tuer.SchuleAus(einst)

		blatt, err := pdf.ErzeugeZugangsbuch(zugangsbuchBlatt(zugangsbuchAntwortAus(buch)), schule)
		if err != nil {
			return apierrors.Internal("Zugangsbuch konnte nicht gedruckt werden", err)
		}
		w.Header().Set(tuer.HeaderContentType, tuer.ContentTypePDF)
		w.Header().Set(tuer.HeaderContentDisposition, fmt.Sprintf(`inline; filename="Zugangsbuch_%s_bis_%s.pdf"`,
			buch.Von.Format(tuer.DatumsformatISO), buch.Bis.Format(tuer.DatumsformatISO)))
		w.Header().Set(tuer.HeaderContentLength, fmt.Sprint(len(blatt)))
		if _, err := w.Write(blatt); err != nil {
			return nil // Die Verbindung ist weg, der Browser hat abgebrochen.
		}
		return nil
	})
}
