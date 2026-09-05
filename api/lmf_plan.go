package api

// lmf_plan.go — der LMF-Plan als Reihenfolge (Peter, 05.09.2026 abends, nach dem
// echten Plan der Schule): Rahmen (erster Tag, Startstunde, Stunden je Tag) plus eine
// Reihenfolge von Zeilen (Klassen, Vermerk); Datum und Stunde jeder Zeile rechnet der
// Server (pkg/lmfplan) über die Schultage — Wochenende und Ferien fallen aus. Die
// Vorschau im Planer ist DERSELBE Aufruf mit "vorschau": true, damit es keinen
// JavaScript-Zwilling der Verteilung gibt. Ein Plan je Art und Schuljahr; GET liefert
// den neuesten der Art und, wenn er vorbei ist, dieselbe Reihenfolge als Vorschlag für
// den nächsten (Klassennamen bleiben Jahr für Jahr gleich — die Versetzung verschiebt
// Schüler, nicht Namen). Ohne Vorjahr kommt der Vorschlag aus der Regel: Abschluss-
// klassen zuerst, dann Jahrgang absteigend; die Oberstufe steht unten („ausgelassen",
// sie organisiert sich an dieser Schule selbst).

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"bibliothek/apierrors"
	"bibliothek/pkg/lmfplan"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// LmfPlanVorschlag ist die Reihenfolge, mit der ein neuer Plan beginnt.
type LmfPlanVorschlag struct {
	// Quelle: "vorjahr" (der letzte Plan der Art) oder "regel" (Abschluss zuerst).
	Quelle      string                    `json:"quelle"`
	Zeilen      []repository.LmfPlanZeile `json:"zeilen"`
	Ausgelassen []string                  `json:"ausgelassen"`
}

// LmfPlanStandAntwort ist die Antwort von GET /api/lmf-plan/{art}.
type LmfPlanStandAntwort struct {
	// Plan ist der neueste Plan der Art; nil, wenn es noch keinen gibt.
	Plan        *repository.LmfPlan       `json:"plan"`
	Zeilen      []repository.LmfPlanZeile `json:"zeilen"`
	Ausgelassen []string                  `json:"ausgelassen"`
	// Vorbei: der letzte Termin des Plans liegt hinter heute — der Planer bietet dann
	// den nächsten Plan an, mit derselben Reihenfolge als Vorschlag.
	Vorbei bool `json:"vorbei"`
	// Vorschlag ist gesetzt, wenn es keinen laufenden Plan gibt (kein Plan oder vorbei).
	Vorschlag *LmfPlanVorschlag `json:"vorschlag,omitempty"`
	// Klassen: alle Klassen des Vokabulars, für die Auswahl im Planer.
	Klassen []string `json:"klassen"`
}

// lmfPlanArt liest {art} aus dem Pfad und prüft sie.
func lmfPlanArt(r *http.Request) (string, error) {
	art := r.PathValue("art")
	if art != repository.LmfTerminRueckgabe && art != repository.LmfTerminAusgabe {
		return "", errors.New("art muss 'rueckgabe' oder 'ausgabe' sein")
	}
	return art, nil
}

// GetLmfPlanHandler liefert den neuesten Plan der Art und ggf. den Vorschlag.
// @Summary      LMF-Plan (Reihenfolge) lesen
// @Tags         lernmittel
// @Produce      json
// @Success      200  {object}  LmfPlanStandAntwort
// @Router       /lmf-plan/{art} [get]
func (s *Server) GetLmfPlanHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		art, err := lmfPlanArt(r)
		if err != nil {
			return apierrors.BadRequest(err.Error(), err)
		}
		repo := repository.NewLmfTerminRepository(s.DB.Pool)
		antwort := LmfPlanStandAntwort{Zeilen: []repository.LmfPlanZeile{}, Ausgelassen: []string{}}
		stand, err := repo.NeuesterLmfPlan(r.Context(), art)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			// kein Plan
		case err != nil:
			return apierrors.Internal("LMF-Plan laden", err)
		default:
			plan := stand.Plan
			antwort.Plan, antwort.Zeilen, antwort.Ausgelassen = &plan, stand.Zeilen, stand.Ausgelassen
			antwort.Vorbei = lmfPlanVorbei(stand, s.jetzt())
		}
		if antwort.Plan == nil || antwort.Vorbei {
			if antwort.Vorschlag, err = s.lmfPlanVorschlag(r.Context(), repo, antwort.Plan != nil, stand); err != nil {
				return apierrors.Internal("Vorschlag bauen", err)
			}
		}
		if antwort.Klassen, err = repository.NewStudentRepository(s.DB.Pool).GetDistinctClasses(r.Context()); err != nil {
			return apierrors.Internal("Klassen laden", err)
		}
		RespondJSON(w, http.StatusOK, antwort)
		return nil
	})
}

