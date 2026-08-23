package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/pkg/httpresp"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
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

	var hasActiveLoans bool
	if err := s.DB.Pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM ausleihen
			WHERE schueler_id = $1 AND rueckgabe_am IS NULL
		)
	`, id).Scan(&hasActiveLoans); err != nil {
		return http.StatusInternalServerError, err
	}
	if hasActiveLoans {
		return http.StatusBadRequest, errors.New("löschen nicht möglich: Schüler hat noch entliehene Bücher")
	}

	var hasUnpaidDamages bool
	if err := s.DB.Pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM schadensfaelle WHERE schueler_id = $1 AND ist_bezahlt = false)
	`, id).Scan(&hasUnpaidDamages); err != nil {
		return http.StatusInternalServerError, err
	}
	if hasUnpaidDamages {
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

// addStrLeerbar ist addStr für Felder, die man LÖSCHEN können muss: Ein leerer Wert
// wird zu NULL, nicht zum leeren String. Das ist dieselbe Schreibweise, die der
// DSGVO-Cron und die LUSD-Ausleitung verwenden (jobs/cron_dsgvo.go, api/lusd_apply.go)
// — sonst stünde für „gelöscht" je nach Weg mal NULL und mal ” in der Spalte.
//
// nil heißt weiterhin „nicht mitgeschickt" und lässt die Spalte in Ruhe.
func (b *updateBuilder) addStrLeerbar(spalte string, wert *string) {
	if wert == nil {
		return
	}
	if strings.TrimSpace(*wert) == "" {
		b.add(spalte, nil)
		return
	}
	b.add(spalte, *wert)
}

func (b *updateBuilder) addInt(spalte string, wert *int) {
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
func (s *Server) PatchStudentHandler(auditRepo repository.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.GetClaims(r.Context())
		if !ok {
			apierrors.SendHTTPError(w, http.StatusUnauthorized, errors.New("Sitzungs-Information fehlt oder ist abgelaufen"))
			return
		}

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
		// lusd_id kontrolliert nachtragen (nur wenn leer, eindeutig) und ggf. in denselben
		// Builder einhängen. nachgetragen != "" heißt: hinterher auditieren.
		nachgetragen, ok := s.pruefeUndSetzeLusdID(ctx, w, id, req.LusdID, b)
		if !ok {
			return
		}
		// Ein zweiter Empty-Check: Wenn AUSSER lusd_id nichts drin war und lusd_id ein
		// No-op ist (gleicher Wert), darf kein leerer UPDATE laufen.
		if len(b.sets) == 0 {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("keine zu aktualisierenden Felder angegeben"))
			return
		}

		if !s.fuehreSchuelerUpdateAus(ctx, w, id, b) {
			return
		}

		if nachgetragen != "" {
			if logErr := auditRepo.LogAdminAktion(ctx, claims.UserID, "LUSD_ID_NACHGETRAGEN", getIP(r), map[string]any{
				"schueler_id": id,
				"lusd_id":     nachgetragen,
			}); logErr != nil {
				log.Printf("Audit für LUSD-ID-Nachtrag fehlgeschlagen: %v", logErr)
			}
		}

		w.Header().Set(headerContentType, contentTypeJSON)
		response := map[string]any{"status": "success"}
		if req.AbgaengerJahr != nil {
			response["abgaenger_jahr"] = *req.AbgaengerJahr
		}
		httpresp.Encode(w, response)
	}
}

