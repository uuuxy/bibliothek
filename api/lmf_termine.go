package api

// lmf_termine.go — der LMF-Plan lesend: Rückgabe- und Ausgabetermine je Klasse
// (Register, Entscheidung 3, 05.09.2026). Das Kollegium liest sie im Portal (für alle
// gleich, keine Personalisierung), das PDF sieht aus wie die bisherige Excel-Liste.
// Lesen verlangt nur eine Sitzung (Stufe 0: Daten und Klassen, kein Schülerbezug).
// Geschrieben wird der Plan als Reihenfolge (lmf_plan.go, Migration 097). Sichtbar sind
// nur veröffentlichte Pläne (Migration 100); den Entwurf als PDF für die Schulleitung
// bekommt nur der Planer (edit_books) über eine eigene Route.

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
	// Eingangsjahrgaenge (Einstellung): die Jahrgänge, die nach den Ferien Bücher
	// bekommen — für den erklärenden Satz über der Tabelle.
	Eingangsjahrgaenge []int `json:"eingangsjahrgaenge"`
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
		eingang, err := s.lmfEingangsjahrgaenge(r.Context())
		if err != nil {
			return apierrors.Internal("Einstellungen laden", err)
		}
		termine, err := repo.ListLmfTermine(r.Context(), ab, repository.LmfListenFilter{})
		if err != nil {
			return apierrors.Internal("LMF-Plan laden", err)
		}
		antwort := LmfPlanAntwort{Termine: termine, OhneRueckgabeTermin: []string{}, Eingangsjahrgaenge: eingang}
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

// LmfArtTitel ist die Überschrift je Art — dieselben Worte wie ARTEN in
// lmfplanDienst.js. „Rückgabe" und „Ausgabe" allein waren unklar (Peter, 06.09.2026):
// Vor den Ferien tauschen die Klassen (alte ab, neue direkt mit), nach den Ferien
// bekommen nur die neu gebildeten Klassen ihre Bücher.
func LmfArtTitel(art string) string {
	if art == repository.LmfTerminAusgabe {
		return "Bücherausgabe nach den Sommerferien"
	}
	return "Büchertausch vor den Sommerferien"
}

// LmfArtErklaerung ist der eine Satz unter der Überschrift, mit den Eingangsjahrgängen.
func LmfArtErklaerung(art string, eingang []int) string {
	if art == repository.LmfTerminAusgabe {
		// Ohne bekannte Jahrgänge KEIN leeres Klammerpaar („(Jahrgang )"): Der Satz nennt
		// dann nur die Regel. Beide Seiten müssen das gleich halten — TestLmfTexte prüft es.
		if len(eingang) == 0 {
			return "Nur die neu gebildeten Klassen bekommen ihre Schulbücher."
		}
		return "Nur die neu gebildeten Klassen (Jahrgang " + jahrgaengeText(eingang) + ") bekommen ihre Schulbücher."
	}
	return "Alle Klassen geben die alten Schulbücher ab und bekommen direkt die neuen. " +
		"„Nur Rückgabe“: Abschlussklassen und Klassen, die zum neuen Schuljahr neu gebildet werden."
}

// jahrgaengeText: „5 und 7", „5, 7 und 11".
func jahrgaengeText(eingang []int) string {
	teile := make([]string, 0, len(eingang))
	for _, j := range eingang {
		teile = append(teile, fmt.Sprint(j))
	}
	if len(teile) <= 1 {
		return strings.Join(teile, "")
	}
	return strings.Join(teile[:len(teile)-1], ", ") + " und " + teile[len(teile)-1]
}

// lmfPlanAbschnitte gruppiert die Termine für das PDF: erst der Tausch vor den Ferien,
// dann die Ausgabe danach, je Abschnitt NACH ZEITPUNKT — Datum, Stunde, und bei
// Gleichstand die Position im Plan.
//
// Das ist bewusst NICHT die Reihenfolge der Zeilen im Planer (der Kommentar behauptete
// das bis zum Rasterdurchgang am 06.09.2026): Ein fester Platz kann eine Zeile nach
// hinten schieben, ohne ihre Nummer zu ändern — im Planer bleibt sie Zeile 2, im Portal
// und im PDF steht sie dort, wo sie stattfindet. Das Kollegium liest einen Fahrplan, die
// Bibliothek arbeitet eine Reihenfolge ab. Die Position als letztes Kriterium hält die
// Ausgabe stabil; vorher entschied bei zwei gleichen Plätzen die Zeilen-UUID, und die ist
// nach jedem Speichern neu.
//
// „Nur Rückgabe" steht, wo die Bibliothek es hingeschrieben hat: im Vermerk.
func lmfPlanAbschnitte(termine []repository.LmfTermin, eingang []int) ([]pdf.LmfPlanAbschnitt, error) {
	abschnitte := []pdf.LmfPlanAbschnitt{
		{Titel: strings.ToUpper(LmfArtTitel(repository.LmfTerminRueckgabe)), Untertitel: LmfArtErklaerung(repository.LmfTerminRueckgabe, eingang)},
		{Titel: strings.ToUpper(LmfArtTitel(repository.LmfTerminAusgabe)), Untertitel: LmfArtErklaerung(repository.LmfTerminAusgabe, eingang)},
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
// mitEntwuerfen = true ist die Planer-Route (edit_books): auch der unveröffentlichte
// Entwurf, damit er zur Abnahme an die Schulleitung gehen kann.
// @Summary      LMF-Plan als PDF
// @Tags         lernmittel
// @Produce      application/pdf
// @Router       /lmf-termine/pdf [get]
func (s *Server) GetLmfPlanPDFHandler(mitEntwuerfen bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repo := repository.NewLmfTerminRepository(s.DB.Pool)
		eingang, err := s.lmfEingangsjahrgaenge(r.Context())
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		termine, err := repo.ListLmfTermine(r.Context(), s.lmfPlanAb(r),
			repository.LmfListenFilter{MitEntwuerfen: mitEntwuerfen})
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if len(termine) == 0 {
			apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("der LMF-Plan ist leer"))
			return
		}
		abschnitte, err := lmfPlanAbschnitte(termine, eingang)
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
