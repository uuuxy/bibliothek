package lernmittel

// Entwurf und Veröffentlichung des LMF-Plans. Den Plan nimmt die Schulleitung ab, bevor er
// gilt: Speichern legt deshalb einen Entwurf an, zentral und auf jedem Arbeitsplatz gleich, aber
// für Portal, PDF des Kollegiums und Kopplung der Fristen unsichtbar. Erst POST
// …/veroeffentlichen stempelt ihn und setzt bei einem Rückgabe-Plan die Fristen; danach gilt
// jede weitere Speicherung sofort. Einen zweiten Entwurf neben einem veröffentlichten Plan gibt
// es nicht, sonst wüsste niemand, welcher gilt.
//
// Dazu die Eingangsjahrgänge (Einstellung lmf_eingangsjahrgaenge, Vorgabe „5, 7"):
// Nach den Ferien bekommen nur die neu gebildeten Klassen Bücher; vor den Ferien geben
// die Klassen, die neu gebildet werden (die 6er → 7H/7R/7G), und die Abschlussklassen
// nur zurück, alle anderen tauschen.

import (
	"context"
	"errors"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/db"
	"bibliothek/internal/service"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// lmfEingangsjahrgaenge liest die Einstellung; ohne lesbaren Wert gilt die Vorgabe.
func (t *Tueren) lmfEingangsjahrgaenge(ctx context.Context) ([]int, error) {
	einstellungen, err := repository.NewEinstellungsAbfragen(t.DB.Pool).Lade(ctx)
	if err != nil {
		return nil, err
	}
	return repository.EingangsjahrgaengeAus(einstellungen.LMFEingangsjahrgaenge), nil
}

// lmfPlanNurRueckgabe nennt aus allen Klassen, die der Planer zeigt (Vokabular, Zeilen,
// Vorschlag, Auslassungen), die, die vor den Ferien nur abgeben; leer beim Ausgabe-Plan. Aus
// ihnen belegt service.BelegeVermerkNurRueckgabeVor den Vermerk des Vorschlags vor.
func (t *Tueren) lmfPlanNurRueckgabe(ctx context.Context, abfragen *repository.LMFAbfragen, art string, a lmfPlanStandAntwort, eingang []int) ([]string, error) {
	if art != repository.LMFTerminRueckgabe {
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
	return abfragen.LadeKlassenNurRueckgabe(ctx, namen, eingang)
}

// LMFPlanVeroeffentlichen stempelt den neuesten Plan der Art als veröffentlicht
// und lässt bei einem Rückgabe-Plan die Fristen der Klassen dem Plan folgen. Idempotent:
// ein schon veröffentlichter Plan behält seinen Stempel, Fristen werden dann nicht
// erneut angefasst (das tut jede Speicherung eines veröffentlichten Plans selbst).
// @Summary      LMF-Plan veröffentlichen
// @Tags         lernmittel
// @Produce      json
// @Success      200  {object}  lmfPlanSpeicherAntwort
// @Router       /lmf-plan/{art}/veroeffentlichen [post]
func (t *Tueren) LMFPlanVeroeffentlichen() http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		antwort, gestempelt, err := t.veroeffentlicheLMFPlan(r)
		if err != nil {
			return err
		}
		// Ein schon veröffentlichter Plan ändert sich hier nicht und meldet nichts.
		if gestempelt {
			t.meldeLMFPlanGeaendert()
		}
		tuer.AntworteJSON(w, http.StatusOK, antwort)
		return nil
	})
}

// veroeffentlicheLMFPlan stempelt den neuesten Plan der Art und lässt die Fristen folgen.
// gestempelt ist false, wenn der Plan schon veröffentlicht war.
func (t *Tueren) veroeffentlicheLMFPlan(r *http.Request) (lmfPlanSpeicherAntwort, bool, error) {
	var keine lmfPlanSpeicherAntwort
	art, err := leseLMFPlanArt(r)
	if err != nil {
		return keine, false, err
	}
	abfragen := repository.NewLMFAbfragen(t.DB.Pool)
	stand, err := abfragen.LadeNeuestenPlan(r.Context(), art)
	if errors.Is(err, pgx.ErrNoRows) {
		return keine, false, apierrors.NotFound("Kein Plan vorhanden", pgx.ErrNoRows)
	}
	if err != nil {
		return keine, false, apierrors.Internal("LMF-Plan laden", err)
	}
	antwort := lmfPlanSpeicherAntwort{LMFPlanStand: stand, Ausfaelle: []lmfPlanAusfall{}}
	if stand.Plan.VeroeffentlichtAm != nil {
		return antwort, false, nil
	}
	// Stempel und Fristen stehen in einer Transaktion, wie beim Speichern.
	tx, err := t.DB.Pool.Begin(r.Context())
	if err != nil {
		return keine, false, apierrors.Internal("Transaktion", err)
	}
	defer db.Verwerfe(r.Context(), tx)
	if antwort.LMFPlanStand, err = abfragen.VeroeffentlichePlan(r.Context(), tx, stand.Plan.ID, t.Jetzt()); err != nil {
		return keine, false, apierrors.Internal("LMF-Plan veröffentlichen", err)
	}
	if antwort.FristenAngepasst, err = service.KoppleLMFPlanFristen(r.Context(), abfragen, tx, art, nil, antwort.Zeilen); err != nil {
		return keine, false, apierrors.Internal("Fristen koppeln", err)
	}
	if err := tx.Commit(r.Context()); err != nil {
		return keine, false, apierrors.Internal("LMF-Plan veröffentlichen", err)
	}
	t.protokolliereLMFPlan(r, aktionLMFPlanVeroeffentlicht, art, stand.Plan.ID, antwort.FristenAngepasst)
	return antwort, true, nil
}
