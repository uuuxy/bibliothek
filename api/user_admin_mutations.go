package api

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"bibliothek/apierrors"
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
	s.auditiereKonto(r.Context(), aktion, details)
}

// auditiereKonto schreibt einen Eintrag über ein Konto mit der angemeldeten Person als
// Handelnder — für Türen, die nur den Kontext in der Hand haben.
func (s *Server) auditiereKonto(ctx context.Context, aktion string, details map[string]any) {
	s.protokolliereVerwaltung(ctx, aktion, details)
}

// protokolliereKontoAnlage schreibt USER_CREATE — für jede Tür im Handler-Paket, über die ein
// Konto mit einer Person als Handelnder entsteht: Benutzer & Rechte (CreateUserHandler) und die
// Neuanlage einer Lehrkraft in der Leserdatei (legeSchuelerAn). Bis zum 29.09.2026 schrieb nur
// die erste; die Rechenschaft hing an der Tür (docs/OFFEN.md 5.19). ziel_id ist die Kennung des
// Kontos: Daran findet die Auskunft nach Art. 15 den Eintrag, auch nachdem das Konto gelöscht
// ist (repository/dsgvo_konto.go). Bis zum 24.09.2026 trug die Anlage nur die Adresse.
func (s *Server) protokolliereKontoAnlage(ctx context.Context, kontoID, email, rolle, vorname, nachname string) {
	s.auditiereKonto(ctx, "USER_CREATE", map[string]any{
		"ziel_id": kontoID, "email": email, "rolle": rolle, "vorname": vorname, "nachname": nachname,
	})
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
		s.handleCreateUser(w, r, userRepo)
	}
}

// handleCreateUser prüft Rolle, Adresse und Ausweis, legt das Konto an und protokolliert es.
func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request, userRepo repository.UserRepository) {
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
	if !pruefeKeineLeserzeileOhneKonto(ctx, w, userRepo, req.Vorname, req.Nachname) {
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

	// Die Leserzeile — und damit der Platz für Ausweis und Ausleihen — entsteht dabei
	// von selbst (Trigger trg_benutzer_hat_leserzeile, Migration 125).
	kontoID, err := userRepo.CreateUser(ctx, barcode, req.Vorname, req.Nachname, req.Email, dbEnumRole)
	if err != nil {
		if meldeAusweisKollision(w, err) {
			return
		}
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return
	}

	s.protokolliereKontoAnlage(r.Context(), kontoID, req.Email, dbEnumRole, req.Vorname, req.Nachname)

	w.Header().Set(headerContentType, contentTypeJSON)
	httpresp.Write(w, []byte(`{"status":"success"}`))
}

// UpdateUserRequest nennt, was an einem Konto geändert wird. Ein fehlendes Feld bleibt, wie es
// ist: Die Maske schickt nur, was sie seit dem Öffnen geändert hat, damit sie nichts
// überschreibt, was ein anderer Platz inzwischen gespeichert hat. Ein Passwort gibt es hier
// nicht; Anmeldungen laufen über den Schul-Mailserver (IMAP).
type UpdateUserRequest struct {
	BarcodeID *string `json:"barcode_id"`
	Vorname   *string `json:"vorname"`
	Nachname  *string `json:"nachname"`
	Email     *string `json:"email" validate:"omitempty,email"`
	Rolle     *string `json:"rolle"`
	Aktiv     *bool   `json:"aktiv"`
}

// UpdateUserHandler ändert Name, E-Mail, Ausweisnummer, Rolle oder „aktiv" eines Kontos.
// @Summary      Update system user
// @Description  Ändert an einem Konto die Felder, die der Rumpf nennt; ein fehlendes Feld bleibt, wie es ist.
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
		s.handleUpdateUser(w, r, userRepo)
	}
}

