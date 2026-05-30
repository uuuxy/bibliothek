package api

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

// DamageNoteRequest holds the payload for updating a copy's damage note.
type DamageNoteRequest struct {
	Note string `json:"note"`
}

// UpdateDamageNoteHandler updates the physical condition note of a book copy.
// @Summary      Update damage note
// @Description  Updates the custom damage or condition note text of a physical book copy.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id    path      string             true  "Book copy ID (UUID)"
// @Param        body  body      DamageNoteRequest  true  "Damage note payload"
// @Success      200   {object}  map[string]string
// @Failure      400   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /buecher/exemplare/{id}/schadensnotiz [post]
func (s *Server) UpdateDamageNoteHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing copy ID parameter"))
			return
		}

		var req DamageNoteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		query := `
			UPDATE buecher_exemplare
			SET zustand_notiz = $1, aktualisiert_am = CURRENT_TIMESTAMP
			WHERE id = $2
		`
		_, err := s.DB.Pool.Exec(ctx, query, req.Note, id)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success"}`))
	}
}

// DeleteCopyHandler removes a physical copy from circulation.
// @Summary      Delete physical book copy
// @Description  Deletes a specific physical book copy by its ID from the library catalog and registers the deletion in the audit trail.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Book copy ID (UUID)"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /buecher/exemplare/{id} [delete]
func (s *Server) DeleteCopyHandler(auditRepo repository.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.GetClaims(r.Context())
		if !ok {
			apierrors.SendHTTPError(w, http.StatusUnauthorized, errors.New("missing session information"))
			return
		}

		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing copy ID parameter"))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		err := auditRepo.DeleteCopy(ctx, id, claims.UserID)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success"}`))
	}
}

// DeleteTitleHandler deletes a book title and all its physical copies from the database, creating an audit log.
// @Summary      Delete book title
// @Description  Deletes a specific book title and all associated physical copies, registering the deletion in the audit trail.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Book title ID (UUID)"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /buecher/titel/{id} [delete]
func (s *Server) DeleteTitleHandler(auditRepo repository.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.GetClaims(r.Context())
		if !ok {
			apierrors.SendHTTPError(w, http.StatusUnauthorized, errors.New("missing session information"))
			return
		}
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing title ID parameter"))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		err := auditRepo.DeleteTitle(ctx, id, claims.UserID)
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

// AusleiheDetail holds book-loan info for one physical copy.
type AusleiheDetail struct {
	Titel     string `json:"titel"`
	Autor     string `json:"autor"`
	CoverURL  string `json:"cover_url"`
	BarcodeID string `json:"barcode_id"`
	Frist     string `json:"frist"`
}

// GraduateDetail extends the basic graduate with all open loans.
type GraduateDetail struct {
	ID            string           `json:"id"`
	BarcodeID     string           `json:"barcode_id"`
	Vorname       string           `json:"vorname"`
	Nachname      string           `json:"nachname"`
	Klasse        string           `json:"klasse"`
	AbgaengerJahr int              `json:"abgaenger_jahr"`
	IstGesperrt   bool             `json:"ist_gesperrt"`
	Ausleihen     []AusleiheDetail `json:"ausleihen"`
}

// GetGraduatesHandler lists graduating students with unreturned books.
// Pass ?details=true to include per-student loan details (for Laufzettel PDF).
// @Summary      Get list of graduating students
// @Description  Retrieves former/graduating students who still have unreturned books, optionally including loan details.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        details  query     bool  false  "True to include loan detail structures"
// @Success      200      {array}   GraduateDetail
// @Failure      500      {object}  map[string]string
// @Router       /abgaenger [get]
func (s *Server) GetGraduatesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		if r.URL.Query().Get("details") != "true" {
			// Basic list: one row per student
			query := `
				SELECT DISTINCT s.id, s.barcode_id, s.vorname, s.nachname, s.klasse, s.abgaenger_jahr, s.ist_gesperrt
				FROM schueler s
				JOIN ausleihen a ON s.id = a.schueler_id
				WHERE s.klasse IN ('9h', '10r', '13')
				  AND a.rueckgabe_am IS NULL
				ORDER BY s.klasse, s.nachname
			`
			rows, err := s.DB.Pool.Query(ctx, query)
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			defer rows.Close()

			students := []any{}
			for rows.Next() {
				var id, barcode, vorname, nachname, klasse string
				var abgaengerJahr int
				var gesperrt bool
				if err := rows.Scan(&id, &barcode, &vorname, &nachname, &klasse, &abgaengerJahr, &gesperrt); err != nil {
					apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
					return
				}
				students = append(students, map[string]any{
					"id":             id,
					"barcode_id":     barcode,
					"vorname":        vorname,
					"nachname":       nachname,
					"klasse":         klasse,
					"abgaenger_jahr": abgaengerJahr,
					"ist_gesperrt":   gesperrt,
				})
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(students)
			return
		}

		// Detail mode: one row per loan, assembled into per-student objects
		detailQuery := `
			SELECT s.id, s.barcode_id, s.vorname, s.nachname, s.klasse, s.abgaenger_jahr, s.ist_gesperrt,
			       t.titel,
			       coalesce(t.autor, '') AS autor,
			       coalesce(t.cover_url, '') AS cover_url,
			       e.barcode_id AS ex_barcode,
			       coalesce(to_char(a.rueckgabe_frist, 'DD.MM.YYYY'), '') AS frist
			FROM schueler s
			JOIN ausleihen a ON s.id = a.schueler_id
			JOIN buecher_exemplare e ON a.exemplar_id = e.id
			JOIN buecher_titel t ON e.titel_id = t.id
			WHERE s.klasse IN ('9h', '10r', '13')
			  AND a.rueckgabe_am IS NULL
			ORDER BY s.klasse, s.nachname, t.titel
		`
		rows, err := s.DB.Pool.Query(ctx, detailQuery)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		studMap := map[string]*GraduateDetail{}
		var studOrder []string
		for rows.Next() {
			var id, barcode, vorname, nachname, klasse string
			var abgaengerJahr int
			var gesperrt bool
			var titel, autor, coverURL, exBarcode, frist string
			if err := rows.Scan(&id, &barcode, &vorname, &nachname, &klasse,
				&abgaengerJahr, &gesperrt, &titel, &autor, &coverURL, &exBarcode, &frist); err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			if _, ok := studMap[id]; !ok {
				studMap[id] = &GraduateDetail{
					ID:            id,
					BarcodeID:     barcode,
					Vorname:       vorname,
					Nachname:      nachname,
					Klasse:        klasse,
					AbgaengerJahr: abgaengerJahr,
					IstGesperrt:   gesperrt,
					Ausleihen:     []AusleiheDetail{},
				}
				studOrder = append(studOrder, id)
			}
			studMap[id].Ausleihen = append(studMap[id].Ausleihen, AusleiheDetail{
				Titel:     titel,
				Autor:     autor,
				CoverURL:  coverURL,
				BarcodeID: exBarcode,
				Frist:     frist,
			})
		}

		result := make([]*GraduateDetail, 0, len(studOrder))
		for _, id := range studOrder {
			result = append(result, studMap[id])
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result)
	}
}

// AuditLogEntry represents a joined row in the audit log table.
type AuditLogEntry struct {
	ID                 string    `json:"id"`
	Tabelle            string    `json:"tabelle"`
	Aktion             string    `json:"aktion"`
	DatensatzID        string    `json:"datensatz_id"`
	Timestamp          time.Time `json:"timestamp"`
	BearbeiterID       string    `json:"bearbeiter_id"`
	BearbeiterVorname  string    `json:"bearbeiter_vorname"`
	BearbeiterNachname string    `json:"bearbeiter_nachname"`
}

// GetAuditLogsHandler returns logs of immutable security events.
// @Summary      Get audit logs
// @Description  Retrieves all immutable records in the system's audit trail, including deletions and cancellations.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Success      200  {array}   AuditLogEntry
// @Failure      500  {object}  map[string]string
// @Router       /audit [get]
func (s *Server) GetAuditLogsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		query := `
			SELECT l.id, l.tabelle, l.aktion, l.datensatz_id, l.timestamp, l.bearbeiter_id, b.vorname, b.nachname
			FROM audit_log l
			JOIN benutzer b ON l.bearbeiter_id = b.id
			ORDER BY l.timestamp DESC
		`
		rows, err := s.DB.Pool.Query(ctx, query)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		logs := []AuditLogEntry{}
		for rows.Next() {
			var l AuditLogEntry
			err := rows.Scan(&l.ID, &l.Tabelle, &l.Aktion, &l.DatensatzID, &l.Timestamp, &l.BearbeiterID, &l.BearbeiterVorname, &l.BearbeiterNachname)
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			logs = append(logs, l)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(logs)
	}
}

// GetTitleCopiesHandler lists all physical copies belonging to a book title.
func (s *Server) GetTitleCopiesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing title ID parameter"))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		query := `
			SELECT id, barcode_id, coalesce(zustand_notiz, ''), ist_ausleihbar
			FROM buecher_exemplare
			WHERE titel_id = $1
			ORDER BY barcode_id
		`
		rows, err := s.DB.Pool.Query(ctx, query, id)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		type CopyResponse struct {
			ID            string `json:"id"`
			BarcodeID     string `json:"barcode_id"`
			ZustandNotiz  string `json:"zustand_notiz"`
			IstAusleihbar bool   `json:"ist_ausleihbar"`
		}

		copies := []CopyResponse{}
		for rows.Next() {
			var cp CopyResponse
			if err := rows.Scan(&cp.ID, &cp.BarcodeID, &cp.ZustandNotiz, &cp.IstAusleihbar); err == nil {
				copies = append(copies, cp)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(copies)
	}
}

type PermissionSetting struct {
	Role       string `json:"role"`
	Permission string `json:"permission"`
	Allowed    bool   `json:"allowed"`
}

// GetPermissionsHandler returns all permission configurations grouped by role.
// @Summary      Get role permissions
// @Description  Retrieves current allowed/denied flags for permissions across all system roles.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Success      200  {array}   PermissionSetting
// @Failure      500  {object}  map[string]string
// @Router       /admin/permissions [get]
func (s *Server) GetPermissionsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		query := `
			SELECT role::text, permission, allowed 
			FROM role_permissions 
			ORDER BY role, permission
		`
		rows, err := s.DB.Pool.Query(ctx, query)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		settings := []PermissionSetting{}
		for rows.Next() {
			var ps PermissionSetting
			if err := rows.Scan(&ps.Role, &ps.Permission, &ps.Allowed); err == nil {
				ps.Role = strings.ToLower(ps.Role) // Lowercase for frontend
				settings = append(settings, ps)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(settings)
	}
}

type UpdatePermissionsRequest struct {
	Role       string `json:"role"`
	Permission string `json:"permission"`
	Allowed    bool   `json:"allowed"`
}

// UpdatePermissionsHandler updates a specific permission setting.
// @Summary      Update role permission
// @Description  Enables or disables a specific permission for a user role.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        body  body      UpdatePermissionsRequest  true  "Permission update payload"
// @Success      200   {object}  map[string]string
// @Failure      400   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /admin/permissions [put]
func (s *Server) UpdatePermissionsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req UpdatePermissionsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		query := `
			UPDATE role_permissions 
			SET allowed = $1 
			WHERE UPPER(role) = UPPER($2) AND permission = $3
		`
		_, err := s.DB.Pool.Exec(ctx, query, req.Allowed, strings.ToUpper(req.Role), req.Permission)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success"}`))
	}
}

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

		// Encrypt password
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
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

		// Save actual role to benutzer_rollen
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
			hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
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

		// Update role in benutzer_rollen
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
