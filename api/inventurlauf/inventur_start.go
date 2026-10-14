package inventurlauf

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/repository"
)

// startAnfrage holds the parameters needed to define the scope
// of a new physical stock-take (inventory).
type startAnfrage struct {
	Art string `json:"type"` // "global" | "signature" | "filter"
	// Signatur ist bei "signature" ein Präfix von buecher_titel.signatur, also eine
	// Regaladresse: "BIB Deu" erfasst auch "BIB Deu 5 KRÜ" (Migration 060). Früher
	// stand hier eine signature_id auf eine Tabelle, die niemand pflegte.
	Signatur *string `json:"signatur,omitempty"`
	Fach     *string `json:"subject,omitempty"` // bei "filter": Fach (buecher_titel.subject)
	Jahrgang *int    `json:"grade,omitempty"`   // bei "filter": Klasse (Jahrgangsbereich enthält)
}

// signaturNormalisiert liefert die getrimmte Signatur — eine Quelle für Label und Scope,
// damit das Label nie einen anderen Bereich benennt, als die Session tatsächlich erfasst.
func (a startAnfrage) signaturNormalisiert() string {
	if a.Signatur == nil {
		return ""
	}
	return strings.TrimSpace(*a.Signatur)
}

// bereich leitet den auswertbaren Scope aus dem Request ab.
func (a startAnfrage) bereich() repository.InventurBereich {
	switch a.Art {
	case "signature":
		signatur := a.signaturNormalisiert()
		return repository.InventurBereich{Signatur: &signatur}
	case "filter":
		return repository.InventurBereich{Fach: a.Fach, Jahrgang: a.Jahrgang}
	default:
		return repository.InventurBereich{}
	}
}

// startAntwort returns the new session and the number of copies expected.
type startAntwort struct {
	LaufID      string `json:"session_id"`
	Bereich     string `json:"scope"`
	Bezeichnung string `json:"label"`
	Erwartet    int    `json:"erwartet"`
}

// pruefeStartAnfrage prüft Typ und die je Typ erforderlichen Scope-Felder.
func pruefeStartAnfrage(w http.ResponseWriter, req startAnfrage) bool {
	switch req.Art {
	case "global":
		return true
	case "signature":
		// Leerstring abweisen: Er wäre ein Präfix JEDER Signatur und machte aus der
		// Regal-Inventur unbemerkt eine Gesamtinventur.
		if req.signaturNormalisiert() == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("signatur ist erforderlich, wenn type 'signature' ist"))
			return false
		}
		return true
	case "filter":
		if req.Fach == nil && req.Jahrgang == nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("filter scope requires at least 'subject' or 'grade'"))
			return false
		}
		return true
	default:
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("invalid type, must be 'global', 'signature' or 'filter'"))
		return false
	}
}

// bezeichnungFuer bestimmt die Anzeigebezeichnung des Inventur-Scopes. Seit Migration 060
// braucht der Signatur-Fall keinen DB-Zugriff mehr: Die Signatur IST ihre Bezeichnung.
func bezeichnungFuer(req startAnfrage) string {
	switch req.Art {
	case "signature":
		return "Signatur " + req.signaturNormalisiert()
	case "filter":
		return filterBezeichnung(req)
	default:
		return "Gesamtbestand"
	}
}

// filterBezeichnung baut ein lesbares Scope-Label, z. B. "Mathematik · Kl. 5", "Deutsch" oder "Kl. 7".
func filterBezeichnung(req startAnfrage) string {
	var teile []string
	if req.Fach != nil && *req.Fach != "" {
		teile = append(teile, *req.Fach)
	}
	if req.Jahrgang != nil {
		teile = append(teile, fmt.Sprintf("Kl. %d", *req.Jahrgang))
	}
	if len(teile) == 0 {
		return "Teilbestand"
	}
	return strings.Join(teile, " · ")
}

// InventurStarten eröffnet eine neue Inventur-Session für den gewählten Scope.
//
// Anders als früher wird KEIN globaler Zustand mehr zurückgesetzt: Jede Session hat
// ihren eigenen, session-gebundenen Fortschritt. Läuft für denselben Scope bereits
// eine Session, antwortet der Handler mit 409 — der bisherige stille globale Reset
// (der den Fortschritt eines parallel scannenden Kollegen löschte) entfällt.
//
// @Summary      Start an inventory session
// @Description  Opens a new inventory session for the chosen scope (global, per signature, or a Fach/Klasse filter).
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Param        body  body      startAnfrage   true  "Scope configuration"
// @Success      200   {object}  startAntwort
// @Failure      400   {object}  map[string]string
// @Failure      409   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /inventur/start [post]
func (t *Tueren) InventurStarten() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req startAnfrage
		if !tuer.LeseAnfrage(w, r, &req) {
			return
		}
		if !pruefeStartAnfrage(w, req) {
			return
		}

		ctx := r.Context()
		var benutzerID string
		if claims, ok := auth.GetClaims(ctx); ok {
			benutzerID = claims.UserID
		}

		abfragen := repository.NewInventurAbfragen(t.DB.Pool)

		lauf, err := abfragen.LegeLaufAn(ctx, req.Art, req.bereich(), bezeichnungFuer(req), benutzerID)
		if err != nil {
			// Identischer Scope läuft schon ODER der gewünschte überschneidet sich mit
			// einer offenen Inventur — beides ist ein bewusster 409, kein Serverfehler.
			if errors.Is(err, repository.ErrInventurLaeuftBereits) || errors.Is(err, repository.ErrInventurUeberlappt) {
				apierrors.SendHTTPError(w, http.StatusConflict, err)
				return
			}
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		erwartet, err := abfragen.LadeLaufMitErwartung(ctx, lauf.ID)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		tuer.AntworteJSON(w, http.StatusOK, startAntwort{
			LaufID:      lauf.ID,
			Bereich:     lauf.BereichArt,
			Bezeichnung: lauf.Bezeichnung,
			Erwartet:    erwartet.Erwartet,
		})
	}
}
