package api

import (
	"errors"
	"net/http"
	"strings"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/pkg/httpresp"
	"bibliothek/repository"
)

// CreateUserRequest holds payload data for user creation.
type CreateUserRequest struct {
	BarcodeID string `json:"barcode_id"`
	Vorname   string `json:"vorname" validate:"required"`
	Nachname  string `json:"nachname" validate:"required"`
	Email     string `json:"email" validate:"required,email"`
	Rolle     string `json:"rolle" validate:"required"`
}

// CreateUserHandler inserts a new user with bcrypt-hashed credentials.
// @Summary      Create system user
// @Description  Registers a new system user (admin, teacher, staff) with hashed password and role assignments.
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

		// Validate email uniqueness
		exists, err := userRepo.CheckEmailExists(ctx, req.Email, "")
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if exists {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("ein Benutzer mit dieser E-Mail existiert bereits"))
			return
		}

		// Validate barcode uniqueness if provided
		var barcode *string
		if req.BarcodeID != "" {
			barcode = &req.BarcodeID
			exists, err = userRepo.CheckBarcodeExists(ctx, req.BarcodeID, "")
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			if exists {
				apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("dieser Barcode wird bereits verwendet"))
				return
			}
		}

		dbEnumRole := strings.ToLower(req.Rolle)
		if dbEnumRole != "admin" && dbEnumRole != "lehrer" && dbEnumRole != "mitarbeiter" && dbEnumRole != "helfer" {
			dbEnumRole = "mitarbeiter"
		}

		_, err = userRepo.CreateUser(ctx, barcode, req.Vorname, req.Nachname, req.Email, dbEnumRole)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		httpresp.Write(w, []byte(`{"status":"success"}`))
	}
}

type UpdateUserRequest struct {
	BarcodeID string `json:"barcode_id"`
	Vorname   string `json:"vorname" validate:"required"`
	Nachname  string `json:"nachname" validate:"required"`
	Email     string `json:"email" validate:"required,email"`
	Rolle     string `json:"rolle" validate:"required"`
	Password  string `json:"password,omitempty"` // nur gehasht, wenn nicht leer
	Aktiv     bool   `json:"aktiv"`
}

// UpdateUserHandler modifies user properties and dynamically updates details/passwords.
// @Summary      Update system user
// @Description  Modifies an existing user's properties, role, active status, or password.
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
		claims, ok := auth.GetClaims(r.Context())
		if ok && claims.UserID == id {
			if strings.ToUpper(req.Rolle) != "ADMIN" {
				apierrors.SendHTTPError(w, http.StatusForbidden, errors.New("eigene Admin-Rolle kann nicht herabgestuft werden"))
				return
			}
			if !req.Aktiv {
				apierrors.SendHTTPError(w, http.StatusForbidden, errors.New("eigenes Konto kann nicht deaktiviert werden"))
				return
			}
		}

		ctx := r.Context()

		// Validate email uniqueness excluding this user
		exists, err := userRepo.CheckEmailExists(ctx, req.Email, id)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if exists {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("ein Benutzer mit dieser E-Mail existiert bereits"))
			return
		}

		// Validate barcode uniqueness excluding this user
		var barcode *string
		if req.BarcodeID != "" {
			barcode = &req.BarcodeID
			exists, err = userRepo.CheckBarcodeExists(ctx, req.BarcodeID, id)
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			if exists {
				apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("dieser Barcode wird bereits von einem anderen Benutzer verwendet"))
				return
			}
		}

		dbEnumRole := strings.ToLower(req.Rolle)
		if dbEnumRole != "admin" && dbEnumRole != "lehrer" && dbEnumRole != "mitarbeiter" && dbEnumRole != "helfer" {
			dbEnumRole = "mitarbeiter"
		}

		err = userRepo.UpdateUser(ctx, id, barcode, req.Vorname, req.Nachname, req.Email, dbEnumRole, req.Aktiv)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		InvalidatePermissionCache()

		w.Header().Set("Content-Type", "application/json")
		httpresp.Write(w, []byte(`{"status":"success"}`))
	}
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
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		httpresp.Write(w, []byte(`{"status":"success"}`))
	}
}
