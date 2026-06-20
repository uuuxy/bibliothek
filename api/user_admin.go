package api

// user_admin.go — Handlers for system user and role-permission management.
// Covers: listing/creating/updating/deleting staff accounts and reading/writing
// the role_permissions table.

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/repository"
)

// UserResponse holds public user data sent to administrative screens.
type UserResponse struct {
	ID          string    `json:"id"`
	BarcodeID   string    `json:"barcode_id"`
	Vorname     string    `json:"vorname"`
	Nachname    string    `json:"nachname"`
	Email       string    `json:"email"`
	Rolle       string    `json:"rolle"`
	Aktiv       bool      `json:"aktiv"`
	ErstelltAm  time.Time `json:"erstellt_am"`
	Permissions []string  `json:"permissions"`
}

// ListUsersHandler returns a list of all system users.
// @Summary      List system users
// @Description  Retrieves all administrative and staff users registered in the system.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Success      200  {array}   UserResponse
// @Failure      500  {object}  map[string]string
// @Router       /benutzer [get]
func (s *Server) ListUsersHandler(userRepo repository.UserRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		users, err := userRepo.GetUsers(ctx)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		responseUsers := []UserResponse{}
		for _, u := range users {
			ur := UserResponse{
				ID:         u.ID,
				BarcodeID:  u.BarcodeID,
				Vorname:    u.Vorname,
				Nachname:   u.Nachname,
				Email:      u.Email,
				Rolle:      strings.ToLower(u.Rolle),
				Aktiv:      u.Aktiv,
				ErstelltAm: u.ErstelltAm,
			}

			// Permissions analog zum Login statisch mappen
			switch ur.Rolle {
			case "admin":
				ur.Permissions = []string{"manage_users", "manage_settings", "print_classes", "manage_inventory"}
			case "mitarbeiter":
				ur.Permissions = []string{"print_classes", "manage_inventory"}
			case "lehrer":
				ur.Permissions = []string{"view_media"}
			default:
				ur.Permissions = []string{}
			}

			responseUsers = append(responseUsers, ur)
		}

		RespondJSON(w, http.StatusOK, responseUsers)
	}
}

// CreateUserRequest holds payload data for user creation.
type CreateUserRequest struct {
	BarcodeID string `json:"barcode_id"`
	Vorname   string `json:"vorname"`
	Nachname  string `json:"nachname"`
	Email     string `json:"email"`
	Rolle     string `json:"rolle"`
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
		if !DecodeJSON(w, r, &req) {
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

		// NOTE: benutzer.rolle (Enum) ist die kanonische Quelle.
		// benutzer_rollen wird nicht mehr separat geschrieben.

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success"}`))
	}
}

// UpdateUserRequest holds modification inputs for a user.
type UpdateUserRequest struct {
	BarcodeID string `json:"barcode_id"`
	Vorname   string `json:"vorname"`
	Nachname  string `json:"nachname"`
	Email     string `json:"email"`
	Rolle     string `json:"rolle"`
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
		if !DecodeJSON(w, r, &req) {
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

		// NOTE: benutzer.rolle (Enum) ist die kanonische Quelle.
		// benutzer_rollen wird nicht mehr separat geschrieben.

		// Invalidate permission cache so role changes take effect immediately
		InvalidatePermissionCache()

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success"}`))
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
		_, _ = w.Write([]byte(`{"status":"success"}`))
	}
}
