package bestellung

import (
	"errors"
	"log/slog"
	"net/http"

	"bibliothek/internal/api/tuer"
	"bibliothek/internal/apierrors"
	"bibliothek/internal/auth"
	"bibliothek/internal/pkg/betrag"
	"bibliothek/internal/pkg/clientip"
	"bibliothek/internal/repository"
	"bibliothek/internal/service"

	"github.com/jackc/pgx/v5"
)

// Den Preis einer Bestellposition ändern: Der Preis beim Bestellen stammt oft vom Fachbereich
// und ist manchmal veraltet, und der Händler rechnet am Ende anders ab, mit einem Rabatt oder
// einem neuen Listenpreis. Die Rechnung kommt auf Papier, also trägt die Bibliothek den Preis von
// Hand nach, auch lange nach der Bestellung.

// positionspreisAnfrage ist der neue Einzelpreis in Euro.
type positionspreisAnfrage struct {
	Einzelpreis *betrag.Cent `json:"einzelpreis"`
}

const aktionBestellpreisGeaendert = "BESTELLPREIS_GEAENDERT"

// PositionspreisAendern liefert die Tür PUT /api/bestellungen/{id}/positionen/{position_id}/einzelpreis.
func (t *Tueren) PositionspreisAendern() http.HandlerFunc {
	return t.aenderePositionspreis
}

func (t *Tueren) aenderePositionspreis(w http.ResponseWriter, r *http.Request) {
	bestellungID, positionID := r.PathValue("id"), r.PathValue("position_id")
	var req positionspreisAnfrage
	if !tuer.LeseAnfrage(w, r, &req) {
		return
	}
	if req.Einzelpreis == nil {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Bitte einen Preis angeben", nil)
		return
	}
	preis := *req.Einzelpreis
	if preis < 0 || preis > service.HoechsterEinzelpreis {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, meldungPreisUngueltig, nil)
		return
	}

	titel, alt, exemplare, err := repository.AenderePositionspreis(r.Context(), t.DB.Pool, bestellungID, positionID, preis)
	if errors.Is(err, pgx.ErrNoRows) {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusNotFound, "Diese Position gehört nicht zu der Bestellung", nil)
		return
	}
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return
	}
	if alt != preis {
		t.protokollierePositionspreis(r, map[string]any{
			"bestellung_id": bestellungID, "position_id": positionID, "titel": titel, "von": alt, "nach": preis,
			"exemplare": exemplare,
		})
	}
	tuer.AntworteJSON(w, http.StatusOK, map[string]any{"einzelpreis": preis, "von": alt})
}

// protokollierePositionspreis schreibt die Änderung ins Protokoll der Verwaltung, wie die
// Korrektur des Topfs; ohne Anmeldung gibt es nichts zu schreiben.
func (t *Tueren) protokollierePositionspreis(r *http.Request, details map[string]any) {
	claims, ok := auth.ClaimsAusKontext(r.Context())
	if !ok {
		return
	}
	if err := repository.NewProtokollAbfragen(t.DB.Pool).
		ProtokolliereAdminAktion(r.Context(), claims.KontoID, aktionBestellpreisGeaendert, clientip.AusAnfrage(r), details); err != nil {
		slog.Error("Audit fehlgeschlagen", "aktion", aktionBestellpreisGeaendert, "fehler", err)
	}
}