// pruefeUndSetzeLusdID trägt die LUSD-ID kontrolliert nach. Die LUSD-ID ist der
// einzige Zuordnungsschlüssel des Landesabgleichs (kein Adress-/Geburtsdatum-Fallback
// im Import); wer sie roh überschreibt, kann still die falsche Identität verknüpfen.
// Regeln: nur NACHTRAGBAR, wenn bisher leer (Waise adoptieren) — ein bestehender Wert
// wird nicht überschrieben und nicht geleert; der neue Wert muss eindeutig sein. Der
// automatische Gegenpart ist das Import-Auto-Matching (Name+Geburtsdatum).
//
// Rückgabe: (nachgetragenerWert, ok). nachgetragenerWert != "" heißt: bitte auditieren.
// Ist req nil oder ein No-op (gleicher Wert), wird ("", true) zurückgegeben.
func (s *Server) pruefeUndSetzeLusdID(ctx context.Context, w http.ResponseWriter, id string, reqLusd *string, b *updateBuilder) (string, bool) {
	if reqLusd == nil {
		return "", true
	}
	neu := strings.TrimSpace(*reqLusd)

	var aktuell string
	if err := s.DB.Pool.QueryRow(ctx, "SELECT COALESCE(lusd_id, '') FROM schueler WHERE id = $1", id).Scan(&aktuell); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("schüler nicht gefunden"))
			return "", false
		}
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return "", false
	}

	if neu == aktuell {
		return "", true // No-op: Formular schickt den unveränderten Wert mit.
	}
	if aktuell != "" {
		// Bestehende Verknüpfung: weder überschreiben noch leeren.
		apierrors.SendHTTPError(w, http.StatusForbidden,
			errors.New("die LUSD-ID dieses Schülers ist bereits gesetzt und kann über das Formular nicht geändert oder geleert werden"))
		return "", false
	}
	if neu == "" {
		return "", true // war leer, bleibt leer.
	}

	// Eindeutigkeit VOR dem Schreiben prüfen, damit statt eines 500 am partiellen
	// Unique-Index (uniq_schueler_lusd_id_active) eine klare 409 zurückkommt.
	var belegt bool
	if err := s.DB.Pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM schueler WHERE lusd_id = $1 AND deleted_at IS NULL AND id <> $2)", neu, id).Scan(&belegt); err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return "", false
	}
	if belegt {
		apierrors.SendHTTPError(w, http.StatusConflict,
			errors.New("diese LUSD-ID ist bereits einem anderen aktiven Schüler zugeordnet"))
		return "", false
	}

	b.addStr("lusd_id", &neu)
	return neu, true
}

// patchStudentRequest bündelt die optional aktualisierbaren Stammdatenfelder (nil = unverändert).
type patchStudentRequest struct {
	Vorname       *string `json:"vorname"`
	Nachname      *string `json:"nachname"`
	Klasse        *string `json:"klasse"`
	LusdID        *string `json:"lusd_id"`
	BarcodeID     *string `json:"barcode_id"`
	AbgaengerJahr *int    `json:"abgaenger_jahr"`
	Geburtsdatum  *string `json:"geburtsdatum"`
	// KEINE Sperrfelder hier. Sperren und Entsperren läuft ausschliesslich über
	// PATCH /api/admin/students/{id}/lock (api/student_lock.go) — und das aus zwei
	// Gründen, die dieser Weg beide nicht erfüllte:
	//
	//   - Der Sperr-Endpunkt verlangt einen GRUND. Über diesen PATCH gesetzt, lief eine
	//     Sperre ohne Grund in chk_schueler_block_reason und kam als 500 „Ein interner
	//     Datenbankfehler ist aufgetreten" zurück (der Sanitizer ersetzt jede 500-Meldung).
	//     Derselbe Vorgang, eine Tür weiter: 400 mit dem Satz, was zu tun ist.
	//   - Beim ENTSPERREN räumt der Sperr-Endpunkt den Grund nur weg, wenn keine
	//     Systemsperre mehr besteht. Dieser PATCH liess ihn schlicht stehen.
	//
	// Zwei Türen zu demselben Zustand, von denen nur eine die Regeln kennt, sind keine
	// Bequemlichkeit — die falsche Tür geht irgendwann auf.
	Strasse     *string `json:"strasse"`
	Hausnummer  *string `json:"hausnummer"`
	Plz         *string `json:"plz"`
	Ort         *string `json:"ort"`
	ElternEmail *string `json:"eltern_email"`
}

