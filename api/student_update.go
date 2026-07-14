package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/pkg/httpresp"
	"bibliothek/repository"
)

// pruefeSchuelerLoeschbar prüft, ob ein Schüler gelöscht werden darf. Rückgabe
// (0, nil) bedeutet löschbar; andernfalls der passende HTTP-Status samt Fehler.
func (s *Server) pruefeSchuelerLoeschbar(ctx context.Context, id string) (int, error) {
	var studentExists bool
	if err := s.DB.Pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schueler WHERE id = $1)", id).Scan(&studentExists); err != nil {
		return http.StatusInternalServerError, err
	}
	if !studentExists {
		return http.StatusNotFound, errors.New("schüler nicht gefunden")
	}

	var activeLoansCount int
	if err := s.DB.Pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM ausleihen
		WHERE schueler_id = $1 AND rueckgabe_am IS NULL
	`, id).Scan(&activeLoansCount); err != nil {
		return http.StatusInternalServerError, err
	}
	if activeLoansCount > 0 {
		return http.StatusBadRequest, errors.New("löschen nicht möglich: Schüler hat noch entliehene Bücher")
	}

	var unpaidDamagesCount int
	if err := s.DB.Pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM schadensfaelle
		WHERE schueler_id = $1 AND ist_bezahlt = false
	`, id).Scan(&unpaidDamagesCount); err != nil {
		return http.StatusInternalServerError, err
	}
	if unpaidDamagesCount > 0 {
		return http.StatusBadRequest, errors.New("löschen nicht möglich: Schüler hat noch unbezahlte Schadensfälle/Gebühren")
	}

	return 0, nil
}

// DeleteStudentHandler deletes a student after checking for outstanding loans and unpaid damage cases, logging it to the audit trail.
// @Summary      Delete student
// @Description  Transactionally deletes a student from the system, checks for active loans or unpaid damage fees, anonymizes historical loans, and writes to audit_log.
// @Tags         students
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Student ID (UUID)"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /schueler/{id} [delete]
func (s *Server) DeleteStudentHandler(auditRepo repository.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.GetClaims(r.Context())
		if !ok {
			apierrors.SendHTTPError(w, http.StatusUnauthorized, errors.New("missing session information"))
			return
		}

		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("fehlende Schüler-ID"))
			return
		}

		ctx := r.Context()

		if status, err := s.pruefeSchuelerLoeschbar(ctx, id); err != nil {
			apierrors.SendHTTPError(w, status, err)
			return
		}

		// Transaktionales Löschen mit Audit-Log
		if err := auditRepo.DeleteStudent(ctx, id, claims.UserID, "Manuelle Löschung"); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// Admin audit log
		details := fmt.Sprintf(`{"student_id":"%s"}`, id)
		logExec(s.DB.Pool.Exec(ctx, "INSERT INTO audit_logs (admin_id, aktion, details, ip_adresse) VALUES ($1, $2, $3::jsonb, $4)", claims.UserID, "DELETE_STUDENT", details, getIP(r)))

		RespondJSON(w, http.StatusOK, map[string]any{
			"status": "success",
		})
	}
}

// updateBuilder sammelt optionale SET-Zuweisungen für ein dynamisches UPDATE.
type updateBuilder struct {
	sets []string
	args []interface{}
}

func (b *updateBuilder) add(spalte string, wert interface{}) {
	b.sets = append(b.sets, spalte)
	b.args = append(b.args, wert)
}

func (b *updateBuilder) addStr(spalte string, wert *string) {
	if wert != nil {
		b.add(spalte, *wert)
	}
}

func (b *updateBuilder) addInt(spalte string, wert *int) {
	if wert != nil {
		b.add(spalte, *wert)
	}
}

func (b *updateBuilder) addBool(spalte string, wert *bool) {
	if wert != nil {
		b.add(spalte, *wert)
	}
}

// build hängt die gesammelten SET-Zuweisungen (nummeriert ab $1) und die
// WHERE-Bedingung an prefix an und liefert Query samt Argumentliste.
func (b *updateBuilder) build(prefix, idValue string) (string, []interface{}) {
	query := prefix
	args := make([]interface{}, 0, len(b.args)+1)
	for i, spalte := range b.sets {
		query += fmt.Sprintf(", %s = $%d", spalte, i+1)
		args = append(args, b.args[i])
	}
	query += fmt.Sprintf(" WHERE id = $%d", len(b.sets)+1)
	args = append(args, idValue)
	return query, args
}

// parseGeburtsdatum parst ein optionales ISO-Datum. Leerstring ergibt (nil, nil)
// und setzt das Feld damit auf NULL.
func parseGeburtsdatum(raw string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	t, err := time.Parse(dateFormatISO, raw)
	if err != nil {
		return nil, fmt.Errorf("ungültiges Datumsformat für Geburtsdatum: %q — erwartet YYYY-MM-DD", raw)
	}
	return &t, nil
}

