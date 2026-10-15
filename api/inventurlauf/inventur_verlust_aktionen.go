package inventurlauf

// Die zwei Knöpfe des Fehlbestandsberichts: Ein als Verlust gebuchtes Exemplar wird
// wiedergefunden und geht zurück in den Umlauf, oder es wird endgültig gelöscht.

import (
	"errors"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/repository"
)

// FundMelden bucht ein als Verlust gebuchtes Exemplar als beim Nachsuchen wiedergefunden und
// bringt es zurück in den Umlauf.
// @Router /buecher/exemplare/{id}/gefunden [post]
func (t *Tueren) FundMelden() http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		if id == "" {
			return apierrors.BadRequest("exemplar-id fehlt", nil)
		}
		claims, ok := auth.ClaimsAusKontext(r.Context())
		if !ok {
			return apierrors.Unauthorized("nicht angemeldet", errors.New("missing session information"))
		}

		ctx := r.Context()
		tx, err := t.DB.Pool.Begin(ctx)
		if err != nil {
			return apierrors.Internal("Transaktion konnte nicht gestartet werden", err)
		}
		defer db.Verwerfe(ctx, tx)

		abfragen := repository.NewInventurAbfragen(tx)
		gefunden, befund, err := abfragen.MarkiereVerlustAlsGefunden(ctx, id, claims.KontoID)
		if errors.Is(err, repository.ErrExemplarNichtGefunden) {
			// Zwischen Lesen und Zurückholen verschwunden — kein Serverfehler, nichts gebucht.
			return apierrors.NotFound("kein offener Verlust mit dieser ID", err)
		}
		if err != nil {
			return apierrors.Internal("Fund konnte nicht verbucht werden", err)
		}
		if !gefunden {
			return apierrors.NotFound("kein offener Verlust mit dieser ID", nil)
		}
		if err := tx.Commit(ctx); err != nil {
			return apierrors.Internal("Transaktion konnte nicht abgeschlossen werden", err)
		}
		// Die Folge für die Forderung steht in der Antwort: Der Bericht sagt, dass die
		// Forderung storniert ist — oder dass die Aufsicht zu informieren ist, weil der
		// Bescheid dort schon liegt (dann bleibt sie offen).
		tuer.AntworteJSON(w, http.StatusOK, fundAntwort{
			Status:                "ok",
			StornierteForderungen: befund.StornierteForderungen,
			StornierterBetrag:     befund.StornierterBetrag,
			Hinweis:               befund.AufsichtHinweis(),
		})
		return nil
	})
}

// fundAntwort ist die Antwort der Fund-Meldung: was mit der Forderung
// geschah, die das Buch abgerechnet hatte.
type fundAntwort struct {
	Status                string  `json:"status"`
	StornierteForderungen int     `json:"stornierte_forderungen"`
	StornierterBetrag     float64 `json:"stornierter_betrag"`
	// Hinweis ist leer, wenn nichts zu tun ist.
	Hinweis string `json:"hinweis,omitempty"`
}

// verlusteLoeschenAnfrage benennt die endgültig zu löschenden Exemplare.
type verlusteLoeschenAnfrage struct {
	ExemplarIDs []string `json:"exemplar_ids" validate:"omitempty,dive,required,uuid_oder_leer"`
}

// VerlusteEndgueltigLoeschen löscht als Verlust gebuchte Exemplare endgültig. Der allgemeine
// Weg zum Löschen sondert nur aus (ProtokollAbfragen.SondereExemplarAus); diese Tür rührt nur
// Exemplare an, die schon als Verlust gelten.
// @Router /buecher/exemplare/verlust-endgueltig-loeschen [post]
func (t *Tueren) VerlusteEndgueltigLoeschen() http.HandlerFunc {
	return apierrors.MitFehlerantwort(t.loescheVerlusteEndgueltig)
}

// loescheVerlusteEndgueltig löscht die genannten Verlust-Exemplare in einer Transaktion.
func (t *Tueren) loescheVerlusteEndgueltig(w http.ResponseWriter, r *http.Request) error {
	var req verlusteLoeschenAnfrage
	if !tuer.LeseAnfrage(w, r, &req) {
		return nil
	}
	if len(req.ExemplarIDs) == 0 {
		return apierrors.BadRequest("exemplar_ids fehlt oder ist leer", nil)
	}
	claims, ok := auth.ClaimsAusKontext(r.Context())
	if !ok {
		return apierrors.Unauthorized("nicht angemeldet", errors.New("missing session information"))
	}

	ctx := r.Context()
	tx, err := t.DB.Pool.Begin(ctx)
	if err != nil {
		return apierrors.Internal("Transaktion konnte nicht gestartet werden", err)
	}
	defer db.Verwerfe(ctx, tx)

	abfragen := repository.NewInventurAbfragen(tx)
	geloeschteIDs, err := abfragen.LoescheVerlusteEndgueltig(ctx, req.ExemplarIDs, claims.KontoID)
	if err != nil {
		// 409 statt 500: Ein gebundenes Exemplar ist eine Lage, kein Störfall. Die Meldung
		// sagt der Bedienung, welches Exemplar warum im Weg steht; MitFehlerantwort gibt sie
		// unverändert aus.
		if errors.Is(err, repository.ErrVerlustNochGebunden) {
			return apierrors.Conflict(err.Error(), err)
		}
		return apierrors.Internal("Endgültiges Löschen fehlgeschlagen", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return apierrors.Internal("Transaktion konnte nicht abgeschlossen werden", err)
	}
	// Beide Angaben: die Zahl für die Meldung, die IDs, damit die Oberfläche genau
	// die Zeilen entfernt, die wirklich weg sind — und nicht die, die sie angefragt
	// hatte. Nie nil, damit die Oberfläche [] statt null bekommt.
	if geloeschteIDs == nil {
		geloeschteIDs = []string{}
	}
	tuer.AntworteJSON(w, http.StatusOK, map[string]any{
		"geloescht":      len(geloeschteIDs),
		"geloeschte_ids": geloeschteIDs,
	})
	return nil
}