// lmfPlanVorbei: Der letzte Platz des Plans (oder der erste Tag, wenn er leer ist) liegt
// vor dem heutigen Kalendertag der Schule.
func lmfPlanVorbei(st repository.LmfPlanStand, jetzt time.Time) bool {
	letzter := st.Plan.ErsterTag
	for _, z := range st.Zeilen {
		if z.Datum > letzter {
			letzter = z.Datum
		}
	}
	heute := jetzt.In(schulzeit.Zone()).Format("2006-01-02")
	return letzter < heute
}

// lmfPlanVorschlag baut die Start-Reihenfolge: das Vorjahr, wenn es eines gibt, ergänzt
// um Klassen, die inzwischen neu sind; sonst die Regel. Klassen ohne Schüler bleiben aus
// dem Vorjahr erhalten (ein „7G6" kann wiederkommen) — der Planer entfernt sie mit
// einem Klick.
func (s *Server) lmfPlanVorschlag(ctx context.Context, repo *repository.LmfTerminRepository, vorjahr bool, st repository.LmfPlanStand) (*LmfPlanVorschlag, error) {
	klassen, err := repo.KlassenMitSchuelern(ctx)
	if err != nil {
		return nil, err
	}
	v := &LmfPlanVorschlag{Quelle: "regel", Zeilen: []repository.LmfPlanZeile{}, Ausgelassen: []string{}}
	bekannt := map[string]bool{}
	if vorjahr {
		v.Quelle = "vorjahr"
		for _, z := range st.Zeilen {
			v.Zeilen = append(v.Zeilen, repository.LmfPlanZeile{Klassen: z.Klassen, Vermerk: z.Vermerk})
			for _, k := range z.Klassen {
				bekannt[repository.KlassenSchluessel(k)] = true
			}
		}
		v.Ausgelassen = append(v.Ausgelassen, st.Ausgelassen...)
		for _, k := range st.Ausgelassen {
			bekannt[repository.KlassenSchluessel(k)] = true
		}
	}
	for _, k := range klassen {
		if bekannt[repository.KlassenSchluessel(k.Name)] {
			continue
		}
		if k.Oberstufe {
			v.Ausgelassen = append(v.Ausgelassen, k.Name)
			continue
		}
		v.Zeilen = append(v.Zeilen, repository.LmfPlanZeile{Klassen: []string{k.Name}})
	}
	return v, nil
}

// lmfPlanRequest ist der Körper von PUT /api/lmf-plan/{art}.
type lmfPlanRequest struct {
	ErsterTag    string `json:"erster_tag"`
	Startstunde  int    `json:"startstunde"`
	StundenJeTag int    `json:"stunden_je_tag"`
	Zeilen       []struct {
		Klassen []string `json:"klassen"`
		Vermerk string   `json:"vermerk"`
	} `json:"zeilen"`
	Ausgelassen []string `json:"ausgelassen"`
	// Vorschau: nur rechnen, nichts schreiben — die Verteilung für den Planer.
	Vorschau bool `json:"vorschau"`
}

// LmfPlanSpeicherAntwort ist die Antwort von PUT: der Stand und die Zahl der Ausleihen, deren
// Frist dem Plan gefolgt ist (0 bei Vorschau und bei Ausgabe-Plänen).
type LmfPlanSpeicherAntwort struct {
	repository.LmfPlanStand
	Vorschau         bool  `json:"vorschau"`
	FristenAngepasst int64 `json:"fristen_angepasst"`
}

