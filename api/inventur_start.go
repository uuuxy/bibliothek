package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/repository"
)

// InventurStartRequest holds the parameters needed to define the scope
// of a new physical stock-take (inventory).
type InventurStartRequest struct {
	Type string `json:"type"` // "global" | "signature" | "filter"
	// Signatur ist bei "signature" ein Präfix von buecher_titel.signatur, also eine
	// Regaladresse: "BIB Deu" erfasst auch "BIB Deu 5 KRÜ" (Migration 060). Früher
	// stand hier eine signature_id auf eine Tabelle, die niemand pflegte.
	Signatur *string `json:"signatur,omitempty"`
	Subject  *string `json:"subject,omitempty"` // bei "filter": Fach (buecher_titel.subject)
	Grade    *int    `json:"grade,omitempty"`   // bei "filter": Klasse (Jahrgangsbereich enthält)
}

// signaturNormalisiert liefert die getrimmte Signatur — eine Quelle für Label und Scope,
// damit das Label nie einen anderen Bereich benennt, als die Session tatsächlich erfasst.
func (req InventurStartRequest) signaturNormalisiert() string {
	if req.Signatur == nil {
		return ""
	}
	return strings.TrimSpace(*req.Signatur)
}

// scope leitet den auswertbaren Scope aus dem Request ab.
func (req InventurStartRequest) scope() repository.InventurScope {
	switch req.Type {
	case "signature":
		sig := req.signaturNormalisiert()
		return repository.InventurScope{Signatur: &sig}
	case "filter":
		return repository.InventurScope{Subject: req.Subject, Grade: req.Grade}
	default:
		return repository.InventurScope{}
	}
}

// InventurStartResponse returns the new session and the number of copies expected.
type InventurStartResponse struct {
	SessionID string `json:"session_id"`
	Scope     string `json:"scope"`
	Label     string `json:"label"`
	Erwartet  int    `json:"erwartet"`
}

// validateInventurStart prüft Typ und die je Typ erforderlichen Scope-Felder.
func validateInventurStart(w http.ResponseWriter, req InventurStartRequest) bool {
	switch req.Type {
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
		if req.Subject == nil && req.Grade == nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("filter scope requires at least 'subject' or 'grade'"))
			return false
		}
		return true
	default:
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("invalid type, must be 'global', 'signature' or 'filter'"))
		return false
	}
}

// scopeLabelFuer bestimmt die Anzeigebezeichnung des Inventur-Scopes. Seit Migration 060
// braucht der Signatur-Fall keinen DB-Zugriff mehr: Die Signatur IST ihre Bezeichnung.
func scopeLabelFuer(req InventurStartRequest) string {
	switch req.Type {
	case "signature":
		return "Signatur " + req.signaturNormalisiert()
	case "filter":
		return filterLabel(req)
	default:
		return "Gesamtbestand"
	}
}

// filterLabel baut ein lesbares Scope-Label, z. B. "Mathematik · Kl. 5", "Deutsch" oder "Kl. 7".
func filterLabel(req InventurStartRequest) string {
	var teile []string
	if req.Subject != nil && *req.Subject != "" {
		teile = append(teile, *req.Subject)
	}
	if req.Grade != nil {
		teile = append(teile, fmt.Sprintf("Kl. %d", *req.Grade))
	}
	if len(teile) == 0 {
		return "Teilbestand"
	}
	return strings.Join(teile, " · ")
}

// InventurStartHandler eröffnet eine neue Inventur-Session für den gewählten Scope.
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
// @Param        body  body      InventurStartRequest   true  "Scope configuration"
// @Success      200   {object}  InventurStartResponse
// @Failure      400   {object}  map[string]string
// @Failure      409   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /inventur/start [post]
func (s *Server) InventurStartHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req InventurStartRequest
		if !DecodeAndValidate(w, r, &req) {
			return
		}
		if !validateInventurStart(w, req) {
			return
		}

		ctx := r.Context()
		var benutzerID string
		if claims, ok := auth.GetClaims(ctx); ok {
			benutzerID = claims.UserID
		}

		invRepo := repository.NewInventoryRepository(s.DB.Pool)

		session, err := invRepo.CreateInventurSession(ctx, req.Type, req.scope(), scopeLabelFuer(req), benutzerID)
		if err != nil {
			if errors.Is(err, repository.ErrInventurLaeuftBereits) {
				apierrors.SendHTTPError(w, http.StatusConflict, err)
				return
			}
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		erwartet, err := invRepo.GetInventurSession(ctx, session.ID)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondJSON(w, http.StatusOK, InventurStartResponse{
			SessionID: session.ID,
			Scope:     session.ScopeType,
			Label:     session.ScopeLabel,
			Erwartet:  erwartet.Erwartet,
		})
	}
}
