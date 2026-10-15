package druck

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// MaxAusweisLayoutBytes begrenzt das Design (Base64-Logos können groß werden).
const MaxAusweisLayoutBytes = 5 << 20 // 5 MiB

// AusweisLayout liefert das gespeicherte Design der Ausweise als JSON und "{}", solange keines
// gespeichert ist; dann nimmt die Oberfläche ihre Vorgaben.
//
// Der Fehler wird vor dem leeren Wert geprüft, weil der Wert nach einem gescheiterten Scan
// immer leer ist: Auf "{}" speichert der Designer seine Vorgaben, für alle Arbeitsplätze.
func (t *Tueren) AusweisLayout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wert, err := repository.LadeAusweisLayout(r.Context(), t.DB.Pool)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			wert = "{}"
		case err != nil:
			apierrors.AntworteFehler(w, http.StatusInternalServerError, fmt.Errorf("ausweis-design laden: %w", err))
			return
		case strings.TrimSpace(wert) == "":
			wert = "{}"
		}
		w.Header().Set(tuer.HeaderContentType, "application/json; charset=utf-8")
		_, _ = w.Write([]byte(wert)) //nolint:errcheck // Die Antwort ist schon gesendet.
	}
}

// AusweisLayoutSchreiben speichert das Design der Ausweise für alle Arbeitsplätze. Es nimmt
// nur gültiges JSON bis MaxAusweisLayoutBytes an.
func (t *Tueren) AusweisLayoutSchreiben() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, MaxAusweisLayoutBytes+1))
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusBadRequest, errors.New("konnte die Anfrage nicht lesen"))
			return
		}
		if len(body) > MaxAusweisLayoutBytes {
			apierrors.AntworteFehler(w, http.StatusRequestEntityTooLarge, errors.New("Ausweis-Design ist zu groß"))
			return
		}
		if !json.Valid(body) {
			apierrors.AntworteFehler(w, http.StatusBadRequest, errors.New("ungültiges JSON"))
			return
		}

		if err := repository.SchreibeAusweisLayout(r.Context(), t.DB.Pool, string(body)); err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, errors.New("Ausweis-Design konnte nicht gespeichert werden"))
			return
		}

		tuer.AntworteJSON(w, http.StatusOK, map[string]string{"status": "gespeichert"})
	}
}
