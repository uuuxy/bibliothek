package api

import (
	"fmt"
	"net/http"
	"time"

	"bibliothek/apierrors"
	"bibliothek/pdf"
	"bibliothek/repository"
)

// Das Abgangsbuch — Punkt 1 des Protokolls vom 16.09.2026.
//
// Zwei Türen auf dieselbe Abfrage: eine für den Bildschirm (JSON) und eine für das Blatt
// (PDF). Der Zeitraum wird an EINER Stelle bestimmt (abgangsbuchZeitraum): Ließe die
// Oberfläche ihn selbst vorbelegen, gäbe es zwei Auslegungen von „laufendes Halbjahr" —
// und der Ausdruck deckte am Ende einen anderen Zeitraum ab als die Liste, aus der er
// entstand.

// AbgangsbuchAntwort ist das Abgangsbuch, wie Bildschirm und Blatt es lesen: fertig in
// Abschnitte geteilt, mit den Überschriften des Servers. Die Oberfläche gruppiert NICHT
// selbst — sonst stünden auf dem Blatt und auf dem Bildschirm zwei Wörter für denselben Topf.
type AbgangsbuchAntwort struct {
	Von           time.Time                            `json:"von"`
	Bis           time.Time                            `json:"bis"`
	Abschnitte    []Abschnitt[repository.AbgangsZeile] `json:"abschnitte"`
	Gesamt        int                                  `json:"gesamt"`
	OhneZeitpunkt int                                  `json:"ohne_zeitpunkt"`
}

func abgangsbuchAntwort(buch repository.Abgangsbuch) AbgangsbuchAntwort {
	return AbgangsbuchAntwort{
		Von:           buch.Von,
		Bis:           buch.Bis,
		Abschnitte:    abschnitteAus(buch.Zeilen, func(z repository.AbgangsZeile) string { return z.Topf }),
		Gesamt:        len(buch.Zeilen),
		OhneZeitpunkt: buch.OhneZeitpunkt,
	}
}

// AbgangsbuchHandler liefert die Abgänge eines Zeitraums für den Bildschirm.
// GET /api/bestand/abgangsbuch?von=JJJJ-MM-TT&bis=JJJJ-MM-TT
func (s *Server) AbgangsbuchHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		von, bis, err := bestandsbuchZeitraum(r)
		if err != nil {
			return apierrors.BadRequest(err.Error(), err)
		}
		buch, err := repository.LadeAbgangsbuch(r.Context(), s.DB.Pool, von, bis)
		if err != nil {
			return apierrors.Internal("Abgangsbuch konnte nicht gelesen werden", err)
		}
		RespondJSON(w, http.StatusOK, abgangsbuchAntwort(buch))
		return nil
	})
}

// AbgangsbuchPDFHandler druckt dasselbe als Blatt zum Abheften.
// GET /api/bestand/abgangsbuch/pdf?von=JJJJ-MM-TT&bis=JJJJ-MM-TT
func (s *Server) AbgangsbuchPDFHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		von, bis, err := bestandsbuchZeitraum(r)
		if err != nil {
			return apierrors.BadRequest(err.Error(), err)
		}
		ctx := r.Context()
		buch, err := repository.LadeAbgangsbuch(ctx, s.DB.Pool, von, bis)
		if err != nil {
			return apierrors.Internal("Abgangsbuch konnte nicht gelesen werden", err)
		}

		einst, err := repository.NewSystemSettingsRepository(s.DB.Pool).GetSettings(ctx)
		if err != nil {
			return apierrors.Internal("Einstellungen konnten nicht gelesen werden", err)
		}
		schule := pdf.SchuleInfo{
			Name: einst.SchuleName, Strasse: einst.SchuleStrasse,
			PLZ: einst.SchulePLZ, Ort: einst.SchuleOrt,
		}

		blatt, err := generateAbgangsbuchPDF(buch, schule)
		if err != nil {
			return apierrors.Internal("Abgangsbuch konnte nicht gedruckt werden", err)
		}
		w.Header().Set(headerContentType, contentTypePDF)
		w.Header().Set(headerContentDisposition, fmt.Sprintf(`inline; filename="Abgangsbuch_%s_bis_%s.pdf"`,
			buch.Von.Format(dateFormatISO), buch.Bis.Format(dateFormatISO)))
		w.Header().Set(headerContentLength, fmt.Sprint(len(blatt)))
		if _, err := w.Write(blatt); err != nil {
			return nil // Verbindung weg — der Client hat abgebrochen.
		}
		return nil
	})
}
