package benutzer

import (
	"errors"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/pkg/httpresp"
	"bibliothek/repository"
)

// Das Löschen eines Benutzers steht für sich, weil es eigene Regeln hat, die mit dem
// Anlegen und Ändern nichts zu tun haben: kein Selbstlöschen, keine Admin-Konten ohne
// Adminrechte, und aktive Handapparat-Ausleihen sind ein Konflikt (409), kein Fehler.

// KontoLoeschen deletes a user and logs it in the audit log.
// @Summary      Delete user
// @Description  Deletes a system user by their ID and registers the deletion in the audit log.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "User ID (UUID)"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /benutzer/{id} [delete]
func (t *Tueren) KontoLoeschen(protokollAbfragen repository.ProtokollAbfragen, kontoAbfragen repository.KontoAbfragen) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t.loescheKonto(w, r, protokollAbfragen, kontoAbfragen)
	}
}

// loescheKonto prüft, ob das Konto gelöscht werden darf, und ordnet die Ablehnungen des
// Löschens ein.
func (t *Tueren) loescheKonto(w http.ResponseWriter, r *http.Request, protokollAbfragen repository.ProtokollAbfragen, kontoAbfragen repository.KontoAbfragen) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		apierrors.SendHTTPError(w, http.StatusUnauthorized, errors.New("missing session information"))
		return
	}
	id := r.PathValue("id")
	if id == "" {
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing user ID parameter"))
		return
	}

	// Prevent self-deletion
	if id == claims.UserID {
		apierrors.SendHTTPError(w, http.StatusForbidden, errors.New("eigenes Konto kann nicht gelöscht werden"))
		return
	}

	ctx := r.Context()

	// Ohne diese Prüfung wäre die Rechtetrennung an der Bearbeitung wirkungslos:
	// Wer den letzten Administrator nicht ändern darf, aber löschen kann, entfernt
	// damit auch jede Instanz, die eine Selbstbeförderung noch zurücknehmen könnte.
	if !pruefeAdminZiel(ctx, w, r, kontoAbfragen, id) {
		return
	}

	err := protokollAbfragen.LoescheKonto(ctx, id, claims.UserID)
	if err != nil {
		// Aktive Handapparat-Ausleihen sind ein Konflikt (409), kein Serverfehler:
		// Der Admin muss die Bücher erst zurückbuchen.
		if errors.Is(err, repository.ErrKontoHatAusleihen) {
			apierrors.SendHTTPError(w, http.StatusConflict, err)
			return
		}
		if errors.Is(err, repository.ErrKontoNichtGefunden) {
			apierrors.SendHTTPError(w, http.StatusNotFound, err)
			return
		}
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set(tuer.HeaderContentType, tuer.ContentTypeJSON)
	httpresp.Schreibe(w, []byte(`{"status":"success"}`))
}
