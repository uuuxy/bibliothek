package api

// user_admin.go — Handlers for system user and role-permission management.
// Covers: listing/creating/updating/deleting staff accounts and reading/writing
// the role_permissions table.

import (
	"context"
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

		// EINMAL fuer alle Rollen laden, nicht je Benutzer: Sonst wird aus einer Liste
		// mit zwanzig Eintraegen eine Abfragelawine gegen dieselben vier Rollen.
		rechteJeRolle, err := s.rechteAllerRollen(ctx)
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

			// Die echten Rechte aus role_permissions, nicht mehr eine feste Liste.
			//
			// Hier stand eine hartkodierte Zuordnung, die sich „analog zum Login" nannte,
			// es aber nicht war: Sie nannte manage_settings, print_classes und view_media —
			// Rechte, die es im System gar nicht gibt (kein RequirePermission, kein Seed).
			// Wer die Benutzerliste ansah, bekam also erfundene Angaben, waehrend Login und
			// /api/auth/me die tatsaechlichen Rechte laden. Audit-Befund vom 01.08.2026.
			ur.Permissions = rechteJeRolle[ur.Rolle]
			if ur.Permissions == nil {
				ur.Permissions = []string{}
			}

			responseUsers = append(responseUsers, ur)
		}

		RespondJSON(w, http.StatusOK, responseUsers)
	}
}

// rechteAllerRollen liefert die freigeschalteten Rechte je Rolle aus role_permissions —
// derselben Tabelle, aus der auch Login und RequirePermission lesen.
//
// Admin bekommt "*" wie im Login und im Middleware-Bypass; ein Admin hat implizit alle
// Rechte, unabhaengig davon, was in der Tabelle steht.
//
// Die Rollen kommen kleingeschrieben zurueck, passend zu UserResponse.Rolle.
func (s *Server) rechteAllerRollen(ctx context.Context) (map[string][]string, error) {
	rows, err := s.DB.Pool.Query(ctx, `
		SELECT lower(role), permission
		FROM role_permissions
		WHERE allowed = true
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rechte := map[string][]string{}
	for rows.Next() {
		var rolle, recht string
		if err := rows.Scan(&rolle, &recht); err != nil {
			return nil, err
		}
		rechte[rolle] = append(rechte[rolle], recht)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rechte[strings.ToLower(string(auth.RoleAdmin))] = []string{"*"}
	return rechte, nil
}