// PatchStudentHandler aktualisiert editierbare Felder eines Schülers (klasse, abgaenger_jahr).
// Wird nun auch für das Bearbeiten aller Stammdaten in der UI genutzt.
func (s *Server) PatchStudentHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("fehlende Schüler-ID"))
			return
		}

		var req patchStudentRequest
		if !DecodeAndValidate(w, r, &req) {
			return
		}

		b, ok := baueSchuelerUpdate(w, &req)
		if !ok {
			return
		}

		ctx := r.Context()
		if !s.fuehreSchuelerUpdateAus(ctx, w, id, b) {
			return
		}

		w.Header().Set(headerContentType, contentTypeJSON)
		response := map[string]any{"status": "success"}
		if req.AbgaengerJahr != nil {
			response["abgaenger_jahr"] = *req.AbgaengerJahr
		}
		httpresp.Encode(w, response)
	}
}

// patchStudentRequest bündelt die optional aktualisierbaren Stammdatenfelder (nil = unverändert).
type patchStudentRequest struct {
	Vorname           *string `json:"vorname"`
	Nachname          *string `json:"nachname"`
	Klasse            *string `json:"klasse"`
	LusdID            *string `json:"lusd_id"`
	BarcodeID         *string `json:"barcode_id"`
	AbgaengerJahr     *int    `json:"abgaenger_jahr"`
	Geburtsdatum      *string `json:"geburtsdatum"`
	IsManuallyBlocked *bool   `json:"is_manually_blocked"`
	BlockReason       *string `json:"block_reason"`
	Strasse           *string `json:"strasse"`
	Hausnummer        *string `json:"hausnummer"`
	Plz               *string `json:"plz"`
	Ort               *string `json:"ort"`
	ElternEmail       *string `json:"eltern_email"`
}

// baueSchuelerUpdate erzeugt aus dem PATCH-Request den dynamischen updateBuilder (inkl.
// Klassen→Abgängerjahr-Ableitung und Geburtsdatum-Parsing). ok=false: die Fehlerantwort
// (ungültiges Datum bzw. leerer PATCH) wurde bereits geschrieben.
func baueSchuelerUpdate(w http.ResponseWriter, req *patchStudentRequest) (*updateBuilder, bool) {
	// Bei Klassenänderung ohne explizites Abgängerjahr dieses automatisch ableiten.
	if req.Klasse != nil && req.AbgaengerJahr == nil {
		newJahr := calculateAbgaengerJahr(*req.Klasse)
		req.AbgaengerJahr = &newJahr
	}

	b := &updateBuilder{}
	b.addStr("vorname", req.Vorname)
	b.addStr("nachname", req.Nachname)
	b.addStr("lusd_id", req.LusdID)
	b.addStr("barcode_id", req.BarcodeID)
	b.addStr("klasse", req.Klasse)
	b.addInt("abgaenger_jahr", req.AbgaengerJahr)

	if req.Geburtsdatum != nil {
		parsedDate, err := parseGeburtsdatum(*req.Geburtsdatum)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return nil, false
		}
		b.add("geburtsdatum", parsedDate)
	}

	b.addBool("is_manually_blocked", req.IsManuallyBlocked)
	b.addStr("block_reason", req.BlockReason)
	// Postanschrift & Elternkontakt (Stammdaten): nur bei vorhandenem Feld ändern.
	b.addStr("strasse", req.Strasse)
	b.addStr("hausnummer", req.Hausnummer)
	b.addStr("plz", req.Plz)
	b.addStr("ort", req.Ort)
	b.addStr("eltern_email", req.ElternEmail)

	// Empty PATCH (kein aktualisierbares Feld): als 400 ablehnen, statt einen
	// No-op-UPDATE zu fahren, dessen RowsAffected==0 fälschlich als 404 gälte.
	if len(b.sets) == 0 {
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("keine zu aktualisierenden Felder angegeben"))
		return nil, false
	}
	return b, true
}

// fuehreSchuelerUpdateAus baut das dynamische UPDATE und führt es aus. ok=false: die
// Fehlerantwort (500 bzw. 404 bei unbekanntem Schüler) wurde bereits geschrieben.
func (s *Server) fuehreSchuelerUpdateAus(ctx context.Context, w http.ResponseWriter, id string, b *updateBuilder) bool {
	query, args := b.build("UPDATE schueler SET aktualisiert_am = CURRENT_TIMESTAMP", id)
	tag, err := s.DB.Pool.Exec(ctx, query, args...)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return false
	}
	if tag.RowsAffected() == 0 {
		apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("schüler nicht gefunden"))
		return false
	}
	return true
}
