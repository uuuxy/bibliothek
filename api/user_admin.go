package api

// user_admin.go — Handlers for system user and role-permission management.
// Covers: listing/creating/updating/deleting staff accounts and reading/writing
// the role_permissions table.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/repository"

	"golang.org/x/crypto/bcrypt"
)

// UserResponse holds public user data sent to administrative screens.
type UserResponse struct {
	ID         string    `json:"id"`
	BarcodeID  string    `json:"barcode_id"`
	Vorname    string    `json:"vorname"`
	Nachname   string    `json:"nachname"`
	Email      string    `json:"email"`
	Rolle      string    `json:"rolle"`
	Aktiv      bool      `json:"aktiv"`
	ErstelltAm time.Time `json:"erstellt_am"`
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
func (s *Server) ListUsersHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		query := `
			SELECT b.id, coalesce(b.barcode_id, ''), b.vorname, b.nachname, b.email, coalesce(br.rolle, 'HELFER'), b.aktiv, b.erstellt_am
			FROM benutzer b
			LEFT JOIN benutzer_rollen br ON b.id = br.benutzer_id
			ORDER BY b.nachname, b.vorname
		`
		rows, err := s.DB.Pool.Query(ctx, query)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		users := []UserResponse{}
		for rows.Next() {
			var u UserResponse
			err := rows.Scan(&u.ID, &u.BarcodeID, &u.Vorname, &u.Nachname, &u.Email, &u.Rolle, &u.Aktiv, &u.ErstelltAm)
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			u.Rolle = strings.ToLower(u.Rolle) // Normalize for frontend
			users = append(users, u)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(users)
	}
}

// CreateUserRequest holds payload data for user creation.
type CreateUserRequest struct {
	BarcodeID string `json:"barcode_id"`
	Vorname   string `json:"vorname"`
	Nachname  string `json:"nachname"`
	Email     string `json:"email"`
	Rolle     string `json:"rolle"`
	Password  string `json:"password"`
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
func (s *Server) CreateUserHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}

		if req.Vorname == "" || req.Nachname == "" || req.Email == "" || req.Rolle == "" || req.Password == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("alle Felder sind Pflichtfelder"))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		// Validate email uniqueness
		var exists bool
		err := s.DB.Pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM benutzer WHERE email = $1)", req.Email).Scan(&exists)
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
			err = s.DB.Pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM benutzer WHERE barcode_id = $1)", req.BarcodeID).Scan(&exists)
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			if exists {
				apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("dieser Barcode wird bereits verwendet"))
				return
			}
		}

		// Encrypt password with bcrypt cost factor 10
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		dbEnumRole := strings.ToLower(req.Rolle)
		if dbEnumRole != "admin" && dbEnumRole != "lehrer" && dbEnumRole != "mitarbeiter" {
			dbEnumRole = "mitarbeiter"
		}

		var userID string
		query := `
			INSERT INTO benutzer (barcode_id, vorname, nachname, email, passwort_hash, rolle, aktiv)
			VALUES ($1, $2, $3, $4, $5, $6::benutzer_rolle, true)
			RETURNING id
		`
		err = s.DB.Pool.QueryRow(ctx, query, barcode, req.Vorname, req.Nachname, req.Email, string(hash), dbEnumRole).Scan(&userID)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// Persist the actual role to benutzer_rollen (used for RBAC checks)
		_, err = s.DB.Pool.Exec(ctx, `
			INSERT INTO benutzer_rollen (benutzer_id, rolle)
			VALUES ($1, $2)
			ON CONFLICT (benutzer_id) DO UPDATE SET rolle = EXCLUDED.rolle
		`, userID, strings.ToUpper(req.Rolle))
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

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
	Password  string `json:"password"`
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
func (s *Server) UpdateUserHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing user ID parameter"))
			return
		}

		var req UpdateUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}

		if req.Vorname == "" || req.Nachname == "" || req.Email == "" || req.Rolle == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("vorname, Nachname, E-Mail und Rolle sind Pflichtfelder"))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		// Validate email uniqueness excluding this user
		var exists bool
		err := s.DB.Pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM benutzer WHERE email = $1 AND id != $2)", req.Email, id).Scan(&exists)
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
			err = s.DB.Pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM benutzer WHERE barcode_id = $1 AND id != $2)", req.BarcodeID, id).Scan(&exists)
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
		if dbEnumRole != "admin" && dbEnumRole != "lehrer" && dbEnumRole != "mitarbeiter" {
			dbEnumRole = "mitarbeiter"
		}

		if req.Password != "" {
			hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			query := `
				UPDATE benutzer
				SET barcode_id = $1, vorname = $2, nachname = $3, email = $4, rolle = $5::benutzer_rolle, aktiv = $6, passwort_hash = $7, aktualisiert_am = CURRENT_TIMESTAMP
				WHERE id = $8
			`
			_, err = s.DB.Pool.Exec(ctx, query, barcode, req.Vorname, req.Nachname, req.Email, dbEnumRole, req.Aktiv, string(hash), id)
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
		} else {
			query := `
				UPDATE benutzer
				SET barcode_id = $1, vorname = $2, nachname = $3, email = $4, rolle = $5::benutzer_rolle, aktiv = $6, aktualisiert_am = CURRENT_TIMESTAMP
				WHERE id = $7
			`
			_, err = s.DB.Pool.Exec(ctx, query, barcode, req.Vorname, req.Nachname, req.Email, dbEnumRole, req.Aktiv, id)
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
		}

		// Update role in benutzer_rollen (used for RBAC checks)
		_, err = s.DB.Pool.Exec(ctx, `
			INSERT INTO benutzer_rollen (benutzer_id, rolle)
			VALUES ($1, $2)
			ON CONFLICT (benutzer_id) DO UPDATE SET rolle = EXCLUDED.rolle
		`, id, strings.ToUpper(req.Rolle))
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

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

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		err := auditRepo.DeleteUser(ctx, id, claims.UserID)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success"}`))
	}
}
