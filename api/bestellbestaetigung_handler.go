package api

import (
	"context"
	"errors"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// BestaetigenRequest ist der Body von PUT /api/bestellungen/{id}/bestaetigen.
type BestaetigenRequest struct {
	// EtikettenGroesse: welche Etikettengröße der Lieferant (laut externer Rückmeldung,
	// z. B. per Naacher-Link) letztlich gewählt/gedruckt hat.
	EtikettenGroesse string `json:"etiketten_groesse"`
	// EtikettenFormat: bei 'klein' zusätzlich das Bogenraster (siehe LabelFormatAuswahl).
	// Optional — wer nachträgt, weiß es nicht immer.
	EtikettenFormat string `json:"etiketten_format"`
}

// bestellungImBestaetigungsweg beantwortet für BEIDE Bestätigungs-Handler dieselbe Frage:
// Gehört diese Bestellung überhaupt zum Bestätigungs-Weg?
// Die Abfrage und ihre zwei Wege stehen an repository.BestellungImBestaetigungsweg.
func (s *Server) bestellungImBestaetigungsweg(ctx context.Context, id string) (bool, error) {
	return repository.BestellungImBestaetigungsweg(ctx, s.DB.Pool, id)
}

// BestaetigenBestellungHandler trägt einen rein externen Vorgang nach: Lieferanten wie
// Naacher wählen über ihren eigenen Link die Etikettengröße und bestätigen die
// Bestellung selbst — Bibliosys bekommt davon keine automatische Rückmeldung. Dieser
// Endpunkt lässt jemanden aus der Bibliothek diesen Status manuell nachtragen, damit er
// in der Bestellhistorie sichtbar ist.
func (s *Server) BestaetigenBestellungHandler() http.HandlerFunc {
	return s.bestaetigenBestellung
}

// bestaetigenBestellung steht auf der obersten Ebene und nicht als Closure in
// BestaetigenBestellungHandler.
//
// Das ist keine Stilfrage: Für die Cognitive-Complexity-Messung zählt eine Closure als
// eigene Verschachtelungsebene, und dadurch wiegt jedes `if` in diesem Rumpf eins mehr.
// Als Closure kam die Funktion auf 18 und riss die Schwelle von 15 — obwohl sie nichts
// weiter tut als sechs Vorbedingungen der Reihe nach abzuräumen. Verschoben, nicht
// zerschnitten: Die Abfolge bleibt am Stück lesbar (vgl. die übrigen Handler).
func (s *Server) bestaetigenBestellung(w http.ResponseWriter, r *http.Request) {
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
	if !istBekanntesEtikettFormat(req.EtikettenFormat) {
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("unbekanntes etiketten_format"))
		return
	}

	ctx := r.Context()

	imBestaetigungsweg, err := s.bestellungImBestaetigungsweg(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("bestellung not found"))
			return
		}
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return
	}

	if !imBestaetigungsweg {
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("diese bestellung hat keinen bestaetigungsschritt"))
		return
	}

	// 'bibliothek' hält fest, dass hier jemand aus dem Haus nachgetragen hat. Über den
	// Link bestätigt der Lieferant selbst und die Spalte trägt 'lieferant' — dieselbe
	// Statuszeile, aber eine andere Aussage.
	bereits, err := s.bestaetigeBestellung(ctx, id, req.EtikettenGroesse, req.EtikettenFormat, "bibliothek")
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

// bestaetigeBestellung trägt die Bestätigung ein und meldet über bereits=true, dass sie schon
// vorlag. Der Link und der Nachtrag von Hand laufen beide hier durch: eine Stelle, an der der
// Zustand kippt. Die Anweisung steht an repository.BestaetigeBestellung.
func (s *Server) bestaetigeBestellung(ctx context.Context, bestellungID, groesse, format, durch string) (bereits bool, err error) {
	return repository.BestaetigeBestellung(ctx, s.DB.Pool, bestellungID, groesse, format, durch)
}