const lmfPlanMaxZeilen = 400

// pruefeLmfPlan validiert fachlich und liefert Rahmen und Zeilen.
func pruefeLmfPlan(art string, req lmfPlanRequest) (repository.LmfPlan, []repository.LmfPlanZeile, []string, error) {
	plan := repository.LmfPlan{Art: art, ErsterTag: strings.TrimSpace(req.ErsterTag),
		Startstunde: req.Startstunde, StundenJeTag: req.StundenJeTag}
	if _, err := time.Parse("2006-01-02", plan.ErsterTag); err != nil {
		return plan, nil, nil, errors.New("erster_tag muss als JJJJ-MM-TT angegeben sein")
	}
	if plan.StundenJeTag < 1 || plan.StundenJeTag > 12 {
		return plan, nil, nil, errors.New("stunden_je_tag muss zwischen 1 und 12 liegen")
	}
	if plan.Startstunde < 1 || plan.Startstunde > plan.StundenJeTag {
		return plan, nil, nil, errors.New("startstunde muss zwischen 1 und stunden_je_tag liegen")
	}
	if len(req.Zeilen) > lmfPlanMaxZeilen {
		return plan, nil, nil, errors.New("zu viele Zeilen")
	}
	zeilen := make([]repository.LmfPlanZeile, 0, len(req.Zeilen))
	for i, z := range req.Zeilen {
		zeile := repository.LmfPlanZeile{Vermerk: strings.TrimSpace(z.Vermerk)}
		for _, k := range z.Klassen {
			if k = strings.TrimSpace(k); k != "" {
				zeile.Klassen = append(zeile.Klassen, k)
			}
		}
		if len(zeile.Klassen) == 0 && zeile.Vermerk == "" {
			return plan, nil, nil, fmt.Errorf("zeile %d hat weder Klasse noch Vermerk", i+1)
		}
		zeilen = append(zeilen, zeile)
	}
	var ausgelassen []string
	for _, k := range req.Ausgelassen {
		if k = strings.TrimSpace(k); k != "" {
			ausgelassen = append(ausgelassen, k)
		}
	}
	return plan, zeilen, ausgelassen, nil
}

// PutLmfPlanHandler rechnet die Verteilung und speichert den Plan (oder zeigt sie nur).
// @Summary      LMF-Plan (Reihenfolge) speichern oder als Vorschau rechnen
// @Tags         lernmittel
// @Accept       json
// @Produce      json
// @Success      200  {object}  LmfPlanSpeicherAntwort
// @Router       /lmf-plan/{art} [put]
func (s *Server) PutLmfPlanHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		art, err := lmfPlanArt(r)
		if err != nil {
			return apierrors.BadRequest(err.Error(), err)
		}
		var req lmfPlanRequest
		if !DecodeAndValidate(w, r, &req) {
			return nil
		}
		plan, zeilen, ausgelassen, err := pruefeLmfPlan(art, req)
		if err != nil {
			return apierrors.BadRequest(err.Error(), err)
		}
		repo := repository.NewLmfTerminRepository(s.DB.Pool)
		plaetze, err := s.verteileLmfPlan(r.Context(), repo, plan, len(zeilen))
		if err != nil {
			return apierrors.Internal("Verteilung rechnen", err)
		}
		if req.Vorschau {
			for i := range zeilen {
				zeilen[i].Position = i + 1
				zeilen[i].Datum = plaetze[i].Datum.Format("2006-01-02")
				zeilen[i].Stunde = plaetze[i].Stunde
			}
			RespondJSON(w, http.StatusOK, LmfPlanSpeicherAntwort{Vorschau: true,
				LmfPlanStand: repository.LmfPlanStand{Plan: plan, Zeilen: zeilen, Ausgelassen: ausgelassen}})
			return nil
		}
		// Den alten Stand DIESES Schuljahres lesen: Klassen, die ihren Termin verlieren,
		// kehren zum Stichtag zurück (lmf_termine_frist.go).
		alt, err := s.lmfPlanZeilenVorher(r.Context(), repo, art, plan.ErsterTag)
		if err != nil {
			return apierrors.Internal("alten Plan lesen", err)
		}
		stand, err := repo.SaveLmfPlan(r.Context(), plan, zeilen, plaetze, ausgelassen)
		if err != nil {
			return apierrors.Internal("LMF-Plan speichern", err)
		}
		angepasst, err := s.koppleLmfPlanFristen(r.Context(), art, alt, stand.Zeilen)
		if err != nil {
			return apierrors.Internal("Fristen koppeln", err)
		}
		RespondJSON(w, http.StatusOK, LmfPlanSpeicherAntwort{LmfPlanStand: stand, FristenAngepasst: angepasst})
		return nil
	})
}

