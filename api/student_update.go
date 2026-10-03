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
	// `leser` statt der Sicht `schueler`: Die zeigt nur Schüler, und ein Kollege kam hier
	// als „nicht gefunden" zurück, bevor auch nur eine Regel geprüft war.
	var studentExists bool
	if err := s.DB.Pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM leser WHERE id = $1)", id).Scan(&studentExists); err != nil {
		return http.StatusInternalServerError, err
	}
	if !studentExists {
		return http.StatusNotFound, errors.New("leser nicht gefunden")
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
		return http.StatusBadRequest, errors.New("löschen nicht möglich: Dieser Leser hat noch entliehene Bücher")
	}

	var hasUnpaidDamages bool
	if err := s.DB.Pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM schadensfaelle WHERE schueler_id = $1 AND ist_bezahlt = false)
	`, id).Scan(&hasUnpaidDamages); err != nil {
		return http.StatusInternalServerError, err
	}
	if hasUnpaidDamages {
		return http.StatusBadRequest, errors.New("löschen nicht möglich: Dieser Leser hat noch unbezahlte Schadensfälle/Gebühren")
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

		// Niemand löscht seine eigene Leserzeile. Beim Kollegium fiele damit auch das
		// eigene Konto (DeleteStudent) — der Löschende spielte sich mitten im Vorgang
		// selbst aus der Anmeldung. Die Benutzerverwaltung hält dieselbe Regel für Konten
		// (DeleteUserHandler, „eigenes Konto kann nicht gelöscht werden").
		eigene, err := repository.LeserIDVonKonto(ctx, s.DB.Pool, claims.UserID)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if eigene != "" && eigene == id {
			apierrors.SendHTTPError(w, http.StatusForbidden,
				//nolint:staticcheck // ST1005: nutzer-sichtbare Meldung in der Akte
				errors.New("Der eigene Eintrag lässt sich nicht löschen — mit ihm fiele der eigene Zugang."))
			return
		}

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
		// Schlüssel `schueler_id` wie überall: Auskunft und Tilgung fragen genau diesen
		// Schlüssel ab (Migration 091 hat die alte `student_id`-Form umgeschlüsselt).
		details := fmt.Sprintf(`{"schueler_id":"%s"}`, id)
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
		s.handlePatchStudent(w, r, auditRepo)
	}
}

// handlePatchStudent prüft die Änderung, schreibt sie und trägt nach, was erst nach dem
// Schreiben entstehen darf.
func (s *Server) handlePatchStudent(w http.ResponseWriter, r *http.Request, auditRepo repository.AuditRepository) {
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

	ctx := r.Context()
	aenderung, ok := s.pruefeSchuelerAenderung(ctx, w, id, &req)
	if !ok {
		return
	}
	if !s.fuehreSchuelerUpdateAus(ctx, w, id, aenderung.update) {
		return
	}
	if !s.trageNachUndProtokolliere(w, r, auditRepo, claims, id, aenderung) {
		return
	}

	w.Header().Set(headerContentType, contentTypeJSON)
	response := map[string]any{"status": "success"}
	if req.AbgaengerJahr != nil {
		response["abgaenger_jahr"] = *req.AbgaengerJahr
	}
	httpresp.Encode(w, response)
}

// schuelerAenderung ist eine geprüfte Änderung an der Leserzeile samt dem, was nach dem
// Schreiben noch aussteht.
type schuelerAenderung struct {
	update *updateBuilder
	// lusdNachgetragen ist die LUSD-ID, die diese Änderung einträgt; sie gehört ins Protokoll.
	lusdNachgetragen string
	// kontoAdresse ist die Schul-E-Mail, für die nach dem Schreiben ein Konto entsteht.
	kontoAdresse string
}

// pruefeSchuelerAenderung baut die Zuweisungen aus der Anfrage und prüft jedes Feld, das
// eigene Regeln hat. false heißt: Die Ablehnung ist schon beantwortet.
func (s *Server) pruefeSchuelerAenderung(ctx context.Context, w http.ResponseWriter, id string, req *patchStudentRequest) (schuelerAenderung, bool) {
	var keine schuelerAenderung
	b, ok := baueSchuelerUpdate(w, req)
	if !ok {
		return keine, false
	}
	// Die LUSD-ID lässt sich nur nachtragen: wenn sie bisher leer war und der neue Wert
	// eindeutig ist.
	lusdNachgetragen, ok := s.pruefeUndSetzeLusdID(ctx, w, id, req.LusdID, b)
	if !ok {
		return keine, false
	}
	if !s.pruefeUndSetzeArt(ctx, w, id, req.Art, b) {
		return keine, false
	}
	// Die Schul-E-Mail wird vor dem Schreiben geprüft und nach ihm eingetragen: Das Konto
	// soll nicht an einer Leserzeile hängen, deren Änderung gescheitert ist. Eine Adresse
	// kommt nur zurück, wenn es noch kein Konto gibt und sie gültig ist.
	kontoAdresse, ok := s.pruefeSchulEmail(ctx, w, id, req.Email, req.Art)
	if !ok {
		return keine, false
	}
	if !s.pruefeAusweisLeerung(ctx, w, id, req.BarcodeID) {
		return keine, false
	}
	// Ohne Zuweisung läuft kein UPDATE, etwa wenn nur die unveränderte LUSD-ID mitkam. Eine
	// nachzutragende Adresse ist Arbeit, auch wenn an der Leserzeile nichts steht; sonst
	// wäre „nur die Schul-E-Mail nachtragen" ein 400.
	if len(b.sets) == 0 && kontoAdresse == "" {
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("keine zu aktualisierenden Felder angegeben"))
		return keine, false
	}
	return schuelerAenderung{update: b, lusdNachgetragen: lusdNachgetragen, kontoAdresse: kontoAdresse}, true
}

// trageNachUndProtokolliere legt nach dem Schreiben das Konto zur Schul-E-Mail an und
// schreibt die Protokolleinträge der Nachträge. false heißt: Die Ablehnung ist schon
// beantwortet.
func (s *Server) trageNachUndProtokolliere(w http.ResponseWriter, r *http.Request, auditRepo repository.AuditRepository, claims *auth.Claims, id string, a schuelerAenderung) bool {
	ctx := r.Context()
	if a.kontoAdresse != "" {
		// Das Konto entsteht immer, aktiv nur bei manage_users — dieselbe Paarung wie beim
		// Anlegen (student_create.go). Wer hier nur edit_students hat, bekäme sonst still
		// das Recht, Zugänge freizuschalten.
		if !s.trageKontoNach(ctx, w, id, a.kontoAdresse, s.BesitztRecht(r, "manage_users")) {
			return false
		}
		if logErr := auditRepo.LogAdminAktion(ctx, claims.UserID, "KOLLEGIUMSKONTO_NACHGETRAGEN", getIP(r), map[string]any{
			"schueler_id": id,
		}); logErr != nil {
			log.Printf("Audit für Kontonachtrag fehlgeschlagen: %v", logErr)
		}
	}

	if a.lusdNachgetragen != "" {
		if logErr := auditRepo.LogAdminAktion(ctx, claims.UserID, "LUSD_ID_NACHGETRAGEN", getIP(r), map[string]any{
			"schueler_id": id,
			"lusd_id":     a.lusdNachgetragen,
		}); logErr != nil {
			log.Printf("Audit für LUSD-ID-Nachtrag fehlgeschlagen: %v", logErr)
		}
	}
	return true
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
	if err := s.DB.Pool.QueryRow(ctx, "SELECT COALESCE(lusd_id, '') FROM leser WHERE id = $1", id).Scan(&aktuell); err != nil {
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

// pruefeUndSetzeArt aendert die Art eines Lesers (Migration 123). Erlaubt ist der Wechsel
// innerhalb des Kollegiums: Lehrkraft, LiV, Praktikum, Sekretariat, U-plus, Fachbereich
// (Migration 153). Beide Richtungen ueber die Grenze zum Schueler sind zu — und zwar nicht
// aus Vorsicht, sondern weil jede von ihnen einen echten Schaden anrichtet:
//
//	Schueler -> Kollege: Die Zeile verliert damit ihre LUSD-Bindung. Traegt sie eine
//	  lusd_id, bricht chk_leser_nur_schueler_werden_abgaenger; traegt sie keine, waere
//	  der Schueler beim naechsten Import ein unbekannter Name und liefe als Abgaenger
//	  samt Anonymisierung durch. Absprache vom 16.09.2026: "ein Schueler kann nie ein Lehrer
//	  werden!"
//	Kollege -> Schueler: chk_leser_schueler_pflichtfelder verlangt Klasse, Abgaengerjahr
//	  UND Ausweisnummer. Ein Kollege hat die ersten beiden nicht; das UPDATE liefe in
//	  den CHECK und damit in eine 500.
//
// Ein Mensch wechselt die Seite nicht. Wer wirklich falsch angelegt wurde, wird
// geloescht und neu angelegt — ein sichtbarer Vorgang statt einer stillen Umwidmung.
//
// nil heisst "nicht mitgeschickt"; derselbe Wert ist ein No-op, damit das Formular
// die Art unveraendert mitschicken darf.
//
// Ein bestehendes Konto bleibt beim Wechsel stehen, auch zu Praktikum oder Fachbereich: Neu
// angelegt wird dort keines (repository.ArtMitKonto), ueber ein vorhandenes entscheidet die
// Benutzerverwaltung.
func (s *Server) pruefeUndSetzeArt(ctx context.Context, w http.ResponseWriter, id string, reqArt *string, b *updateBuilder) bool {
	if reqArt == nil {
		return true
	}
	neu := strings.TrimSpace(*reqArt)
	// Dieselbe Menge wie beim Anlegen (leserArten, api/leser_art.go) und dieselbe wie
	// chk_leser_art in der Datenbank. Eine unbekannte Art ist ein Tippfehler, kein neuer
	// Personenkreis — und soll als Auskunft zurueckkommen, nicht als CHECK-500.
	if !leserArten[neu] {
		//nolint:staticcheck // ST1005: nutzer-sichtbare Meldung im Formular
		apierrors.SendHTTPError(w, http.StatusBadRequest,
			fmt.Errorf("Unbekannte Art %q. Möglich sind: %s.", neu, moeglicheArten()))
		return false
	}

	var aktuell string
	// `leser` und nicht die Sicht `schueler`: Die Sicht ist auf art='schueler'
	// eingeschraenkt (schema.sql), ein Kollege steht schlicht nicht darin.
	if err := s.DB.Pool.QueryRow(ctx, "SELECT art FROM leser WHERE id = $1", id).Scan(&aktuell); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("leser nicht gefunden"))
			return false
		}
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return false
	}

	if neu == aktuell {
		return true // No-op: Formular schickt den unveraenderten Wert mit.
	}
	if istSchuelerArt(aktuell) || istSchuelerArt(neu) {
		//nolint:staticcheck // ST1005: nutzer-sichtbare Meldung im Formular
		apierrors.SendHTTPError(w, http.StatusBadRequest,
			errors.New("Aus einem Schüler wird kein Kollege und umgekehrt. "+
				"Lege die Person neu an und lösche den falschen Eintrag."))
		return false
	}

	b.addStr("art", &neu)
	return true
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
	// Art des Lesers (leserArten, Migration 123 und 153). Sie steht hier, weil die Akte
	// sie zeigen und ein Kollege innerhalb des Kollegiums wechseln koennen muss. Sie hat einen eigenen kontrollierten Pfad (pruefeUndSetzeArt) und liegt
	// NICHT im generischen Feld-Beutel: Ein Wechsel der Art verschiebt die Zeile
	// zwischen zwei Pflichtfeld-Welten (chk_leser_schueler_pflichtfelder,
	// chk_leser_nur_schueler_werden_abgaenger) und liefe roh in einen CHECK — also in
	// eine 500, die der Sanitizer zu "interner Datenbankfehler" macht.
	Art *string `json:"art"`
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
	// Email ist die SCHUL-Adresse eines Kollegen mit Zugang und steht nicht in `leser`,
	// sondern am Konto (benutzer.email). Sie liegt deshalb ebenfalls nicht im
	// generischen Feld-Beutel, sondern hat ihren eigenen Pfad (pruefeSchulEmail,
	// api/student_schul_email.go): nachtragbar, solange keine da ist — dann entsteht
	// das Konto; steht eine da, ist sie eine Anzeige und wird in der
	// Benutzerverwaltung geändert.
	Email *string `json:"email"`
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
	// Die Ausweisnummer steht NICHT mehr in dieser Liste: Ihre Pflicht ist an die Art
	// gepaart, wie in der Datenbank (chk_leser_schueler_pflichtfelder), und die Art steht
	// hier nicht fest. Ein Schüler ohne Nummer ist an der Theke unauffindbar; ein Kollege
	// muss eine falsch eingetragene wieder loswerden können — mit aktivem Konto bekommt er
	// dabei eine neue aus dem Generator (Migration 145). Geprüft wird sie deshalb im Handler
	// an der wirklichen Art (pruefeAusweisLeerung).
	for _, feld := range []struct {
		bezeichnung string
		wert        *string
	}{
		{"Vorname", req.Vorname},
		{"Nachname", req.Nachname},
		{"Klasse", req.Klasse},
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
	// Leerbar, nicht addStr: Eine geräumte Ausweisnummer gehört als NULL in die Spalte.
	// Der leere String wäre ein Wert — und `uniq_schueler_barcode_active` ließe genau
	// einen zweiten Leser mit „" nicht zu, der dritte scheiterte an einer Kollision mit
	// einer Nummer, die niemand hat.
	b.addStrLeerbar("barcode_id", req.BarcodeID)
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
	// Geschrieben wird auf die TABELLE `leser`, nicht auf die Sicht `schueler`.
	//
	// Die Sicht ist `SELECT * FROM leser WHERE art = 'schueler' WITH CHECK OPTION`
	// (Migration 123/schema.sql). Sie ist ein Schutz und bleibt einer: Jede Abfrage, die
	// „Schueler" meint, meint durch sie auch wirklich Schueler. Fuer den Aenderungspfad
	// der Akte ist sie aber die falsche Tuer, seit die Leserdatei alle fuehrt — ein
	// Kollege steht NICHT in ihr, das UPDATE traf null Zeilen, und der Handler
	// antwortete 404 „schueler nicht gefunden". Genau das war der Befund am
	// 16.09.2026: „ich kann dort aber keine adressedaten etc nachtragen." Es fehlte
	// nicht nur der Knopf in der Akte — der Server haette ihn ohnehin abgewiesen.
	//
	// Was die Sicht hier verhindert hat, verhindert jetzt pruefeUndSetzeArt mit einer
	// Begruendung statt mit einer 404, und die Paarungs-CHECKs der Tabelle
	// (chk_leser_schueler_pflichtfelder, chk_leser_nur_schueler_werden_abgaenger) liegen
	// unveraendert darunter. Die Zusage ist dieselbe, nur das Mittel hat gewechselt.
	query, args := b.build("UPDATE leser SET aktualisiert_am = CURRENT_TIMESTAMP", id)
	tag, err := s.DB.Pool.Exec(ctx, query, args...)
	if err != nil {
		// Eine vergebene Ausweisnummer (unter den Schülern oder im Kollegium, Migration 118) ist
		// eine Auskunft, kein Serverfehler.
		if repository.IstAusweisKollision(err) {
			apierrors.SendHTTPError(w, http.StatusConflict,
				errors.New("diese Ausweisnummer trägt bereits eine andere Person (Schüler oder Kollegium)"))
			return false
		}
		if repository.IstNummerBuchOderAusweisKollision(err) {
			apierrors.SendHTTPError(w, http.StatusConflict,
				errors.New("diese Nummer ist der Barcode eines Buchs und kann kein Ausweis sein"))
			return false
		}
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return false
	}
	if tag.RowsAffected() == 0 {
		apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("leser nicht gefunden"))
		return false
	}
	return true
}