// baueSchuelerUpdate erzeugt aus dem PATCH-Request den dynamischen updateBuilder (inkl.
// Klassen→Abgängerjahr-Ableitung und Geburtsdatum-Parsing). ok=false: die Fehlerantwort
// (ungültiges Datum bzw. leerer PATCH) wurde bereits geschrieben.
func baueSchuelerUpdate(w http.ResponseWriter, req *patchStudentRequest) (*updateBuilder, bool) {
	// Die Sperre des Spezialwerts 'lehrer' gilt auch für die zweite Tür (PATCH) —
	// siehe pruefeKlassenname und Migration 072.
	if req.Klasse != nil {
		if err := pruefeKlassenname(*req.Klasse); err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return nil, false
		}
	}
	// Bei Klassenänderung ohne explizites Abgängerjahr dieses automatisch ableiten.
	if req.Klasse != nil && req.AbgaengerJahr == nil {
		newJahr := calculateAbgaengerJahr(*req.Klasse)
		req.AbgaengerJahr = &newJahr
	}

	// Pflichtfelder lassen sich nicht über den PATCH wegräumen.
	//
	// Beim ANLEGEN sind vorname/nachname/klasse `validate:"required"` (student_create.go);
	// hier waren sie es nicht, und ein leerer String kam mit 200 durch — der Schüler
	// verlor Namen, Klasse und Ausweisnummer, die Oberfläche meldete "Änderungen
	// gespeichert". Bei der Klasse kam ein zweiter Schaden dazu: Aus dem leeren Namen
	// leitete calculateAbgaengerJahr noch ein Abgängerjahr ab.
	//
	// Dass das bis zum 23.08.2026 nie passierte, war Zufall und keine Regel: Das
	// Formular schickte geräumte Felder als JSON-null, und null landet im *string als
	// nil ("nicht mitgeschickt"). Dieselbe Zufälligkeit hielt die Löschung der
	// Postanschrift auf — sie kam ebenfalls nie an.
	//
	// Der Barcode steht bewusst mit in der Liste: Beim Anlegen darf er fehlen, dann
	// vergibt ihn das System. Nachträglich entfernen hieße, den Schüler an der Theke
	// unauffindbar zu machen — dafür gibt es keinen Vorgang.
	for _, feld := range []struct {
		bezeichnung string
		wert        *string
	}{
		{"Vorname", req.Vorname},
		{"Nachname", req.Nachname},
		{"Klasse", req.Klasse},
		{"Ausweisnummer", req.BarcodeID},
	} {
		if feld.wert != nil && strings.TrimSpace(*feld.wert) == "" {
			//nolint:staticcheck // ST1005: nutzer-sichtbare Meldung im Formular
			apierrors.SendHTTPError(w, http.StatusBadRequest,
				errors.New(feld.bezeichnung+" darf nicht leer sein."))
			return nil, false
		}
	}

	b := &updateBuilder{}
	b.addStr("vorname", req.Vorname)
	b.addStr("nachname", req.Nachname)
	// lusd_id NICHT hier — sie hat einen eigenen kontrollierten Pfad (nur nachtragbar
	// wenn leer, mit Eindeutigkeits-Prüfung und Audit), siehe pruefeUndSetzeLusdID im
	// Handler. Ein roher Wert im generischen Feld-Beutel verknüpfte den Datensatz sonst
	// ungeprüft mit einer fremden LUSD-Identität (Betreiber-Entscheidung 18.08.2026).
	b.addStr("barcode_id", req.BarcodeID)
	b.addStr("klasse", req.Klasse)
	b.addInt("abgaenger_jahr", req.AbgaengerJahr)

	if req.Geburtsdatum != nil {
		// Pflichtfeld seit 21.08.2026 (Schlüssel des LUSD-Abgleichs): POST verlangt es,
		// PATCH durfte es bis 22.08. mit "" still auf NULL setzen — zweite Tür, gleiche
		// Regel (Prüfung 22.08., B). Leer = 400, nicht blanken.
		if strings.TrimSpace(*req.Geburtsdatum) == "" {
			//nolint:staticcheck // ST1005: nutzer-sichtbare Meldung im Formular
			apierrors.SendHTTPError(w, http.StatusBadRequest,
				errors.New("Geburtsdatum kann nicht geleert werden — es ist der Schlüssel für den LUSD-Abgleich"))
			return nil, false
		}
		parsedDate, err := parseGeburtsdatum(*req.Geburtsdatum)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return nil, false
		}
		b.add("geburtsdatum", parsedDate)
	}

	// Postanschrift & Elternkontakt: nur bei vorhandenem Feld ändern — und ein
	// mitgeschicktes LEERES Feld heißt hier wirklich löschen (NULL). Das sind genau die
	// Angaben, deren Entfernung jemand verlangen kann; ein Weg, der Erfolg meldet und
	// nichts tut, ist dafür der schlechteste Zustand.
	b.addStrLeerbar("strasse", req.Strasse)
	b.addStrLeerbar("hausnummer", req.Hausnummer)
	b.addStrLeerbar("plz", req.Plz)
	b.addStrLeerbar("ort", req.Ort)
	b.addStrLeerbar("eltern_email", req.ElternEmail)

	// Der Empty-PATCH-Check steht bewusst NICHT hier, sondern im Handler NACH
	// pruefeUndSetzeLusdID: Eine reine lusd_id-Nachtragung ist ein gültiger PATCH,
	// dessen einziges Feld erst der kontrollierte lusd_id-Pfad hinzufügt.
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
