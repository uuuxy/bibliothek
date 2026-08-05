package api

import (
	"context"
	"errors"
	"net/http"

	"bibliothek/apierrors"

	"github.com/jackc/pgx/v5"
)

// BestaetigenRequest ist der Body von PUT /api/bestellungen/{id}/bestaetigen.
type BestaetigenRequest struct {
	// EtikettenGroesse: welche Etikettengröße der Lieferant (laut externer Rückmeldung,
	// z. B. per Naacher-Link) letztlich gewählt/gedruckt hat.
	EtikettenGroesse string `json:"etiketten_groesse"`
}

// BestaetigenBestellungHandler trägt einen rein externen Vorgang nach: Lieferanten wie
// Naacher wählen über ihren eigenen Link die Etikettengröße und bestätigen die
// Bestellung selbst — Bibliosys bekommt davon keine automatische Rückmeldung. Dieser
// Endpunkt lässt jemanden aus der Bibliothek diesen Status manuell nachtragen, damit er
// in der Bestellhistorie sichtbar ist.
func (s *Server) BestaetigenBestellungHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing bestellung id"))
			return
		}

		var req BestaetigenRequest
		if !DecodeAndValidate(w, r, &req) {
			return
		}
		if req.EtikettenGroesse != "klein" && req.EtikettenGroesse != "gross" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("etiketten_groesse muss 'klein' oder 'gross' sein"))
			return
		}

		ctx := r.Context()

		var bietetBestellbestaetigung bool
		// COALESCE gegen lieferant_id IS NULL (ON DELETE SET NULL, Migration 037) — eine
		// Bestellung überlebt den gelöschten Lieferanten als Beleg, ein NULL-Scan in *bool
		// würde diesen sonst mit einem 500 abbrechen (siehe Memory NULL-Scan-Bugklasse).
		err := s.DB.Pool.QueryRow(ctx, `
			SELECT coalesce(l.bietet_bestellbestaetigung, false)
			FROM bestellungen_verlauf b
			LEFT JOIN lieferanten l ON l.id = b.lieferant_id
			WHERE b.id = $1
		`, id).Scan(&bietetBestellbestaetigung)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("bestellung not found"))
				return
			}
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		if !bietetBestellbestaetigung {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("dieser lieferant bietet keine bestellbestaetigung an"))
			return
		}

		// 'bibliothek' hält fest, dass hier jemand aus dem Haus nachgetragen hat. Über den
		// Link bestätigt der Lieferant selbst und die Spalte trägt 'lieferant' — dieselbe
		// Statuszeile, aber eine andere Aussage.
		bereits, err := s.bestaetigeBestellung(ctx, id, req.EtikettenGroesse, "bibliothek")
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if bereits {
			apierrors.SendHTTPError(w, http.StatusConflict, errors.New("bestellung ist bereits bestaetigt"))
			return
		}

		RespondJSON(w, http.StatusOK, map[string]any{
			"status":            "success",
			"etiketten_groesse": req.EtikettenGroesse,
		})
	}
}

// bestaetigeBestellung trägt die Bestätigung ein und meldet über bereits=true, dass sie
// schon vorlag. Beide Wege — Link und manueller Nachtrag — laufen hier durch, damit es
// nur EINE Stelle gibt, an der der Zustand kippt.
//
// bestaetigt_am IS NULL macht das atomar: Bestätigen der Lieferant und die Bibliothek
// gleichzeitig (oder zwei Arbeitsplätze im Multi-PC-Betrieb), gewinnt genau einer — der
// andere bekommt 409, statt den Eintrag still zu überschreiben. Eine getrennte
// Vorab-Prüfung hätte zwischen SELECT und UPDATE ein Wettlauf-Fenster.
//
// groesse darf leer sein: Über den Link ist die Etikettengröße nur eine Notiz, kein
// Pflichtfeld. NULLIF hält die Spalte dann auf NULL, wie es der CHECK verlangt.
func (s *Server) bestaetigeBestellung(ctx context.Context, bestellungID, groesse, durch string) (bereits bool, err error) {
	tag, err := s.DB.Pool.Exec(ctx, `
		UPDATE bestellungen_verlauf
		SET bestaetigt_am = now(), etiketten_groesse = NULLIF($1, ''), bestaetigt_durch = $2
		WHERE id = $3 AND bestaetigt_am IS NULL
	`, groesse, durch, bestellungID)
	if err != nil {
		return false, err
	}
	// Kein Löschpfad für bestellungen_verlauf existiert — 0 betroffene Zeilen nach der
	// Existenzprüfung des Aufrufers heißt daher immer: schon bestätigt.
	return tag.RowsAffected() == 0, nil
}
