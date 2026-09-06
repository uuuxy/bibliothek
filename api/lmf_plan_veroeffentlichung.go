package api

// lmf_plan_veroeffentlichung.go — Entwurf und Veröffentlichung des LMF-Plans (Migration
// 100). Peter, 06.09.2026: „Stille Vorbereitung — der Plan nimmt die Schulleitung immer
// erst ab." Speichern legt deshalb einen ENTWURF an: zentral, auf jedem PC gleich, aber
// für Portal, Kollegiums-PDF und Frist-Kopplung unsichtbar. Erst POST …/veroeffentlichen
// stempelt ihn und setzt bei einem Rückgabe-Plan die Fristen; danach gilt jede weitere
// Speicherung sofort (die Korrektur-Mail von früher). Einen zweiten Entwurf neben einem
// veröffentlichten Plan gibt es bewusst nicht — dann wüsste niemand, welcher gilt.
//
// Dazu die Eingangsjahrgänge (Einstellung lmf_eingangsjahrgaenge, Vorgabe „5, 7"):
// Nach den Ferien bekommen nur die neu gebildeten Klassen Bücher; vor den Ferien geben
// die Klassen, die neu gebildet werden (die 6er → 7H/7R/7G), und die Abschlussklassen
// NUR zurück, alle anderen tauschen.

import (
	"context"
	"errors"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// lmfEingangsjahrgaenge liest die Einstellung; ohne lesbaren Wert gilt die Vorgabe.
func (s *Server) lmfEingangsjahrgaenge(ctx context.Context) ([]int, error) {
	einstellungen, err := repository.NewSystemSettingsRepository(s.DB.Pool).GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	return repository.EingangsjahrgaengeAus(einstellungen.LmfEingangsjahrgaenge), nil
}

// enthaeltJahrgang: ist j einer der Eingangsjahrgänge?
func enthaeltJahrgang(eingang []int, j int) bool {
	for _, e := range eingang {
		if e == j {
			return true
		}
	}
	return false
}

// lmfPlanNurRueckgabe nennt aus allen Klassen, die der Planer zeigt (Vokabular, Zeilen,
// Vorschlag, Auslassungen), die, die vor den Ferien nur abgeben — leer beim Ausgabe-Plan.
// Seit 06.09.2026 nur noch die Quelle der VORBELEGUNG des Vermerks (vermerkNurRueckgabe).
func (s *Server) lmfPlanNurRueckgabe(ctx context.Context, repo *repository.LmfTerminRepository, art string, a LmfPlanStandAntwort, eingang []int) ([]string, error) {
	if art != repository.LmfTerminRueckgabe {
		return []string{}, nil
	}
	gesehen := map[string]bool{}
	var namen []string
	sammle := func(klassen []string) {
		for _, k := range klassen {
			if !gesehen[k] {
				gesehen[k] = true
				namen = append(namen, k)
			}
		}
	}
	sammle(a.Klassen)
	sammle(a.Ausgelassen)
	for _, z := range a.Zeilen {
		sammle(z.Klassen)
	}
	if a.Vorschlag != nil {
		sammle(a.Vorschlag.Ausgelassen)
		for _, z := range a.Vorschlag.Zeilen {
			sammle(z.Klassen)
		}
	}
	return repo.KlassenNurRueckgabe(ctx, namen, eingang)
}

// PostLmfPlanVeroeffentlichenHandler stempelt den neuesten Plan der Art als veröffentlicht
// und lässt bei einem Rückgabe-Plan die Fristen der Klassen dem Plan folgen. Idempotent:
// ein schon veröffentlichter Plan behält seinen Stempel, Fristen werden dann nicht
// erneut angefasst (das tut jede Speicherung eines veröffentlichten Plans selbst).
// @Summary      LMF-Plan veröffentlichen
// @Tags         lernmittel
// @Produce      json
// @Success      200  {object}  LmfPlanSpeicherAntwort
// @Router       /lmf-plan/{art}/veroeffentlichen [post]
func (s *Server) PostLmfPlanVeroeffentlichenHandler() http.HandlerFunc {
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
		antwort := LmfPlanSpeicherAntwort{LmfPlanStand: st, Ausfaelle: []LmfPlanAusfall{}}
		if st.Plan.VeroeffentlichtAm != nil {
			RespondJSON(w, http.StatusOK, antwort)
			return nil
		}
		if antwort.LmfPlanStand, err = repo.VeroeffentlicheLmfPlan(r.Context(), st.Plan.ID, s.jetzt()); err != nil {
			return apierrors.Internal("LMF-Plan veröffentlichen", err)
		}
		if antwort.FristenAngepasst, err = s.koppleLmfPlanFristen(r.Context(), art, nil, antwort.Zeilen); err != nil {
			return apierrors.Internal("Fristen koppeln", err)
		}
		s.auditiereLmfPlan(r, auditLmfPlanVeroeffentlicht, art, st.Plan.ID, antwort.FristenAngepasst)
		RespondJSON(w, http.StatusOK, antwort)
		return nil
	})
}
