package api

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/pkg/httpresp"
	"bibliothek/repository"
)

// normalisiereBenutzerRolle bildet die Eingaberolle auf einen gültigen DB-Enum-Wert
// ab; unbekannte Rollen werden auf "mitarbeiter" zurückgesetzt.
func normalisiereBenutzerRolle(rolle string) string {
	dbEnumRole := strings.ToLower(rolle)
	if dbEnumRole != "admin" && dbEnumRole != "lehrer" && dbEnumRole != "mitarbeiter" && dbEnumRole != "helfer" {
		dbEnumRole = "mitarbeiter"
	}
	return dbEnumRole
}

// pruefeEmailEindeutig prüft die E-Mail-Eindeutigkeit (excludeID leer bei Neuanlage,
// sonst die eigene ID). Bei Konflikt oder DB-Fehler wird die HTTP-Antwort direkt
// geschrieben und false zurückgegeben.
func pruefeEmailEindeutig(ctx context.Context, w http.ResponseWriter, userRepo repository.UserRepository, email, excludeID string) bool {
	exists, err := userRepo.CheckEmailExists(ctx, email, excludeID)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return false
	}
	if exists {
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("ein Benutzer mit dieser E-Mail existiert bereits"))
		return false
	}
	return true
}

// pruefeBarcodeEindeutig liefert den optionalen Barcode-Pointer und validiert dessen
// Eindeutigkeit. Ist kein Barcode gesetzt, wird (nil, true) geliefert. Bei Konflikt
// oder DB-Fehler wird die HTTP-Antwort direkt geschrieben (ok=false).
func pruefeBarcodeEindeutig(ctx context.Context, w http.ResponseWriter, userRepo repository.UserRepository, barcodeID, excludeID, konfliktMsg string) (barcode *string, ok bool) {
	if barcodeID == "" {
		return nil, true
	}
	exists, err := userRepo.CheckBarcodeExists(ctx, barcodeID, excludeID)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return nil, false
	}
	if exists {
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New(konfliktMsg))
		return nil, false
	}
	return &barcodeID, true
}

// CreateUserRequest holds payload data for user creation.
type CreateUserRequest struct {
	BarcodeID string `json:"barcode_id"`
	Vorname   string `json:"vorname" validate:"required"`
	Nachname  string `json:"nachname" validate:"required"`
	Email     string `json:"email" validate:"required,email"`
	Rolle     string `json:"rolle" validate:"required"`
}

// CreateUserHandler inserts a new user. Es gibt keine lokalen Passwörter — die
// Authentifizierung läuft über den Schul-Mailserver (IMAP) bzw. Barcode/PIN.
// @Summary      Create system user
// @Description  Registers a new system user (admin, teacher, staff) with role assignments. Login erfolgt über IMAP/Barcode, nicht über ein lokales Passwort.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        body  body      CreateUserRequest  true  "User registration payload"
// @Success      200   {object}  map[string]string
// @Failure      400   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /benutzer [post]
func (s *Server) CreateUserHandler(userRepo repository.UserRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateUserRequest
		if !DecodeAndValidate(w, r, &req) {
			return
		}

		if req.Vorname == "" || req.Nachname == "" || req.Email == "" || req.Rolle == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("alle Felder sind Pflichtfelder"))
			return
		}

		ctx := r.Context()

		if !pruefeEmailEindeutig(ctx, w, userRepo, req.Email, "") {
			return
		}

		barcode, ok := pruefeBarcodeEindeutig(ctx, w, userRepo, req.BarcodeID, "", "dieser Barcode wird bereits verwendet")
		if !ok {
			return
		}

		dbEnumRole := normalisiereBenutzerRolle(req.Rolle)

		if _, err := userRepo.CreateUser(ctx, barcode, req.Vorname, req.Nachname, req.Email, dbEnumRole); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set(headerContentType, contentTypeJSON)
		httpresp.Write(w, []byte(`{"status":"success"}`))
	}
}

