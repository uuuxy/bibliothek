package api

import (
	"fmt"
	"net/http"
	"time"

	"bibliothek/apierrors"
	"bibliothek/pdf"
	"bibliothek/repository"
)

// Das Zugangsbuch — die Gegenrichtung zum Abgangsbuch, zweite Hälfte von Punkt 1 des
// Protokolls vom 16.09.2026. Aufbau und Zeitraum sind dieselben; die Unterschiede stehen
// in repository/zugangsbuch.go.

// ZugangsbuchAntwort ist das Zugangsbuch, fertig in Abschnitte geteilt.
type ZugangsbuchAntwort struct {
	Von        time.Time                            `json:"von"`
	Bis        time.Time                            `json:"bis"`
	Abschnitte []Abschnitt[repository.ZugangsZeile] `json:"abschnitte"`
	Gesamt     int                                  `json:"gesamt"`
}

func zugangsbuchAntwort(buch repository.Zugangsbuch) ZugangsbuchAntwort {
	return ZugangsbuchAntwort{
		Von:        buch.Von,
		Bis:        buch.Bis,
		Abschnitte: abschnitteAus(buch.Zeilen, func(z repository.ZugangsZeile) string { return z.Topf }),
		Gesamt:     len(buch.Zeilen),
	}
}

// ZugangsbuchHandler liefert die Zugänge eines Zeitraums für den Bildschirm.
// GET /api/bestand/zugangsbuch?von=JJJJ-MM-TT&bis=JJJJ-MM-TT
func (s *Server) ZugangsbuchHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		von, bis, err := bestandsbuchZeitraum(r)
		if err != nil {
			return apierrors.BadRequest(err.Error(), err)
		}
		buch, err := repository.LadeZugangsbuch(r.Context(), s.DB.Pool, von, bis)
		if err != nil {
			return apierrors.Internal("Zugangsbuch konnte nicht gelesen werden", err)
		}
		RespondJSON(w, http.StatusOK, zugangsbuchAntwort(buch))
		return nil
	})
}

// ZugangsbuchPDFHandler druckt dasselbe als Blatt zum Abheften.
// GET /api/bestand/zugangsbuch/pdf?von=JJJJ-MM-TT&bis=JJJJ-MM-TT
func (s *Server) ZugangsbuchPDFHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		von, bis, err := bestandsbuchZeitraum(r)
		if err != nil {
			return apierrors.BadRequest(err.Error(), err)
		}
		ctx := r.Context()
		buch, err := repository.LadeZugangsbuch(ctx, s.DB.Pool, von, bis)
		if err != nil {
			return apierrors.Internal("Zugangsbuch konnte nicht gelesen werden", err)
		}

		einst, err := repository.NewSystemSettingsRepository(s.DB.Pool).GetSettings(ctx)
		if err != nil {
			return apierrors.Internal("Einstellungen konnten nicht gelesen werden", err)
		}
		schule := pdf.SchuleInfo{
			Name: einst.SchuleName, Strasse: einst.SchuleStrasse,
			PLZ: einst.SchulePLZ, Ort: einst.SchuleOrt,
		}

		blatt, err := generateZugangsbuchPDF(buch, schule)
		if err != nil {
			return apierrors.Internal("Zugangsbuch konnte nicht gedruckt werden", err)
		}
		w.Header().Set(headerContentType, contentTypePDF)
		w.Header().Set(headerContentDisposition, fmt.Sprintf(`inline; filename="Zugangsbuch_%s_bis_%s.pdf"`,
			buch.Von.Format(dateFormatISO), buch.Bis.Format(dateFormatISO)))
		w.Header().Set(headerContentLength, fmt.Sprint(len(blatt)))
		if _, err := w.Write(blatt); err != nil {
			return nil // Verbindung weg — der Client hat abgebrochen.
		}
		return nil
	})
}
