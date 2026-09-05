package api

// lmf_termine.go — der LMF-Plan lesend: Rückgabe- und Ausgabetermine je Klasse
// (Register, Entscheidung 3, 05.09.2026). Das Kollegium liest sie im Portal (für alle
// gleich, keine Personalisierung), das PDF sieht aus wie die bisherige Excel-Liste.
// Lesen verlangt nur eine Sitzung (Stufe 0: Daten und Klassen, kein Schülerbezug).
// Geschrieben wird der Plan als Reihenfolge (lmf_plan.go, Migration 097).

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"bibliothek/apierrors"
	"bibliothek/pdf"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
)

// LmfPlanAntwort ist die Antwort von GET /api/lmf-termine.
type LmfPlanAntwort struct {
	// Ab ist das Datum, ab dem gelistet wird (Beginn des laufenden Schuljahres), leer bei ?alle=1.
	Ab      string                 `json:"ab"`
	Termine []repository.LmfTermin `json:"termine"`
	// OhneRueckgabeTermin nennt Klassen mit Schülern, die ab dem Datum keinen
	// Rückgabe-Termin haben — der Plan startet leer, die Seite zeigt, wer fehlt.
	OhneRueckgabeTermin []string `json:"ohne_rueckgabe_termin"`
}

// lmfPlanAb bestimmt, ab wann gelistet wird: der 1. August des laufenden Schuljahres,
// damit im Juni die neue Rückgabe und die August-Ausgabe zusammen stehen und im
// Herbst die alte Rückgabe verschwunden ist; ?alle=1 hebt die Grenze auf.
func (s *Server) lmfPlanAb(r *http.Request) time.Time {
	if r.URL.Query().Get("alle") == "1" {
		return time.Time{}
	}
	return repository.SchuljahrBeginn(s.jetzt())
}

// GetLmfTermineHandler listet den Plan.
// @Summary      LMF-Plan
// @Description  Rückgabe- und Ausgabetermine je Klasse ab Beginn des laufenden Schuljahres (?alle=1: alle), plus Klassen ohne Rückgabe-Termin.
// @Tags         lernmittel
// @Produce      json
// @Success      200  {object}  LmfPlanAntwort
// @Router       /lmf-termine [get]
func (s *Server) GetLmfTermineHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		repo := repository.NewLmfTerminRepository(s.DB.Pool)
		ab := s.lmfPlanAb(r)
		termine, err := repo.ListLmfTermine(r.Context(), ab)
		if err != nil {
			return apierrors.Internal("LMF-Plan laden", err)
		}
		antwort := LmfPlanAntwort{Termine: termine, OhneRueckgabeTermin: []string{}}
		if !ab.IsZero() {
			antwort.Ab = ab.Format("2006-01-02")
			if antwort.OhneRueckgabeTermin, err = repo.KlassenOhneRueckgabeTermin(r.Context(), ab); err != nil {
				return apierrors.Internal("Klassen ohne Termin laden", err)
			}
		}
		RespondJSON(w, http.StatusOK, antwort)
		return nil
	})
}

// lmfPlanAbschnitte gruppiert die Termine für das PDF: erst Rückgabe, dann Ausgabe,
// je Abschnitt in der Reihenfolge des Plans.
func lmfPlanAbschnitte(termine []repository.LmfTermin) ([]pdf.LmfPlanAbschnitt, error) {
	abschnitte := []pdf.LmfPlanAbschnitt{
		{Titel: "BÜCHERRÜCKGABE"},
		{Titel: "BÜCHERAUSGABE"},
	}
	for _, t := range termine {
		datum, err := time.ParseInLocation("2006-01-02", t.Datum, schulzeit.Zone())
		if err != nil {
			return nil, err
		}
		z := pdf.LmfPlanZeile{Datum: datum, Stunde: t.Stunde, Klassen: strings.Join(t.Klassen, "/"), Vermerk: t.Vermerk}
		if t.Art == repository.LmfTerminAusgabe {
			abschnitte[1].Zeilen = append(abschnitte[1].Zeilen, z)
		} else {
			abschnitte[0].Zeilen = append(abschnitte[0].Zeilen, z)
		}
	}
	return abschnitte, nil
}

// GetLmfPlanPDFHandler liefert den Plan als PDF in der Form der bisherigen Excel-Liste.
// @Summary      LMF-Plan als PDF
// @Tags         lernmittel
// @Produce      application/pdf
// @Router       /lmf-termine/pdf [get]
func (s *Server) GetLmfPlanPDFHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repo := repository.NewLmfTerminRepository(s.DB.Pool)
		termine, err := repo.ListLmfTermine(r.Context(), s.lmfPlanAb(r))
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if len(termine) == 0 {
			apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("der LMF-Plan ist leer"))
			return
		}
		abschnitte, err := lmfPlanAbschnitte(termine)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		pdfBytes, err := pdf.GenerateLmfPlan(abschnitte, s.jetzt())
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		filename := "LMF-Plan.pdf"
		w.Header().Set(headerContentType, contentTypePDF)
		w.Header().Set(headerContentDisposition, fmt.Sprintf(`attachment; filename="%s"`, filename))
		w.Header().Set(headerContentLength, fmt.Sprint(len(pdfBytes)))
		http.ServeContent(w, r, filename, time.Now(), bytes.NewReader(pdfBytes))
	}
}
