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

// ausweisSchule ist der Kopf der Karte. Er kommt mit dem Design, weil auf jedem Weg, der Ausweise
// druckt, der echte Schulname auf die Karte gehört, und wer druckt, die Einstellungen oft nicht
// lesen darf.
type ausweisSchule struct {
	Name    string `json:"name"`
	Adresse string `json:"adresse"`
}

// meldungDesignVeraltet steht im Designer, dessen Stand ein anderer Platz überholt hat.
const meldungDesignVeraltet = "Das Ausweis-Design wurde inzwischen an einem anderen Platz geändert. Nichts gespeichert."

// designMitSchule stellt der Antwort den Kopf voran: die Schule und den Änderungsstand des
// Designs, den der Designer beim Speichern zurückschickt (leer: noch keines gespeichert). Das
// gespeicherte Design bleibt dabei Byte für Byte, wie es ist. Ist es kein Objekt, gilt es als
// nicht gespeichert; die Oberfläche nimmt dann ihre Vorgaben.
func designMitSchule(design, stand string, e *repository.SystemEinstellungen) ([]byte, error) {
	ort := strings.TrimSpace(e.SchulePLZ + " " + e.SchuleOrt)
	adresse := e.SchuleStrasse
	if adresse != "" && ort != "" {
		adresse += ", "
	}
	kopf, err := json.Marshal(struct {
		Schule ausweisSchule `json:"schule"`
		Stand  string        `json:"stand"`
	}{ausweisSchule{Name: e.SchuleName, Adresse: adresse + ort}, stand})
	if err != nil {
		return nil, err
	}
	rumpf := strings.TrimSpace(design)
	if !strings.HasPrefix(rumpf, "{") {
		return kopf, nil
	}
	felder := strings.TrimSpace(strings.TrimSuffix(rumpf[1:], "}"))
	if felder == "" {
		return kopf, nil
	}
	return append(append(kopf[:len(kopf)-1], ','), felder+"}"...), nil
}

// AusweisLayout liefert den Kopf der Schule und das gespeicherte Design der Ausweise als JSON.
// Ist keines gespeichert, steht nur der Kopf da, und die Oberfläche nimmt ihre Vorgaben.
//
// Der Fehler wird vor dem leeren Wert geprüft, weil der Wert nach einem gescheiterten Scan
// immer leer ist; gedruckt würde sonst mit den Vorgaben statt mit dem Design der Schule.
func (t *Tueren) AusweisLayout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wert, stand, err := repository.LadeAusweisLayout(r.Context(), t.DB.Pool)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			wert = "{}"
		case err != nil:
			apierrors.AntworteFehler(w, http.StatusInternalServerError, fmt.Errorf("ausweis-design laden: %w", err))
			return
		case strings.TrimSpace(wert) == "":
			wert = "{}"
		}
		einstellungen, err := repository.LadeEinstellungen(r.Context(), t.DB.Pool)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, fmt.Errorf("ausweis-design laden: einstellungen: %w", err))
			return
		}
		antwort, err := designMitSchule(wert, stand, einstellungen)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, fmt.Errorf("ausweis-design laden: %w", err))
			return
		}
		w.Header().Set(tuer.HeaderContentType, "application/json; charset=utf-8")
		_, _ = w.Write(antwort) //nolint:errcheck // Die Antwort ist schon gesendet.
	}
}

// AusweisLayoutSchreiben speichert das Design der Ausweise für alle Arbeitsplätze. Es nimmt
// nur gültiges JSON bis MaxAusweisLayoutBytes an, und nur auf dem Stand, den der Designer
// geladen hat (?stand=, leer: noch keines gespeichert). Hat ein anderer Platz inzwischen
// gespeichert, antwortet es mit 409 und schreibt nichts. Die Antwort nennt den neuen Stand.
func (t *Tueren) AusweisLayoutSchreiben() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, MaxAusweisLayoutBytes+1))
		if err != nil {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, apierrors.MeldungAnfrageUnlesbar, err)
			return
		}
		if len(body) > MaxAusweisLayoutBytes {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusRequestEntityTooLarge, "Ausweis-Design ist zu groß", nil)
			return
		}
		if !json.Valid(body) {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Das Ausweis-Design ließ sich nicht lesen", nil)
			return
		}

		stand, err := repository.SchreibeAusweisLayout(r.Context(), t.DB.Pool, string(body), r.URL.Query().Get("stand"))
		if errors.Is(err, repository.ErrAusweisLayoutVeraltet) {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusConflict, meldungDesignVeraltet, err)
			return
		}
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, fmt.Errorf("ausweis-design speichern: %w", err))
			return
		}

		tuer.AntworteJSON(w, http.StatusOK, map[string]string{"status": "gespeichert", "stand": stand})
	}
}
