package api

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"bibliothek/apierrors"
	"bibliothek/pdf"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
)

// Das Abgangsbuch — Punkt 1 des Protokolls vom 16.09.2026.
//
// Zwei Türen auf dieselbe Abfrage: eine für den Bildschirm (JSON) und eine für das Blatt
// (PDF). Der Zeitraum wird an EINER Stelle bestimmt (abgangsbuchZeitraum): Ließe die
// Oberfläche ihn selbst vorbelegen, gäbe es zwei Auslegungen von „laufendes Halbjahr" —
// und der Ausdruck deckte am Ende einen anderen Zeitraum ab als die Liste, aus der er
// entstand.

// abgangsbuchZeitraum liest von/bis aus der Anfrage; fehlt eines, gilt das laufende
// Schulhalbjahr (Stichtage 15.3./15.9., siehe schulzeit.Halbjahr).
func abgangsbuchZeitraum(r *http.Request) (von, bis time.Time, err error) {
	von, bis = schulzeit.Halbjahr(schulzeit.Jetzt())
	lies := func(schluessel string, ziel *time.Time) error {
		roh := r.URL.Query().Get(schluessel)
		if roh == "" {
			return nil
		}
		t, fehler := time.ParseInLocation(dateFormatISO, roh, schulzeit.Zone())
		if fehler != nil {
			return fmt.Errorf("%s muss ein Datum sein (JJJJ-MM-TT)", schluessel)
		}
		*ziel = t
		return nil
	}
	if err = lies("von", &von); err != nil {
		return von, bis, err
	}
	if err = lies("bis", &bis); err != nil {
		return von, bis, err
	}
	if bis.Before(von) {
		return von, bis, errors.New("das Ende des Zeitraums liegt vor seinem Anfang")
	}
	return von, bis, nil
}

// AbgangsbuchHandler liefert die Abgänge eines Zeitraums für den Bildschirm.
// GET /api/bestand/abgangsbuch?von=JJJJ-MM-TT&bis=JJJJ-MM-TT
func (s *Server) AbgangsbuchHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		von, bis, err := abgangsbuchZeitraum(r)
		if err != nil {
			return apierrors.BadRequest(err.Error(), err)
		}
		buch, err := repository.LadeAbgangsbuch(r.Context(), s.DB.Pool, von, bis)
		if err != nil {
			return apierrors.Internal("Abgangsbuch konnte nicht gelesen werden", err)
		}
		RespondJSON(w, http.StatusOK, buch)
		return nil
	})
}

// AbgangsbuchPDFHandler druckt dasselbe als Blatt zum Abheften.
// GET /api/bestand/abgangsbuch/pdf?von=JJJJ-MM-TT&bis=JJJJ-MM-TT
func (s *Server) AbgangsbuchPDFHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		von, bis, err := abgangsbuchZeitraum(r)
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
