package api

// lmf_termine.go — der LMF-Plan: Rückgabe- und Ausgabetermine je Klasse (Register,
// Entscheidung 3, 05.09.2026). Die Bibliothek pflegt die Tabelle, das Kollegium liest
// sie im Portal (für alle gleich, keine Personalisierung), das PDF sieht aus wie die
// bisherige Excel-Liste. Lesen verlangt nur eine Sitzung (Stufe 0: Daten und Klassen,
// kein Schülerbezug), Schreiben edit_books wie die übrige Lernmittel-Pflege.

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

	"github.com/jackc/pgx/v5"
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

// LmfTerminAntwort ist die Antwort von POST/PUT: die gespeicherte Zeile und die Zahl der
// Ausleihen, deren Frist dem Plan gefolgt ist (lmf_termine_frist.go).
type LmfTerminAntwort struct {
	repository.LmfTermin
	FristenAngepasst int64 `json:"fristen_angepasst"`
}

// lmfTerminRequest ist der Körper von POST und PUT.
type lmfTerminRequest struct {
	Datum   string   `json:"datum"`
	Stunde  int      `json:"stunde"`
	Art     string   `json:"art"`
	Klassen []string `json:"klassen"`
	Vermerk string   `json:"vermerk"`
}

// pruefeLmfTermin validiert fachlich: gültiges Datum, Stunde 1–12, bekannte Art, und
// eine Zeile ohne Klasse braucht einen Vermerk („Bücher setzen") — sonst stünde eine
// leere Zeile im Plan.
func pruefeLmfTermin(req lmfTerminRequest) (repository.LmfTermin, error) {
	t := repository.LmfTermin{Datum: strings.TrimSpace(req.Datum), Stunde: req.Stunde,
		Art: strings.TrimSpace(req.Art), Vermerk: strings.TrimSpace(req.Vermerk)}
	if _, err := time.Parse("2006-01-02", t.Datum); err != nil {
		return t, errors.New("datum muss als JJJJ-MM-TT angegeben sein")
	}
	if t.Stunde < 1 || t.Stunde > 12 {
		return t, errors.New("stunde muss zwischen 1 und 12 liegen")
	}
	if t.Art != repository.LmfTerminRueckgabe && t.Art != repository.LmfTerminAusgabe {
		return t, errors.New("art muss 'rueckgabe' oder 'ausgabe' sein")
	}
	for _, k := range req.Klassen {
		if k = strings.TrimSpace(k); k != "" {
			t.Klassen = append(t.Klassen, k)
		}
	}
	if len(t.Klassen) == 0 && t.Vermerk == "" {
		return t, errors.New("ein Termin ohne Klasse braucht einen Vermerk")
	}
	return t, nil
}

// CreateLmfTerminHandler legt einen Termin an.
// @Summary      LMF-Termin anlegen
// @Tags         lernmittel
// @Accept       json
// @Produce      json
// @Success      201  {object}  LmfTerminAntwort
// @Router       /lmf-termine [post]
func (s *Server) CreateLmfTerminHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.speichereLmfTermin(w, r, "")
	}
}

// UpdateLmfTerminHandler schreibt einen Termin um.
// @Summary      LMF-Termin ändern
// @Tags         lernmittel
// @Accept       json
// @Produce      json
// @Success      200  {object}  LmfTerminAntwort
// @Router       /lmf-termine/{id} [put]
func (s *Server) UpdateLmfTerminHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.speichereLmfTermin(w, r, r.PathValue("id"))
	}
}

func (s *Server) speichereLmfTermin(w http.ResponseWriter, r *http.Request, id string) {
	var req lmfTerminRequest
	if !DecodeAndValidate(w, r, &req) {
		return
	}
	t, err := pruefeLmfTermin(req)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusBadRequest, err)
		return
	}
	t.ID = id
	repo := repository.NewLmfTerminRepository(s.DB.Pool)
	var alt *repository.LmfTermin
	if id != "" {
		// Den alten Stand lesen, bevor er überschrieben wird: 404 statt eines stillen
		// Nichts, und für die Frist-Kopplung die Klassen, die den Termin gleich verlieren.
		vorher, err := repo.GetLmfTermin(r.Context(), id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("termin nicht gefunden"))
			} else {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			}
			return
		}
		alt = &vorher
	}
	gespeichert, err := repo.SaveLmfTermin(r.Context(), t)
	if errors.Is(err, pgx.ErrNoRows) {
		apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("termin nicht gefunden"))
		return
	}
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return
	}
	angepasst, err := s.koppleLmfFristen(r.Context(), alt, &gespeichert)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return
	}
	status := http.StatusOK
	if id == "" {
		status = http.StatusCreated
	}
	RespondJSON(w, status, LmfTerminAntwort{LmfTermin: gespeichert, FristenAngepasst: angepasst})
}

// DeleteLmfTerminHandler entfernt einen Termin; die Fristen seiner Klassen gehen an den
// Stichtag zurück (nur die, die auf dem Termin lagen).
// @Summary      LMF-Termin löschen
// @Tags         lernmittel
// @Produce      json
// @Success      200  {object}  map[string]int64
// @Router       /lmf-termine/{id} [delete]
func (s *Server) DeleteLmfTerminHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repo := repository.NewLmfTerminRepository(s.DB.Pool)
		alt, err := repo.GetLmfTermin(r.Context(), r.PathValue("id"))
		if errors.Is(err, pgx.ErrNoRows) {
			apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("termin nicht gefunden"))
			return
		}
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if _, err := repo.DeleteLmfTermin(r.Context(), alt.ID); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		angepasst, err := s.koppleLmfFristen(r.Context(), &alt, nil)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		RespondJSON(w, http.StatusOK, map[string]int64{"fristen_angepasst": angepasst})
	}
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