type UpdateUserRequest struct {
	BarcodeID string `json:"barcode_id"`
	Vorname   string `json:"vorname" validate:"required"`
	Nachname  string `json:"nachname" validate:"required"`
	Email     string `json:"email" validate:"required,email"`
	Rolle     string `json:"rolle" validate:"required"`
	Aktiv     bool   `json:"aktiv"`
	// Kein Passwort-Feld: Staff-Logins laufen über den Schul-Mailserver (IMAP) bzw.
	// Barcode/PIN — es gibt keine lokale Passwortspalte (siehe Migration 012). Ein früher
	// hier vorhandenes `password`-Feld wurde ersatzlos entfernt, weil der Wert nirgends
	// gespeichert wurde und Admins fälschlich glauben ließ, ein Passwort zu setzen.
}

// UpdateUserHandler modifies user properties (Name, E-Mail, Barcode, Rolle, Aktiv-Status).
// Passwörter gibt es hier nicht — Login läuft über IMAP/Barcode (siehe CreateUserHandler).
// @Summary      Update system user
// @Description  Modifies an existing user's properties, role, or active status.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id    path      string             true  "User ID (UUID)"
// @Param        body  body      UpdateUserRequest  true  "User update payload"
// @Success      200   {object}  map[string]string
// @Failure      400   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /benutzer/{id} [put]
func (s *Server) UpdateUserHandler(userRepo repository.UserRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing user ID parameter"))
			return
		}

		var req UpdateUserRequest
		if !DecodeAndValidate(w, r, &req) {
			return
		}

		if req.Vorname == "" || req.Nachname == "" || req.Email == "" || req.Rolle == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("vorname, Nachname, E-Mail und Rolle sind Pflichtfelder"))
			return
		}

		// Prevent admin self-demotion or self-deactivation
		if !pruefeAdminSelbstschutz(w, r, id, req) {
			return
		}

		ctx := r.Context()

		if !pruefeEmailEindeutig(ctx, w, userRepo, req.Email, id) {
			return
		}

		barcode, ok := pruefeBarcodeEindeutig(ctx, w, userRepo, req.BarcodeID, id, "dieser Barcode wird bereits von einem anderen Benutzer verwendet")
		if !ok {
			return
		}

		dbEnumRole := normalisiereBenutzerRolle(req.Rolle)

		if err := userRepo.UpdateUser(ctx, repository.UpdateUserParams{
			ID: id, Barcode: barcode, Vorname: req.Vorname, Nachname: req.Nachname,
			Email: req.Email, Rolle: dbEnumRole, Aktiv: req.Aktiv,
		}); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		InvalidatePermissionCache()

		w.Header().Set(headerContentType, contentTypeJSON)
		httpresp.Write(w, []byte(`{"status":"success"}`))
	}
}

// pruefeAdminSelbstschutz verhindert, dass ein Admin die eigene Rolle herabstuft
// oder das eigene Konto deaktiviert. Bei einem Verstoß wird die HTTP-Antwort direkt
// geschrieben und false zurückgegeben.
func pruefeAdminSelbstschutz(w http.ResponseWriter, r *http.Request, id string, req UpdateUserRequest) bool {
	claims, ok := auth.GetClaims(r.Context())
	if !ok || claims.UserID != id {
		return true
	}
	if strings.ToUpper(req.Rolle) != "ADMIN" {
		apierrors.SendHTTPError(w, http.StatusForbidden, errors.New("eigene Admin-Rolle kann nicht herabgestuft werden"))
		return false
	}
	if !req.Aktiv {
		apierrors.SendHTTPError(w, http.StatusForbidden, errors.New("eigenes Konto kann nicht deaktiviert werden"))
		return false
	}
	return true
}

// DeleteUserHandler deletes a user and logs it in the audit log.
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
func (s *Server) DeleteUserHandler(auditRepo repository.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		err := auditRepo.DeleteUser(ctx, id, claims.UserID)
		if err != nil {
			// Aktive Handapparat-Ausleihen sind ein Konflikt (409), kein Serverfehler:
			// Der Admin muss die Bücher erst zurückbuchen.
			if errors.Is(err, repository.ErrUserHasActiveLoans) {
				apierrors.SendHTTPError(w, http.StatusConflict, err)
				return
			}
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set(headerContentType, contentTypeJSON)
		httpresp.Write(w, []byte(`{"status":"success"}`))
	}
}