// verteileLmfPlan rechnet die Plätze über die Schultage; Ferien aus der Datenbank.
func (s *Server) verteileLmfPlan(ctx context.Context, repo *repository.LmfTerminRepository, plan repository.LmfPlan, n int) ([]lmfplan.Platz, error) {
	ersterTag, err := planTag(plan.ErsterTag)
	if err != nil {
		return nil, err
	}
	// Großzügiges Fenster: 400 Zeilen bei einer Stunde je Tag sind 80 Schulwochen.
	frei, err := repo.FreieTage(ctx, ersterTag, ersterTag.AddDate(2, 0, 0))
	if err != nil {
		return nil, err
	}
	r := lmfplan.Rahmen{ErsterTag: ersterTag, Startstunde: plan.Startstunde, StundenJeTag: plan.StundenJeTag}
	return lmfplan.Verteile(r, n, lmfplan.Schultage(frei)), nil
}

// lmfPlanZeilenVorher liest die Zeilen des Plans, den das Speichern gleich ersetzt —
// der Plan derselben Art im Schuljahr des ersten Tages. Leer, wenn es keinen gibt.
func (s *Server) lmfPlanZeilenVorher(ctx context.Context, repo *repository.LmfTerminRepository, art, ersterTag string) ([]repository.LmfPlanZeile, error) {
	st, err := repo.NeuesterLmfPlan(ctx, art)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	tag, err := planTag(ersterTag)
	if err != nil {
		return nil, err
	}
	if st.Plan.SchuljahrBeginn != repository.SchuljahrBeginn(tag).Format("2006-01-02") {
		// Anderes Schuljahr: das Speichern legt einen neuen Plan an, der alte bleibt.
		return nil, nil
	}
	return st.Zeilen, nil
}

// DeleteLmfPlanHandler verwirft den neuesten Plan der Art; die Fristen seiner Klassen
// gehen an den Stichtag zurück (nur die, die auf den Terminen lagen).
// @Summary      LMF-Plan (Reihenfolge) verwerfen
// @Tags         lernmittel
// @Produce      json
// @Success      200  {object}  map[string]int64
// @Router       /lmf-plan/{art} [delete]
func (s *Server) DeleteLmfPlanHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		art, err := lmfPlanArt(r)
		if err != nil {
			return apierrors.BadRequest(err.Error(), err)
		}
		repo := repository.NewLmfTerminRepository(s.DB.Pool)
		st, err := repo.NeuesterLmfPlan(r.Context(), art)
		if errors.Is(err, pgx.ErrNoRows) {
			return apierrors.NotFound("kein Plan vorhanden", pgx.ErrNoRows)
		}
		if err != nil {
			return apierrors.Internal("LMF-Plan laden", err)
		}
		if _, err := repo.DeleteLmfPlan(r.Context(), st.Plan.ID); err != nil {
			return apierrors.Internal("LMF-Plan löschen", err)
		}
		angepasst, err := s.koppleLmfPlanFristen(r.Context(), art, st.Zeilen, nil)
		if err != nil {
			return apierrors.Internal("Fristen koppeln", err)
		}
		RespondJSON(w, http.StatusOK, map[string]int64{"fristen_angepasst": angepasst})
		return nil
	})
}
