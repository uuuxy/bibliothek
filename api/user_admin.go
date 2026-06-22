package api

// user_admin.go — Handlers for system user and role-permission management.
// Covers: listing/creating/updating/deleting staff accounts and reading/writing
// the role_permissions table.

import (
	"net/http"
	"strings"
	"time"

	"bibliothek/apierrors"
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
