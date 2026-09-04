package api

import (
	"errors"
	"log"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/pkg/httpresp"
	"bibliothek/repository"
)

// auditiereBenutzerMutation protokolliert Anlage/Änderung eines Kontos revisionssicher.
//
// Bis zum 16.08.2026 hinterließen Konten-Anlage und Rollenvergabe KEINE Spur — als auf
// Prod vier aktive Admin-Konten auftauchten, war nicht mehr feststellbar, wer sie wann
// angelegt hatte (der Alarm-Mail-Vorfall). Best effort: Ein Audit-Fehler bricht die
// Mutation nicht ab, aber er steht im Log.
func (s *Server) auditiereBenutzerMutation(r *http.Request, aktion string, details map[string]any) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		return
	}
	// Server ohne Datenbank (nackte Testkonstruktion &Server{}): nichts zu schreiben.
	if s.DB == nil || s.DB.Pool == nil {
		return
	}
	if err := repository.NewAuditRepository(s.DB.Pool).
		LogAdminAktion(r.Context(), claims.UserID, aktion, "", details); err != nil {
		log.Printf("Benutzer-Audit (%s) fehlgeschlagen: %v", aktion, err)
	}
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

		// Einen Administrator darf nur ein Administrator anlegen (siehe
		// user_admin_eskalation.go).
		if !pruefeAdminVergabe(w, r, req.Rolle) {
			return
		}

		ctx := r.Context()

		if !pruefeEmailEindeutig(ctx, w, userRepo, req.Email, "") {
			return
		}

		barcode, ok := pruefeBarcodeEindeutig(ctx, w, userRepo, BarcodePruefOptionen{
			BarcodeID:   req.BarcodeID,
			ExcludeID:   "",
			KonfliktMsg: "dieser Barcode wird bereits verwendet",
		})
		if !ok {
			return
		}

		dbEnumRole := normalisiereBenutzerRolle(req.Rolle)

		if _, err := userRepo.CreateUser(ctx, barcode, req.Vorname, req.Nachname, req.Email, dbEnumRole); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		s.auditiereBenutzerMutation(r, "USER_CREATE", map[string]any{
			"email": req.Email, "rolle": dbEnumRole, "vorname": req.Vorname, "nachname": req.Nachname,
		})

		w.Header().Set(headerContentType, contentTypeJSON)
		httpresp.Write(w, []byte(`{"status":"success"}`))
	}
}

// UpdateUserRequest ist der Änderungssatz für ein Mitarbeiterkonto. Warum hier bewusst
// KEIN Passwort steht, erklärt der Kommentar am Ende des Structs.
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

		ctx := r.Context()

		// Reihenfolge ist Absicht: erst die eigene Rolle/Aktivierung schützen, dann die
		// Vergabe der Admin-Rolle, dann den Schutz bestehender Admin-Konten. Alle drei
		// laufen VOR jedem Schreibzugriff (siehe user_admin_eskalation.go).
		if !pruefeSelbstschutz(w, r, id, req.Rolle, req.Aktiv) {
			return
		}
		if !pruefeAdminVergabe(w, r, req.Rolle) {
			return
		}
		if !pruefeAdminZiel(ctx, w, r, userRepo, id) {
			return
		}

		if !pruefeEmailEindeutig(ctx, w, userRepo, req.Email, id) {
			return
		}

		barcode, ok := pruefeBarcodeEindeutig(ctx, w, userRepo, BarcodePruefOptionen{
			BarcodeID:   req.BarcodeID,
			ExcludeID:   id,
			KonfliktMsg: "dieser Barcode wird bereits von einem anderen Benutzer verwendet",
		})
		if !ok {
			return
		}

		dbEnumRole := normalisiereBenutzerRolle(req.Rolle)

		if err := userRepo.UpdateUser(ctx, repository.UpdateUserParams{
			ID: id, Barcode: barcode, Vorname: req.Vorname, Nachname: req.Nachname,
			Email: req.Email, Rolle: dbEnumRole, Aktiv: req.Aktiv,
		}); err != nil {
			// Kein Audit-Eintrag und kein Cache-Invalidate für eine Änderung, die nie
			// stattfand (Phantom-Erfolg-Sweep 31.08.2026).
			if errors.Is(err, repository.ErrBenutzerNichtGefunden) {
				apierrors.SendHTTPError(w, http.StatusNotFound, err)
				return
			}
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		s.auditiereBenutzerMutation(r, "USER_UPDATE", map[string]any{
			"ziel_id": id, "email": req.Email, "rolle": dbEnumRole, "aktiv": req.Aktiv,
		})

		InvalidatePermissionCache()

		w.Header().Set(headerContentType, contentTypeJSON)
		httpresp.Write(w, []byte(`{"status":"success"}`))
	}
}
