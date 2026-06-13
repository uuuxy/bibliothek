package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"bibliothek/apierrors"
)

// PermissionSetting holds a single role-permission flag returned by the API.
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
				ps.Role = strings.ToLower(ps.Role) // Normalize for frontend
				settings = append(settings, ps)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(settings)
	}
}

// UpdatePermissionsRequest is the payload for toggling a single role-permission flag.
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