// handleUpdateUser prüft die Schutzregeln vor jedem Schreibzugriff, ändert das Konto und
// protokolliert es.
func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request, userRepo repository.UserRepository) {
	id := r.PathValue("id")
	if id == "" {
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing user ID parameter"))
		return
	}

	var req UpdateUserRequest
	if !DecodeAndValidate(w, r, &req) {
		return
	}
	if !pruefeGenanntePflichtfelder(w, req) {
		return
	}

	ctx := r.Context()
	if req.nenntNichts() {
		antworteOhneKontoAenderung(ctx, w, userRepo, id)
		return
	}

	// Reihenfolge ist Absicht: erst die eigene Rolle/Aktivierung schützen, dann die
	// Vergabe der Admin-Rolle, dann den Schutz bestehender Admin-Konten. Alle drei
	// laufen VOR jedem Schreibzugriff (siehe user_admin_eskalation.go).
	if !pruefeSelbstschutz(w, r, id, req.Rolle, req.Aktiv) {
		return
	}
	if req.Rolle != nil && !pruefeAdminVergabe(w, r, *req.Rolle) {
		return
	}
	if !pruefeAdminZiel(ctx, w, r, userRepo, id) {
		return
	}

	aenderung, ok := kontoAenderungAus(ctx, w, userRepo, id, req)
	if !ok {
		return
	}
	stand, err := userRepo.UpdateUser(ctx, aenderung)
	if err != nil {
		// Kein Protokolleintrag und kein Cache-Invalidate für eine Änderung, die nie
		// stattfand.
		antworteAufKontoAenderungsfehler(w, err)
		return
	}

	// Der Eintrag nennt den Stand nach der Änderung, auch für Felder, die der Rumpf nicht nannte.
	s.auditiereBenutzerMutation(r, "USER_UPDATE", map[string]any{
		"ziel_id": id, "email": stand.Email, "rolle": stand.Rolle, "aktiv": stand.Aktiv,
	})

	InvalidatePermissionCache()

	w.Header().Set(headerContentType, contentTypeJSON)
	httpresp.Write(w, []byte(`{"status":"success"}`))
}

// nenntNichts sagt, ob der Rumpf kein Feld des Kontos nennt.
func (req UpdateUserRequest) nenntNichts() bool {
	return req.BarcodeID == nil && req.Vorname == nil && req.Nachname == nil &&
		req.Email == nil && req.Rolle == nil && req.Aktiv == nil
}

// pruefeGenanntePflichtfelder lehnt ein genanntes, aber leeres Pflichtfeld ab: Vorname,
// Nachname, E-Mail und Rolle lassen sich ändern, nicht leeren.
func pruefeGenanntePflichtfelder(w http.ResponseWriter, req UpdateUserRequest) bool {
	for _, wert := range []*string{req.Vorname, req.Nachname, req.Email, req.Rolle} {
		if wert != nil && strings.TrimSpace(*wert) == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest,
				errors.New("vorname, Nachname, E-Mail und Rolle dürfen nicht leer sein"))
			return false
		}
	}
	return true
}

// antworteOhneKontoAenderung beantwortet einen Rumpf, der nichts nennt: nichts geschrieben,
// kein Protokolleintrag. Ein unbekanntes Konto bleibt eine 404.
func antworteOhneKontoAenderung(ctx context.Context, w http.ResponseWriter, userRepo repository.UserRepository, id string) {
	rolle, err := userRepo.GetRolleByID(ctx, id)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return
	}
	if rolle == "" {
		apierrors.SendHTTPError(w, http.StatusNotFound, repository.ErrBenutzerNichtGefunden)
		return
	}
	w.Header().Set(headerContentType, contentTypeJSON)
	httpresp.Write(w, []byte(`{"status":"success"}`))
}

// kontoAenderungAus prüft die genannte E-Mail und Ausweisnummer auf Eindeutigkeit und baut die
// Änderung für das Repository. ok=false: Die Ablehnung ist schon beantwortet.
func kontoAenderungAus(ctx context.Context, w http.ResponseWriter, userRepo repository.UserRepository, id string, req UpdateUserRequest) (repository.UpdateUserParams, bool) {
	aenderung := repository.UpdateUserParams{
		ID: id, Vorname: req.Vorname, Nachname: req.Nachname, Email: req.Email, Aktiv: req.Aktiv,
	}
	if req.Email != nil && !pruefeEmailEindeutig(ctx, w, userRepo, *req.Email, id) {
		return aenderung, false
	}
	if req.BarcodeID != nil {
		barcode, ok := pruefeBarcodeEindeutig(ctx, w, userRepo, BarcodePruefOptionen{
			BarcodeID:   *req.BarcodeID,
			ExcludeID:   id,
			KonfliktMsg: "dieser Barcode wird bereits von einem anderen Benutzer verwendet",
		})
		if !ok {
			return aenderung, false
		}
		aenderung.BarcodeGenannt, aenderung.Barcode = true, barcode
	}
	if req.Rolle != nil {
		rolle := normalisiereBenutzerRolle(*req.Rolle)
		aenderung.Rolle = &rolle
	}
	return aenderung, true
}

// antworteAufKontoAenderungsfehler ordnet den Fehler beim Ändern eines Kontos ein: unbekannte
// Kennung 404, belegter Ausweis 409, alles andere 500.
func antworteAufKontoAenderungsfehler(w http.ResponseWriter, err error) {
	if errors.Is(err, repository.ErrBenutzerNichtGefunden) {
		apierrors.SendHTTPError(w, http.StatusNotFound, err)
		return
	}
	if meldeAusweisKollision(w, err) {
		return
	}
	apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
}
